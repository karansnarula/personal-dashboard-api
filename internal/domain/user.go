package domain

import "time"

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

// AuthToken is what a successful login yields.
type AuthToken struct {
	Token     string
	ExpiresIn time.Duration
}
