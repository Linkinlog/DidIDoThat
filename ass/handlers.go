package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	Layout       = time.RFC3339
	previewLimit = 30
)

func startHTTP(port int, conn *pgx.Conn) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("POST /api/tasks", handleCreateTask(conn))
	mux.HandleFunc("GET /api/tasks", handleGetTasks(conn, previewLimit))

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
		panic(err)
	}
}

func handleGetTasks(conn *pgx.Conn, limit int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tasks := getTasks(r.Context(), conn)

		responses := make([]TaskResponse, len(tasks))
		for i := range tasks {
			respLayout := Layout
			if tasks[i].Interval == Hourly {
				respLayout = Layout
			}
			unit := tasks[i].Interval.toTime()

			date := time.Now().Add(-unit * time.Duration(limit-1))

			completions := getCompletions(r.Context(), conn, tasks[i].ID, date)

			intervalsMap := make(map[string]bool)
			for j := 0; j < limit; j++ {
				timestamp := date.Add(time.Duration(j) * unit).Format(respLayout)
				intervalsMap[timestamp] = false
			}

			for _, c := range completions {
				d := date.Add(c.CompletedAt.Sub(date) / unit * unit).Format(respLayout)

				intervalsMap[d] = true
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
