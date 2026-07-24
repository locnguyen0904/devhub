package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/locnguyen0904/devhub/backend/internal/platform/httpx"
	"github.com/locnguyen0904/devhub/backend/internal/platform/logger"
	"github.com/locnguyen0904/devhub/backend/internal/platform/token"
)

type contextKey int

const identityKey contextKey = iota

// Identity is the authenticated caller attached to a request's context.
type Identity struct {
	UserID   uuid.UUID
	Username string
}

// verifier is the slice of token.Issuer the middleware needs.
type verifier interface {
	Verify(raw string) (token.Claims, error)
}

// OptionalAuth attaches an Identity when the request carries a valid bearer
// token, and otherwise lets the request through unauthenticated. Endpoints that
// require a user call RequireIdentity; this keeps one middleware serving both
// public and protected routes instead of two parallel chains.
func OptionalAuth(v verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := bearerToken(r.Header.Get("Authorization"))
			if raw == "" {
				next.ServeHTTP(w, r)
				return
			}

			claims, err := v.Verify(raw)
			if err != nil {
				// A malformed or expired token on an optional route is not an
				// error: the request simply proceeds as anonymous.
				next.ServeHTTP(w, r)
				return
			}

			userID, err := claims.UserID()
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), identityKey,
				Identity{UserID: userID, Username: claims.Username})
			ctx = logger.WithUserID(ctx, userID.String())
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// IdentityFromContext returns the caller if the request was authenticated.
func IdentityFromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityKey).(Identity)
	return id, ok
}

// RequireIdentity returns the caller or an unauthenticated error, for handlers
// that must have a signed-in user.
func RequireIdentity(ctx context.Context) (Identity, error) {
	id, ok := IdentityFromContext(ctx)
	if !ok {
		return Identity{}, httpx.New(httpx.CodeUnauthenticated, "Authentication required", nil)
	}
	return id, nil
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if len(header) > len(prefix) && strings.EqualFold(header[:len(prefix)], prefix) {
		return header[len(prefix):]
	}
	return ""
}
