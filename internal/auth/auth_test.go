package auth

import (
	"database/sql"
	"encoding/base64"
	"testing"
	"time"

	"github.com/ChessPieceCat/Data-Processing-Platform/internal/database"
)

func TestGenerateSessionToken(t *testing.T) {
	token, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("GenerateSessionToken returned error: %v", err)
	}

	if token == "" {
		t.Fatal("expected non-empty session token")
	}

	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("session token is not valid base64url: %v", err)
	}

	if len(decoded) != 32 {
		t.Fatalf("expected 32 random bytes, got %d", len(decoded))
	}
}

func TestGenerateSessionTokenUnique(t *testing.T) {
	token1, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("GenerateSessionToken returned error: %v", err)
	}

	token2, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("GenerateSessionToken returned error: %v", err)
	}

	if token1 == token2 {
		t.Fatal("expected generated session tokens to be unique")
	}
}

func TestStoreGuestSession(t *testing.T) {
	m := database.RunMigrations()

	if m == nil {
		t.Fatal("RunMigrations returned nil")
	}

	t.Cleanup(func() {
		database.CloseMigrations(m)
	})

	db := database.OpenDatabase()
	defer db.Close()

	if err := db.Ping(); err != nil {
		t.Skipf("PostgreSQL is not available: %v", err)
	}

	token, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("GenerateSessionToken returned error: %v", err)
	}

	if err := StoreGuestSession(db, token); err != nil {
		t.Fatalf("StoreGuestSession returned error: %v", err)
	}

	var (
		id        int64
		userID    sql.NullInt64
		createdAt time.Time
		expiresAt time.Time
	)

	err = db.QueryRow(`
		SELECT id, user_id, created_at, expires_at
		FROM sessions
		WHERE token = $1
	`, token).Scan(
		&id,
		&userID,
		&createdAt,
		&expiresAt,
	)

	if err != nil {
		t.Fatalf("failed to retrieve stored session: %v", err)
	}

	if id == 0 {
		t.Fatal("expected session to have a database ID")
	}

	if userID.Valid {
		t.Fatalf("expected guest session to have NULL user_id, got %d", userID.Int64)
	}

	if expiresAt.Sub(createdAt) != 24*time.Hour {
		t.Fatalf(
			"expected session duration of 24 hours, got %v",
			expiresAt.Sub(createdAt),
		)
	}

	expectedExpiration := createdAt.Add(24 * time.Hour)

	if expiresAt.Before(expectedExpiration.Add(-time.Second)) ||
		expiresAt.After(expectedExpiration.Add(time.Second)) {
		t.Fatalf(
			"expected expires_at approximately %v, got %v",
			expectedExpiration,
			expiresAt,
		)
	}

	t.Cleanup(func() {
		_, _ = db.Exec(
			"DELETE FROM sessions WHERE token = $1",
			token,
		)
	})
}
