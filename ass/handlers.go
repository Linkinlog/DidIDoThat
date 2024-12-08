package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

const Layout = "2006-01-02 15:04:05"

func startHTTP(port int, conn *pgx.Conn) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("POST /api/tasks", handleCreateTask(conn))
	mux.HandleFunc("GET /api/tasks", handleGetTasks(conn))

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
		panic(err)
	}
}

func handleGetTasks(conn *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tasks := getTasks(r.Context(), conn)

		responses := make([]TaskResponse, len(tasks))
		for i := range tasks {
			var unit time.Duration
			switch tasks[i].Interval {
			case Hourly:
				unit = time.Hour
			case Daily:
				unit = 24 * time.Hour
			case Weekly:
				unit = 7 * 24 * time.Hour
			case Monthly:
				unit = 30 * 24 * time.Hour
			}

			date := time.Now().Add(-unit * 30)

			completions := getCompletions(r.Context(), conn, tasks[i].ID, date.Format(Layout))

			intervalsCompleted := make([]int, 0)
			for _, c := range completions {
				intervalsCompleted = append(intervalsCompleted, int(c.CompletedAt.Sub(date)/unit))
			}

			resp := TaskResponse{
				ID:                 tasks[i].ID,
				Name:               tasks[i].Name,
				Description:        tasks[i].Description,
				CreatedAt:          tasks[i].CreatedAt,
				Interval:           tasks[i].Interval.String(),
				IntervalsCompleted: intervalsCompleted,
			}

			responses[i] = resp
		}

		if err := json.NewEncoder(w).Encode(responses); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func handleCreateTask(conn *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		}

		insertTask(r.Context(), conn, t)
	}
}
