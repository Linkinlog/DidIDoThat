package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	LayoutHourly = "2006-01-02T15Z07:00"
	Layout       = "2006-01-02Z07:00"
	previewLimit = 30
)

func startHTTP(port int, conn *pgxpool.Pool) {
	mux := http.NewServeMux()

	// unauthorized
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		if err := conn.Ping(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("POST /api/auth/login", handleAuth(conn))
	mux.HandleFunc("GET /api/auth/logout", handleLogout())
	mux.HandleFunc("GET /api/auth/magic/{magicToken}", handleMagic(conn))

	// authorized
	mux.HandleFunc("GET /api/tasks", withUser(conn, handleGetTasks(conn, previewLimit)))
	mux.HandleFunc("POST /api/tasks", withUser(conn, handleCreateTask(conn)))
	mux.HandleFunc("POST /api/tasks/{taskId}/complete", handleCompleteTask(conn))
	mux.HandleFunc("GET /api/auth/session", withUser(conn, handleSession()))
	mux.HandleFunc("GET /api/auth/qr", withUser(conn, handleQR(conn)))

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
		panic(err)
	}
}

func withUser(conn *pgxpool.Pool, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		session := getSession(r.Context(), conn, cookie.Value)
		if session.ID == 0 {
			http.Error(w, "session not found", http.StatusUnauthorized)
			return
		}

		user := getUserByID(r.Context(), conn, session.UserID)
		if user.ID == 0 {
			http.Error(w, "user not found", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserKey("user"), user)

		h(w, r.WithContext(ctx))
	}
}

func handleLogout() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   isProduction(),
			SameSite: http.SameSiteStrictMode,
			MaxAge:   60 * 60,
		})
	}
}

func handleSession() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(UserKey("user")).(User)
		if user.ID == 0 {
			http.Error(w, "user not found", http.StatusUnauthorized)
			return
		}

		userResp := struct {
			Username string `json:"username"`
		}{
			Username: user.Username,
		}

		if err := json.NewEncoder(w).Encode(userResp); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func handleMagic(conn *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		magicToken := r.PathValue("magicToken")
		if magicToken == "" {
			http.Error(w, "magic_token is required", http.StatusBadRequest)
			return
		}

		magicLink := getMagicLinkByToken(r.Context(), conn, magicToken)
		if magicLink.ID == 0 {
			http.Error(w, "magic link not found", http.StatusNotFound)
			return
		}

		user := getUserByID(r.Context(), conn, magicLink.UserID)
		if user.ID == 0 {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		token := newToken()

		insertSession(r.Context(), conn, user.ID, token)

		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   isProduction(),
			SameSite: http.SameSiteStrictMode,
			MaxAge:   60 * 60,
		})

		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func handleQR(conn *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(UserKey("user")).(User)
		if user.ID == 0 {
			http.Error(w, "user not found", http.StatusUnauthorized)
			return
		}

		magicLink := getMagicLink(r.Context(), conn, user.ID)
		if magicLink.ID != 0 {
			w.Write([]byte(magicLink.Token))
			return
		}

		token := newToken()
		insertMagicLink(r.Context(), conn, token, user.ID)

		w.Write([]byte(token))
	}
}

func handleAuth(conn *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := decoder.Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		username := body.Username
		password := body.Password

		if username == "" || password == "" {
			http.Error(w, "username and password are required", http.StatusBadRequest)
			return
		}

		user := getUser(r.Context(), conn, username)
		if user.ID == 0 {
			user = insertUser(r.Context(), conn, username, password)
		} else {
			if !comparePassword(r.Context(), conn, username, password) {
				http.Error(w, "invalid username or password", http.StatusUnauthorized)
				return
			}
		}

		token := newToken()

		insertSession(r.Context(), conn, user.ID, token)

		http.SetCookie(w, &http.Cookie{
			Name:     "session_token",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   isProduction(),
			SameSite: http.SameSiteStrictMode,
			MaxAge:   60 * 60,
		})

		http.Redirect(w, r, "/", http.StatusFound)
	}
}

func handleCompleteTask(conn *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		taskIDStr := r.PathValue("taskId")
		if taskIDStr == "" {
			http.Error(w, "task_id is required", http.StatusBadRequest)
			return
		}

		taskID, err := strconv.Atoi(taskIDStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		completeTask(r.Context(), conn, taskID)
	}
}

func handleGetTasks(conn *pgxpool.Pool, limit int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(UserKey("user")).(User)
		if user.ID == 0 {
			http.Error(w, "user not found", http.StatusUnauthorized)
			return
		}
		tasks := getTasks(r.Context(), conn, user.ID)

		responses := make([]TaskResponse, len(tasks))
		for i := range tasks {
			layout := Layout
			if tasks[i].Interval == Hourly {
				layout = LayoutHourly
			}
			unit := tasks[i].Interval.toTime()

			date := time.Now().Add(-unit * time.Duration(limit-1))

			completions := getCompletions(r.Context(), conn, tasks[i].ID, date)

			intervalsMap := make(map[string]bool)
			for j := 0; j < limit; j++ {
				timestamp := date.Add(time.Duration(j) * unit).Format(layout)
				intervalsMap[timestamp] = false
			}

			for _, c := range completions {
				timestamp := c.CompletedAt.Format(layout)
				intervalsMap[timestamp] = true
			}

			resp := TaskResponse{
				ID:           tasks[i].ID,
				Name:         tasks[i].Name,
				Description:  tasks[i].Description,
				CreatedAt:    tasks[i].CreatedAt,
				Interval:     tasks[i].Interval.String(),
				IntervalsMap: intervalsMap,
			}

			responses[i] = resp
		}

		if err := json.NewEncoder(w).Encode(responses); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func handleCreateTask(conn *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(UserKey("user")).(User)
		if user.ID == 0 {
			http.Error(w, "user not found", http.StatusUnauthorized)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		name := r.Form.Get("name")
		description := r.Form.Get("description")
		interval := fromString(r.Form.Get("interval"))

		t := Task{
			Name:        name,
			Description: description,
			Interval:    interval,
			UserID:      user.ID,
		}

		insertTask(r.Context(), conn, t)
	}
}
