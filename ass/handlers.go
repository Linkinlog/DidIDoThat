package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	LayoutHourly = "2006-01-02T15Z07:00"
	Layout       = "2006-01-02Z07:00"
	previewLimit = 30
)

func startHTTP(port int, conn *pgxpool.Pool) error {
	mux := http.NewServeMux()

	// unauthorized
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		if err := conn.Ping(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logger.Error("Unable to ping database", "error", err.Error())
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

	fs := http.FileServer(http.Dir("/dist"))
	mux.Handle("/", fs)

	return http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
}

func hello(w http.ResponseWriter, _ *http.Request) {
	w.Write([]byte("Hello, World!"))
}

func withUser(conn *pgxpool.Pool, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			logger.Error("Unable to get cookie", "error", err.Error())
			return
		}

		session, err := getSession(r.Context(), conn, cookie.Value)
		if err != nil {
			logger.Error("Unable to get session", "error", err.Error())
			http.Error(w, "Unable to get session", http.StatusInternalServerError)
			return
		}
		if session.ID == 0 {
			http.Error(w, "session not found", http.StatusUnauthorized)
			logger.Error("Session not found", "error", "session not found")
			return
		}

		user, err := getUserByID(r.Context(), conn, session.UserID)
		if err != nil {
			if !errors.Is(err, pgx.ErrNoRows) && err.Error() != "no rows in result set" {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if user.ID == 0 {
			http.Error(w, "user not found", http.StatusUnauthorized)
			logger.Error("User not found", "error", "user not found")
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
		user := r.Context().Value(UserKey("user")).(*User)
		if user.ID == 0 {
			http.Error(w, "user not found", http.StatusUnauthorized)
			logger.Error("User not found", "error", "user not found")
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
			logger.Error("Magic token is required", "error", "magic_token is required")
			return
		}

		magicLink, err := getMagicLinkByToken(r.Context(), conn, magicToken)
		if err != nil {
			logger.Error("Unable to get magic link by token", "error", err.Error())
			http.Error(w, "Cannot verify magic link", http.StatusNotFound)
			return
		}

		user, err := getUserByID(r.Context(), conn, magicLink.UserID)
		if err != nil {
			logger.Error("Unable to get user by id", "error", err.Error())
			http.Error(w, "Cannot find user", http.StatusNotFound)
			return
		}

		token := newToken()

		if err := insertSession(r.Context(), conn, user.ID, token); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logger.Error("Unable to insert session", "error", err.Error())
			return
		}

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
		user := r.Context().Value(UserKey("user")).(*User)
		if user.ID == 0 {
			http.Error(w, "user not found", http.StatusUnauthorized)
			logger.Error("User not found", "error", "user not found")
			return
		}

		magicLink, err := getMagicLink(r.Context(), conn, user.ID)
		if err != nil {
			logger.Error("Unable to get magic link", "error", err.Error())
			if !errors.Is(err, pgx.ErrNoRows) && err.Error() != "no rows in result set" {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if magicLink == nil || magicLink.ID == 0 {
			if _, err := w.Write([]byte(magicLink.Token)); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				logger.Error("Unable to write magic link token", "error", err.Error())
			}
			return
		}

		token := newToken()
		if err := insertMagicLink(r.Context(), conn, token, user.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logger.Error("Unable to insert magic link", "error", err.Error())
			return
		}

		if _, err := w.Write([]byte(token)); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logger.Error("Unable to write magic link token", "error", err.Error())
			return
		}
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
			logger.Error("Unable to decode request body", "error", err.Error())
			return
		}

		username := body.Username
		password := body.Password

		if username == "" || password == "" {
			http.Error(w, "username and password are required", http.StatusBadRequest)
			logger.Error("Username and password are required", "error", "username and password are required")
			return
		}

		user, err := getUser(r.Context(), conn, username)
		if err != nil {
			logger.Error("Unable to get user", "error", err.Error())
			if !errors.Is(err, pgx.ErrNoRows) && err.Error() != "no rows in result set" {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		if user == nil || user.ID == 0 {
			var err error
			user, err = insertUser(r.Context(), conn, username, password)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				logger.Error("Unable to insert user", "error", err.Error())
				return
			}
		} else {
			valid, err := comparePassword(r.Context(), conn, username, password)
			if err != nil || !valid {
				http.Error(w, "invalid username or password", http.StatusUnauthorized)
				logger.Error("Invalid username or password", "error", "invalid username or password")
				return
			}
		}

		token := newToken()

		if err := insertSession(r.Context(), conn, user.ID, token); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logger.Error("Unable to insert session", "error", err.Error())
			return
		}

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
			logger.Error("Task ID is required", "error", "task_id is required")
			return
		}

		taskID, err := strconv.Atoi(taskIDStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			logger.Error("Unable to convert task ID to int", "error", err.Error())
			return
		}

		if err := completeTask(r.Context(), conn, taskID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logger.Error("Unable to complete task", "error", err.Error())
			return
		}
	}
}

func handleGetTasks(conn *pgxpool.Pool, limit int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := r.Context().Value(UserKey("user")).(*User)
		if user.ID == 0 {
			http.Error(w, "user not found", http.StatusUnauthorized)
			logger.Error("User not found", "error", "user not found")
			return
		}
		tasks, err := getTasks(r.Context(), conn, user.ID)
		if err != nil {
			logger.Error("Unable to get tasks", "error", err.Error())
			if !errors.Is(err, pgx.ErrNoRows) && err.Error() != "no rows in result set" {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		responses := make([]TaskResponse, len(tasks))
		for i := range tasks {
			layout := Layout
			if tasks[i].Interval == Hourly {
				layout = LayoutHourly
			}
			unit := tasks[i].Interval.toTime()

			date := time.Now().Add(-unit * time.Duration(limit-1))

			completions, err := getCompletions(r.Context(), conn, tasks[i].ID, date)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				if !errors.Is(err, pgx.ErrNoRows) && err.Error() != "no rows in result set" {
					logger.Error("Unable to get completions", "error", err.Error())
					return
				}
			}
			if completions == nil {
				completions = make([]*Completion, 0)
			}

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
		user := r.Context().Value(UserKey("user")).(*User)
		if user.ID == 0 {
			http.Error(w, "user not found", http.StatusUnauthorized)
			logger.Error("User not found", "error", "user not found")
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			logger.Error("Unable to parse form", "error", err.Error())
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

		if err := insertTask(r.Context(), conn, t); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			logger.Error("Unable to insert task", "error", err.Error())
			return
		}
	}
}
