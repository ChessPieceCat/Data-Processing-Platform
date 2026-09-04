package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"time"
)

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

// Generate a session token
func GenerateSessionToken() (string, error) {
	token := make([]byte, 32)

	if _, err := rand.Read(token); err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(token), nil
}

func StoreGuestSession(db *sql.DB, token string) error {
	_, err := db.Exec(`
        INSERT INTO sessions (token, created_at, expires_at)
        VALUES ($1, $2, $3)
    `,
		token,
		time.Now(),
		time.Now().Add(24*time.Hour),
	)

	return err
}
