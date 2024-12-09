package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	LayoutHourly = "2006-01-02T15Z07:00"
	Layout       = "2006-01-02Z07:00"
	previewLimit = 30
)

func startHTTP(port int, conn *pgx.Conn) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("POST /api/tasks", handleCreateTask(conn))
	mux.HandleFunc("GET /api/tasks", handleGetTasks(conn, previewLimit))
	mux.HandleFunc("POST /api/tasks/{taskId}/complete", handleCompleteTask(conn))

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
		panic(err)
	}
}

func handleCompleteTask(conn *pgx.Conn) http.HandlerFunc {
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

func handleGetTasks(conn *pgx.Conn, limit int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tasks := getTasks(r.Context(), conn)

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
