// Package server wires the HTTP router, middleware chain, and modules, and owns
// the listener lifecycle.
package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/locnguyen0904/devhub/backend/internal/modules/health"
	"github.com/locnguyen0904/devhub/backend/internal/platform/database"
	"github.com/locnguyen0904/devhub/backend/internal/platform/httpx"
)

// requestTimeout bounds ordinary requests. Long-running operations get their
// own budget when they appear.
const requestTimeout = 15 * time.Second

// NewAPI builds the chi router, mounts huma on it, and registers every module.
//
// There is no CORS middleware and that is deliberate: frontend and API share an
// origin (docs/01-architecture.md §9), so preflight never happens.
func NewAPI(log *slog.Logger, db *database.DB, redis health.Pinger) (chi.Router, huma.API) {
	httpx.UseErrorModel()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	// No RealIP middleware: it rewrites r.RemoteAddr from X-Forwarded-For
	// whether or not the infrastructure sets it, so any client can forge its own
	// address (GHSA-3fxj-6jh8-hvhx). Nothing needs the client IP yet. Phase 4
	// adds rate limiting and must derive the IP from a trusted proxy hop only.
	r.Use(middleware.Recoverer)
	r.Use(requestLogger(log))
	r.Use(middleware.Timeout(requestTimeout))

	// Unmatched routes never reach a huma operation, so they need their own
	// handlers — otherwise clients would meet two different error shapes.
	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteError(w, httpx.CodeNotFound, "No route matches this path")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteError(w, httpx.CodeMethodNotAllowed, "Method not allowed for this path")
	})

	cfg := huma.DefaultConfig("DevHub API", "0.1.0")
	cfg.Info.Description = "API for the DevHub technical blogging platform."
	// Drop huma's $schema link injection: it adds a field to every response body
	// and therefore to every generated TypeScript type, for no benefit to a
	// first-party client that already has the types. Clearing Transformers alone
	// is not enough — DefaultConfig re-adds it through CreateHooks.
	cfg.Transformers = nil
	cfg.CreateHooks = nil
	api := humachi.New(r, cfg)

	health.New(db, redis).Register(api)

	return r, api
}
