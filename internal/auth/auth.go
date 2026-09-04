package auth

import "time"

type Identity struct {
	UserID    *int64
	SessionID int64
}

type Session struct {
	ID        int64
	Token     string
	UserID    *int64
	ExpiresAt time.Time
	CreatedAt time.Time
}
