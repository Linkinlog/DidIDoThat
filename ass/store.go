package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

func createTables(ctx context.Context, conn *pgx.Conn) {
	_, err := conn.Exec(ctx, `
DO $$ BEGIN
	CREATE TYPE interval_enum AS ENUM ('Hourly', 'Daily', 'Weekly', 'Monthly');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE if not exists tasks (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    interval interval_enum NOT NULL
);

CREATE TABLE if not exists completions (
	task_id INT NOT NULL,
	completed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS completions_task_id_completed_at_idx ON completions (task_id, completed_at);
`)
	if err != nil {
		panic(err)
	}
}

func getCompletions(ctx context.Context, conn *pgx.Conn, taskID int, dateEnd time.Time) []Completion {
	queryTime := dateEnd.Format(time.RFC3339)
	query := `
SELECT task_id, completed_at
FROM completions
WHERE task_id = $1
AND completed_at > $2`

	rows, err := conn.Query(ctx, query, taskID, queryTime)
	if err != nil {
		panic(err)
	}

	completions := []Completion{}
	for rows.Next() {
		var c Completion
		err := rows.Scan(&c.TaskID, &c.CompletedAt)
		if err != nil {
			panic(err)
		}

		completions = append(completions, c)
	}

	return completions
}

func completeTask(ctx context.Context, conn *pgx.Conn, taskID int) {
	task := getTask(ctx, conn, taskID)

	interval := ""
	switch task.Interval {
	case Hourly:
		interval = "NOW() -INTERVAL 1 hour"
	case Daily:
		interval = "NOW() -INTERVAL 1 day"
	case Weekly:
		interval = "NOW() -INTERVAL 1 week"
	case Monthly:
		interval = "NOW() -INTERVAL 1 month"
	}

	query := `
SELECT task_id, completed_at
FROM completions
WHERE task_id = $1
AND completed_at > $2`

	rows, err := conn.Query(ctx, query, taskID, interval)
	if err != nil {
		panic(err)
	}

	completions := []Completion{}
	for rows.Next() {
		var c Completion
		err := rows.Scan(&c.TaskID, &c.CompletedAt)
		if err != nil {
			panic(err)
		}

		completions = append(completions, c)
	}

	if len(completions) > 0 {
		return
	}

	_, err = conn.Exec(ctx, `
INSERT INTO completions (task_id)
VALUES ($1)`, taskID)
	if err != nil {
		panic(err)
	}
}

func getTasks(ctx context.Context, conn *pgx.Conn) []Task {
	rows, err := conn.Query(ctx, `
SELECT id, name, description, created_at, interval
FROM tasks
	`)
	if err != nil {
		panic(err)
	}

	tasks := []Task{}
	for rows.Next() {
		var task Task
		var interval string
		err := rows.Scan(&task.ID, &task.Name, &task.Description, &task.CreatedAt, &interval)
		if err != nil {
			panic(err)
		}
		task.Interval = fromString(interval)

		tasks = append(tasks, task)
	}

	return tasks
}

func insertTask(ctx context.Context, conn *pgx.Conn, task Task) {
	_, err := conn.Exec(ctx, `
		INSERT INTO tasks (name, description, interval)
		VALUES ($1, $2, $3)
		`, task.Name, task.Description, task.Interval.String())
	if err != nil {
		panic(err)
	}
}

//	func updateTask(ctx context.Context, conn *pgx.Conn, task Task) {
//		_, err := conn.Exec(ctx, `
//			UPDATE tasks
//			SET name = $1, description = $2, interval = $3
//			WHERE id = $6
//			`, task.Name, task.Description, task.Interval.String(), task.ID)
//		if err != nil {
//			panic(err)
//		}
//	}
//
//	func deleteTask(ctx context.Context, conn *pgx.Conn, id int) {
//		_, err := conn.Exec(ctx, `
//			DELETE FROM tasks
//			WHERE id = $1
//			`, id)
//		if err != nil {
//			panic(err)
//		}
//	}
func getTask(ctx context.Context, conn *pgx.Conn, id int) Task {
	var task Task
	var interval string
	err := conn.QueryRow(ctx, `
		SELECT id, name, description, created_at, interval
		FROM tasks
		WHERE id = $1
		`, id).Scan(&task.ID, &task.Name, &task.Description, &task.CreatedAt, &interval)
	if err != nil {
		panic(err)
	}

	task.Interval = fromString(interval)

	return task
}
