// Package auth handles GitHub OAuth login, issues access and refresh tokens,
// and provides the middleware that authenticates requests.
package auth

import (
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/locnguyen0904/devhub/backend/internal/config"
	"github.com/locnguyen0904/devhub/backend/internal/platform/cache"
	"github.com/locnguyen0904/devhub/backend/internal/platform/database"
	"github.com/locnguyen0904/devhub/backend/internal/platform/token"
)

// tagAuth groups the auth operations in the OpenAPI spec.
const tagAuth = "auth"

// Module wires the auth module together.
type Module struct {
	handler  *Handler
	verifier verifier
}

// New builds the module. It takes user.Service through the userService port and
// the shared platform clients.
func New(
	cfg config.Config,
	db *database.DB,
	c *cache.Client,
	users userService,
	log *slog.Logger,
) *Module {
	issuer := token.NewIssuer(cfg.Auth.JWTSecret, cfg.Auth.AccessTTL)
	gh := newGitHubClient(cfg.Auth.GitHubClientID, cfg.Auth.GitHubClientSecret, cfg.Auth.GitHubRedirectURL)

	svc := newService(db, newRepository(db), users, gh, issuer, c, cfg.Auth.RefreshTTL)
	handler := newHandler(svc, log, cfg.FrontendURL, cfg.IsProduction())

	return &Module{handler: handler, verifier: issuer}
}

// OptionalAuth is the middleware other modules mount to authenticate requests.
func (m *Module) OptionalAuth() func(http.Handler) http.Handler {
	return OptionalAuth(m.verifier)
}

// Register mounts the browser-redirect routes on chi and the JSON endpoints on
// huma. The two GitHub endpoints are plain routes because they redirect and set
// cookies rather than return JSON.
func (m *Module) Register(r chi.Router, api huma.API) {
	r.Get("/api/v1/auth/github", m.handler.redirectToGitHub)
	r.Get("/api/v1/auth/github/callback", m.handler.callback)

	huma.Register(api, huma.Operation{
		OperationID: "refreshSession",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/refresh",
		Summary:     "Exchange the refresh cookie for a new access token",
		Tags:        []string{tagAuth},
	}, m.handler.refresh)

	huma.Register(api, huma.Operation{
		OperationID: "logout",
		Method:      http.MethodPost,
		Path:        "/api/v1/auth/logout",
		Summary:     "Revoke the current session",
		Tags:        []string{tagAuth},
	}, m.handler.logout)

	huma.Register(api, huma.Operation{
		OperationID:   "logoutAll",
		Method:        http.MethodPost,
		Path:          "/api/v1/auth/logout-all",
		Summary:       "Revoke every session for the current user",
		Tags:          []string{"auth"},
		Security:      []map[string][]string{{"bearer": {}}},
		DefaultStatus: http.StatusOK,
	}, m.handler.logoutAll)
}
