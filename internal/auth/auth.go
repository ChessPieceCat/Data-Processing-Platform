package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"net/http"
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

const SessionCookieName = "session_id"

func SetSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
	})
}

func GetSessionToken(r *http.Request) (string, error) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return "", err
	}

	return cookie.Value, nil
}

func GetSession(db *sql.DB, token string) (*Session, error) {
	var session Session

	err := db.QueryRow(`
		SELECT id, token, user_id, expires_at, created_at
		FROM sessions
		WHERE token = $1
	`, token).Scan(
		&session.ID,
		&session.Token,
		&session.UserID,
		&session.ExpiresAt,
		&session.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &session, nil
}
