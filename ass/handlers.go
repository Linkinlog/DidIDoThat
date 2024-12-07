package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"
)

func startHTTP(port int, conn *pgx.Conn) {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /tasks", handleCreateTask(conn))
	mux.HandleFunc("GET /tasks", handleGetTasks(conn))

	http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
}

func handleGetTasks(conn *pgx.Conn) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tasks := getTasks(r.Context(), conn)

		if err := json.NewEncoder(w).Encode(tasks); err != nil {
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
