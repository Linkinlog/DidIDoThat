package main

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type MagicLink struct {
	ID        int
	UserID    int
	Token     string
	CreatedAt time.Time
}

type Session struct {
	ID        int
	UserID    int
	Token     string
	CreatedAt time.Time
}

func newToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

type UserKey string

type User struct {
	ID        int
	Username  string
	CreatedAt time.Time
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(bytes), err
}

func fromString(s string) Interval {
	s = strings.ToLower(s)
	switch s {
	case "hourly":
		return Hourly
	case "daily":
		return Daily
	case "weekly":
		return Weekly
	case "monthly":
		return Monthly
	default:
		return 0
	}
}

type Interval int

const (
	_ Interval = iota
	Hourly
	Daily
	Weekly
	Monthly
)

func (i Interval) String() string {
	return [...]string{"", "Hourly", "Daily", "Weekly", "Monthly"}[i]
}

func (i Interval) toTime() time.Duration {
	switch i {
	case Hourly:
		return time.Hour
	case Daily:
		return 24 * time.Hour
	case Weekly:
		return 7 * 24 * time.Hour
	case Monthly:
		return 24 * 30 * time.Hour
	default:
		return 0
	}
}
