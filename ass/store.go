package main

import (
	"context"

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
    completed BOOLEAN DEFAULT FALSE,
    completed_at TIMESTAMP,
    interval interval_enum NOT NULL
);
	`)
	if err != nil {
		panic(err)
	}
}

func dropTables(ctx context.Context, conn *pgx.Conn) {
	_, err := conn.Exec(ctx, `
DROP TABLE IF EXISTS tasks;
DROP TYPE IF EXISTS interval_enum;`)
	if err != nil {
		panic(err)
	}
}

func insertTask(ctx context.Context, conn *pgx.Conn, task Task) {
	_, err := conn.Exec(ctx, `
INSERT INTO tasks (name, description, completed, completed_at, interval)
VALUES ($1, $2, $3, $4, $5)
	`, task.Name, task.Description, task.Completed, task.CompletedAt, task.Interval.String())
	if err != nil {
		panic(err)
	}
}

func updateTask(ctx context.Context, conn *pgx.Conn, task Task) {
	_, err := conn.Exec(ctx, `
UPDATE tasks
SET name = $1, description = $2, completed = $3, completed_at = $4, interval = $5
WHERE id = $6
	`, task.Name, task.Description, task.Completed, task.CompletedAt, task.Interval.String(), task.ID)
	if err != nil {
		panic(err)
	}
}

func deleteTask(ctx context.Context, conn *pgx.Conn, id int) {
	_, err := conn.Exec(ctx, `
DELETE FROM tasks
WHERE id = $1
	`, id)
	if err != nil {
		panic(err)
	}
}

func getTask(ctx context.Context, conn *pgx.Conn, id int) Task {
	var task Task
	var interval string
	err := conn.QueryRow(ctx, `
SELECT id, name, description, completed, completed_at, interval
FROM tasks
WHERE id = $1
	`, id).Scan(&task.ID, &task.Name, &task.Description, &task.Completed, &task.CompletedAt, &interval)
	if err != nil {
		panic(err)
	}

	task.Interval = fromString(interval)

	return task
}

func getTasks(ctx context.Context, conn *pgx.Conn) []Task {
	rows, err := conn.Query(ctx, `
SELECT id, name, description, completed, completed_at, interval
FROM tasks
	`)
	if err != nil {
		panic(err)
	}

	tasks := []Task{}
	for rows.Next() {
		var task Task
		var interval string
		err := rows.Scan(&task.ID, &task.Name, &task.Description, &task.Completed, &task.CompletedAt, &interval)
		if err != nil {
			panic(err)
		}
		task.Interval = fromString(interval)

		tasks = append(tasks, task)
	}

	return tasks
}
