// Package server wires the HTTP router, middleware chain, and modules, and owns
// the listener lifecycle.
package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/locnguyen0904/devhub/backend/internal/config"
	"github.com/locnguyen0904/devhub/backend/internal/modules/auth"
	"github.com/locnguyen0904/devhub/backend/internal/modules/comment"
	"github.com/locnguyen0904/devhub/backend/internal/modules/health"
	"github.com/locnguyen0904/devhub/backend/internal/modules/post"
	"github.com/locnguyen0904/devhub/backend/internal/modules/reaction"
	"github.com/locnguyen0904/devhub/backend/internal/modules/tag"
	"github.com/locnguyen0904/devhub/backend/internal/modules/upload"
	"github.com/locnguyen0904/devhub/backend/internal/modules/user"
	"github.com/locnguyen0904/devhub/backend/internal/platform/cache"
	"github.com/locnguyen0904/devhub/backend/internal/platform/database"
	"github.com/locnguyen0904/devhub/backend/internal/platform/httpx"
	"github.com/locnguyen0904/devhub/backend/internal/platform/markdown"
	"github.com/locnguyen0904/devhub/backend/internal/platform/storage"
)

// requestTimeout bounds ordinary requests. Long-running operations get their
// own budget when they appear.
const requestTimeout = 15 * time.Second

// Background runs a module's periodic workers until ctx is cancelled.
type Background func(ctx context.Context)

// NewAPI builds the chi router, mounts huma on it, and registers every module.
// It also returns a Background runner for the modules' periodic workers.
//
// There is no CORS middleware and that is deliberate: frontend and API share an
// origin (docs/01-architecture.md §9), so preflight never happens.
func NewAPI(cfg config.Config, log *slog.Logger, db *database.DB, redis *cache.Client) (chi.Router, huma.API, Background) {
	httpx.UseErrorModel()

	// Modules are built first so their middleware can join the chain below.
	renderer := markdown.NewRenderer()
	userMod := user.New(db)
	authMod := auth.New(cfg, db, redis, userMod.Service, log)
	tagMod := tag.New(db)
	reactionMod := reaction.New(db)
	// post consumes reaction for viewer state and the bookmark list. The
	// dependency is one-way, so reaction is built first.
	postMod := post.New(db, userMod.Service, tagMod.Service, renderer, reactionMod.Service, redis, log)
	commentMod := comment.New(db, userMod.Service, renderer)
	uploadMod := upload.NewModule(storage.New(storage.Config{
		Endpoint:  cfg.Storage.Endpoint,
		Region:    cfg.Storage.Region,
		Bucket:    cfg.Storage.Bucket,
		AccessKey: cfg.Storage.AccessKey,
		SecretKey: cfg.Storage.SecretKey,
		PublicURL: cfg.Storage.PublicURL,
	}))

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	// No RealIP middleware: it rewrites r.RemoteAddr from X-Forwarded-For
	// whether or not the infrastructure sets it, so any client can forge its own
	// address (GHSA-3fxj-6jh8-hvhx). Nothing needs the client IP yet. Phase 4
	// adds rate limiting and must derive the IP from a trusted proxy hop only.
	r.Use(middleware.Recoverer)
	r.Use(requestLogger(log))
	r.Use(middleware.Timeout(requestTimeout))
	// Optional auth on every route: it attaches the caller when a valid bearer
	// token is present and is a no-op otherwise. Protected handlers enforce
	// presence with auth.RequireIdentity.
	r.Use(authMod.OptionalAuth())

	// Unmatched routes never reach a huma operation, so they need their own
	// handlers — otherwise clients would meet two different error shapes.
	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteError(w, httpx.CodeNotFound, "No route matches this path")
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, _ *http.Request) {
		httpx.WriteError(w, httpx.CodeMethodNotAllowed, "Method not allowed for this path")
	})

	cfgAPI := huma.DefaultConfig("DevHub API", "0.1.0")
	cfgAPI.Info.Description = "API for the DevHub technical blogging platform."
	// Declare the bearer scheme so protected operations document it and the
	// generated client knows to send Authorization.
	cfgAPI.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"bearer": {Type: "http", Scheme: "bearer", BearerFormat: "JWT"},
	}
	// Drop huma's $schema link injection: it adds a field to every response body
	// and therefore to every generated TypeScript type, for no benefit to a
	// first-party client that already has the types. Clearing Transformers alone
	// is not enough — DefaultConfig re-adds it through CreateHooks.
	cfgAPI.Transformers = nil
	cfgAPI.CreateHooks = nil
	api := humachi.New(r, cfgAPI)

	health.New(db, redis).Register(api)
	authMod.Register(r, api)
	tagMod.Register(api)
	postMod.Register(api)
	commentMod.Register(api)
	reactionMod.Register(api)
	uploadMod.Register(api)

	// Periodic workers. Started by main after the openapi subcommand check, so
	// generating the spec never spins up a background loop.
	background := func(ctx context.Context) {
		postMod.RunViewFlusher(ctx)
	}

	return r, api, background
}
