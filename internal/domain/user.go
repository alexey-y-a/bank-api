package domain

import (
	"errors"
	"time"
)

var (
	ErrInvalidEmail    = errors.New("invalid email format or length")
	ErrInvalidUsername = errors.New("invalid username format or length")
	ErrWeakPassword    = errors.New("password must be at least 8 characters")
	ErrEmptyFields     = errors.New("email, username and password are required")
)

type User struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewUser(email, username, password string) (*User, error) {
	if email == "" || username == "" || password == "" {
		return nil, ErrEmptyFields
	}

	if len(email) < 5 || len(email) > 255 {
		return nil, ErrInvalidEmail
	}

	if len(username) < 3 || len(username) > 50 {
		return nil, ErrInvalidUsername
	}

	if len(password) < 8 {
		return nil, ErrWeakPassword
	}

	now := time.Now()

	return &User{
		Email:     email,
		Username:  username,
		Password:  password,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
