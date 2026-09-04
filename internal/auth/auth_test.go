package auth

import (
	"database/sql"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
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

func TestSetSessionCookie(t *testing.T) {
	rec := httptest.NewRecorder()

	token := "test-session-token"

	SetSessionCookie(rec, token)

	cookies := rec.Result().Cookies()

	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie, got %d", len(cookies))
	}

	cookie := cookies[0]

	if cookie.Name != SessionCookieName {
		t.Fatalf("expected cookie name %q, got %q",
			SessionCookieName,
			cookie.Name,
		)
	}

	if cookie.Value != token {
		t.Fatalf("expected cookie value %q, got %q",
			token,
			cookie.Value,
		)
	}

	if !cookie.HttpOnly {
		t.Fatal("expected HttpOnly cookie")
	}

	if !cookie.Secure {
		t.Fatal("expected Secure cookie")
	}

	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("expected SameSite=Lax, got %v", cookie.SameSite)
	}

	if cookie.Path != "/" {
		t.Fatalf("expected Path=/, got %q", cookie.Path)
	}
}

func TestGetSessionToken(t *testing.T) {
	token := "test-session-token"

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  SessionCookieName,
		Value: token,
	})

	got, err := GetSessionToken(req)
	if err != nil {
		t.Fatalf("GetSessionToken returned error: %v", err)
	}

	if got != token {
		t.Fatalf("expected token %q, got %q", token, got)
	}
}

func TestGetSessionTokenMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	_, err := GetSessionToken(req)

	if err == nil {
		t.Fatal("expected error when session cookie is missing")
	}
}

func TestSessionMiddlewareNoCookie(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := IdentityFromContext(r)

		if ok {
			t.Fatalf("expected no identity, got %+v", identity)
		}

		w.WriteHeader(http.StatusOK)
	})

	handler := SessionMiddleware(nil, next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestSessionMiddlewareInvalidToken(t *testing.T) {
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

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := IdentityFromContext(r)

		if ok {
			t.Fatalf("expected no identity, got %+v", identity)
		}

		w.WriteHeader(http.StatusOK)
	})

	handler := SessionMiddleware(db, next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  SessionCookieName,
		Value: "invalid-session-token",
	})

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestSessionMiddlewareExpiredSession(t *testing.T) {
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

	_, err = db.Exec(`
		INSERT INTO sessions (token, created_at, expires_at)
		VALUES ($1, $2, $3)
	`,
		token,
		time.Now().Add(-48*time.Hour),
		time.Now().Add(-24*time.Hour),
	)
	if err != nil {
		t.Fatalf("failed to create expired session: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.Exec(
			"DELETE FROM sessions WHERE token = $1",
			token,
		)
	})

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := IdentityFromContext(r)

		if ok {
			t.Fatalf("expected no identity, got %+v", identity)
		}

		w.WriteHeader(http.StatusOK)
	})

	handler := SessionMiddleware(db, next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  SessionCookieName,
		Value: token,
	})

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestSessionMiddlewareValidGuestSession(t *testing.T) {
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

	t.Cleanup(func() {
		_, _ = db.Exec(
			"DELETE FROM sessions WHERE token = $1",
			token,
		)
	})

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := IdentityFromContext(r)

		if !ok {
			t.Fatal("expected identity in request context")
		}

		if identity == nil {
			t.Fatal("expected non-nil identity")
		}

		if identity.UserID != nil {
			t.Fatalf(
				"expected guest session to have nil UserID, got %d",
				*identity.UserID,
			)
		}

		if identity.SessionID == 0 {
			t.Fatal("expected non-zero SessionID")
		}

		w.WriteHeader(http.StatusOK)
	})

	handler := SessionMiddleware(db, next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  SessionCookieName,
		Value: token,
	})

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestSessionMiddlewareAuthenticatedSession(t *testing.T) {
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

	var userID int64

	err := db.QueryRow(`
		INSERT INTO users (username, password_hash, created_at)
		VALUES ($1, $2, $3)
		RETURNING id
	`,
		"middleware_test_user",
		"test-password-hash",
		time.Now(),
	).Scan(&userID)
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}

	token, err := GenerateSessionToken()
	if err != nil {
		t.Fatalf("GenerateSessionToken returned error: %v", err)
	}

	_, err = db.Exec(`
		INSERT INTO sessions (token, user_id, created_at, expires_at)
		VALUES ($1, $2, $3, $4)
	`,
		token,
		userID,
		time.Now(),
		time.Now().Add(24*time.Hour),
	)
	if err != nil {
		t.Fatalf("failed to create authenticated session: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.Exec(
			"DELETE FROM sessions WHERE token = $1",
			token,
		)
		_, _ = db.Exec(
			"DELETE FROM users WHERE id = $1",
			userID,
		)
	})

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		identity, ok := IdentityFromContext(r)

		if !ok {
			t.Fatal("expected identity in request context")
		}

		if identity == nil {
			t.Fatal("expected non-nil identity")
		}

		if identity.UserID == nil {
			t.Fatal("expected authenticated session to have UserID")
		}

		if *identity.UserID != userID {
			t.Fatalf(
				"expected UserID %d, got %d",
				userID,
				*identity.UserID,
			)
		}

		if identity.SessionID == 0 {
			t.Fatal("expected non-zero SessionID")
		}

		w.WriteHeader(http.StatusOK)
	})

	handler := SessionMiddleware(db, next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{
		Name:  SessionCookieName,
		Value: token,
	})

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}
