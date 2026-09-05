package auth

import (
	"context"
	"database/sql"
	"net/http"
	"time"
)

type contextKey string

const identityContextKey contextKey = "identity"

func SessionMiddleware(db *sql.DB, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := GetSessionToken(r)
		if err != nil {
			// No session cookie found, continue as anonymous.
			next.ServeHTTP(w, r)
			return
		}

		session, err := GetSession(db, token)
		if err != nil {
			// Session not found, continue as anonymous.
			next.ServeHTTP(w, r)
			return
		}

		if time.Now().After(session.ExpiresAt) {
			// Session expired, continue as anonymous.
			next.ServeHTTP(w, r)
			return
		}

		identity := &Identity{
			UserID:    session.UserID,
			SessionID: session.ID,
		}

		ctx := context.WithValue(
			r.Context(),
			identityContextKey,
			identity,
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func IdentityFromContext(r *http.Request) (*Identity, bool) {
	identity, ok := r.Context().Value(identityContextKey).(*Identity)
	return identity, ok
}
