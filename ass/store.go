package main

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

func createTables(ctx context.Context, conn *pgxpool.Pool) error {
	_, err := conn.Exec(ctx, `
DO $$ BEGIN
	CREATE TYPE interval_enum AS ENUM ('Hourly', 'Daily', 'Weekly', 'Monthly');
EXCEPTION
    WHEN duplicate_object THEN null;
END $$;

CREATE TABLE if not exists tasks (
    id SERIAL PRIMARY KEY,
	user_id INT NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    interval interval_enum NOT NULL
);

CREATE TABLE if not exists completions (
	task_id INT NOT NULL,
	completed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE if not exists users (
	id SERIAL PRIMARY KEY,
	username VARCHAR(255) NOT NULL UNIQUE,
	password VARCHAR(255) NOT NULL,
	created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE if not exists magic_links (
	id SERIAL PRIMARY KEY,
	user_id INT NOT NULL,
	token VARCHAR(255) NOT NULL,
	valid BOOLEAN DEFAULT FALSE,
	created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE if not exists sessions (
	id SERIAL PRIMARY KEY,
	user_id INT NOT NULL,
	token VARCHAR(255) NOT NULL,
	created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS completions_task_id_completed_at_idx ON completions (task_id, completed_at);
CREATE INDEX IF NOT EXISTS magic_links_token_idx ON magic_links (token);
CREATE INDEX IF NOT EXISTS sessions_token_idx ON sessions (token);
CREATE INDEX IF NOT EXISTS users_username_idx ON users (username);
`)
	return err
}

func getMagicLinkByToken(ctx context.Context, conn *pgxpool.Pool, token string) (*MagicLink, error) {
	magicLink := &MagicLink{}
	err := conn.QueryRow(ctx, `
SELECT id, user_id, token, created_at
FROM magic_links
WHERE token = $1
AND valid = true`, token).Scan(&magicLink.ID, &magicLink.UserID, &magicLink.Token, &magicLink.CreatedAt)
	if err != nil {
		return &MagicLink{}, err
	}

	return magicLink, nil
}

func getMagicLink(ctx context.Context, conn *pgxpool.Pool, userId int) (*MagicLink, error) {
	magicLink := &MagicLink{}
	err := conn.QueryRow(ctx, `
SELECT id, user_id, token, created_at
FROM magic_links
WHERE user_id = $1
AND valid = true`, userId).Scan(&magicLink.ID, &magicLink.UserID, &magicLink.Token, &magicLink.CreatedAt)
	if err != nil {
		return &MagicLink{}, err
	}

	return magicLink, nil
}

func insertMagicLink(ctx context.Context, conn *pgxpool.Pool, token string, userID int) error {
	_, err := conn.Exec(ctx, `
UPDATE magic_links
SET valid = false
WHERE user_id = $1`, userID)
	if err != nil {
		return err
	}
	_, err = conn.Exec(ctx, `
INSERT INTO magic_links (valid, token, user_id)
VALUES (true, $1, $2)`, token, userID)

	return err
}

func insertSession(ctx context.Context, conn *pgxpool.Pool, userID int, token string) error {
	_, err := conn.Exec(ctx, `
INSERT INTO sessions (user_id, token)
VALUES ($1, $2)`, userID, token)
	return err
}

func getSession(ctx context.Context, conn *pgxpool.Pool, token string) (*Session, error) {
	session := &Session{}
	err := conn.QueryRow(ctx, `
SELECT id, user_id, token, created_at
FROM sessions
WHERE token = $1`, token).Scan(&session.ID, &session.UserID, &session.Token, &session.CreatedAt)
	if err != nil {
		return nil, err
	}

	return session, nil
}

func getUser(ctx context.Context, conn *pgxpool.Pool, username string) (*User, error) {
	user := &User{}
	err := conn.QueryRow(ctx, `
SELECT id, username, created_at
FROM users
WHERE username = $1`, username).Scan(&user.ID, &user.Username, &user.CreatedAt)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func getUserByID(ctx context.Context, conn *pgxpool.Pool, id int) (*User, error) {
	user := &User{}
	err := conn.QueryRow(ctx, `
SELECT id, username, created_at
FROM users
WHERE id = $1`, id).Scan(&user.ID, &user.Username, &user.CreatedAt)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func comparePassword(ctx context.Context, conn *pgxpool.Pool, username, password string) (bool, error) {
	var hashedPassword string
	err := conn.QueryRow(ctx, `
SELECT password
FROM users
WHERE username = $1`, username).Scan(&hashedPassword)
	if err != nil {
		return false, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return false, err
	}

	return true, nil
}

func insertUser(ctx context.Context, conn *pgxpool.Pool, username string, password string) (*User, error) {
	passwordHash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}

	_, err = conn.Exec(ctx, `
INSERT INTO users (username, password)
VALUES ($1, $2)`, username, passwordHash)
	if err != nil {
		return nil, err
	}

	return getUser(ctx, conn, username)
}

func getCompletions(ctx context.Context, conn *pgxpool.Pool, taskID int, dateEnd time.Time) ([]*Completion, error) {
	queryTime := dateEnd.Format(time.RFC3339)
	query := `
SELECT task_id, completed_at
FROM completions
WHERE task_id = $1
AND completed_at > $2`

	rows, err := conn.Query(ctx, query, taskID, queryTime)
	if err != nil {
		return nil, err
	}

	completions := []*Completion{}
	for rows.Next() {
		c := &Completion{}
		err := rows.Scan(&c.TaskID, &c.CompletedAt)
		if err != nil {
			return nil, err
		}

		completions = append(completions, c)
	}

	return completions, nil
}

func completeTask(ctx context.Context, conn *pgxpool.Pool, taskID int) error {
	task, err := getTask(ctx, conn, taskID)
	if err != nil {
		return err
	}

	interval := ""
	switch task.Interval {
	case Hourly:
		interval = "NOW() - INTERVAL '1 hour'"
	case Daily:
		interval = "NOW() - INTERVAL '1 day'"
	case Weekly:
		interval = "NOW() - INTERVAL '1 week'"
	case Monthly:
		interval = "NOW() - INTERVAL '1 month'"
	}

	query := `
SELECT task_id, completed_at
FROM completions
WHERE task_id = $1
AND completed_at > ` + interval

	rows, err := conn.Query(ctx, query, taskID)
	if err != nil {
		return err
	}

	completions := []Completion{}
	for rows.Next() {
		var c Completion
		err := rows.Scan(&c.TaskID, &c.CompletedAt)
		if err != nil {
			return err
		}

		completions = append(completions, c)
	}

	if len(completions) > 0 {
		return nil
	}

	_, err = conn.Exec(ctx, `
INSERT INTO completions (task_id)
VALUES ($1)`, taskID)
	return err
}

func getTasks(ctx context.Context, conn *pgxpool.Pool, userId int) ([]*Task, error) {
	rows, err := conn.Query(ctx, `
SELECT id, name, description, created_at, interval
FROM tasks
WHERE user_id = $1
	`, userId)
	if err != nil {
		return nil, err
	}

	tasks := []*Task{}
	for rows.Next() {
		task := &Task{}
		var interval string
		err := rows.Scan(&task.ID, &task.Name, &task.Description, &task.CreatedAt, &interval)
		if err != nil {
			return nil, err
		}
		task.Interval = fromString(interval)

		tasks = append(tasks, task)
	}

	return tasks, nil
}

func insertTask(ctx context.Context, conn *pgxpool.Pool, task Task) error {
	_, err := conn.Exec(ctx, `
		INSERT INTO tasks (name, user_id, description, interval)
		VALUES ($1, $2, $3, $4)
		`, task.Name, task.UserID, task.Description, task.Interval.String())

	return err
}

//	func updateTask(ctx context.Context, conn *pgxpool.Pool, task Task) error {
//		_, err := conn.Exec(ctx, `
//			UPDATE tasks
//			SET name = $1, description = $2, interval = $3
//			WHERE id = $6
//			`, task.Name, task.Description, task.Interval.String(), task.ID)
//		if err != nil {
//			return err
//		}
//	}
//
//	func deleteTask(ctx context.Context, conn *pgxpool.Pool, id int) error {
//		_, err := conn.Exec(ctx, `
//			DELETE FROM tasks
//			WHERE id = $1
//			`, id)
//		if err != nil {
//			return err
//		}
//	}
func getTask(ctx context.Context, conn *pgxpool.Pool, id int) (*Task, error) {
	task := &Task{}
	var interval string
	err := conn.QueryRow(ctx, `
		SELECT id, name, description, created_at, interval
		FROM tasks
		WHERE id = $1
		`, id).Scan(&task.ID, &task.Name, &task.Description, &task.CreatedAt, &interval)
	if err != nil {
		return nil, err
	}

	task.Interval = fromString(interval)

	return task, nil
}
