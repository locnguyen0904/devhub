package post

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/locnguyen0904/devhub/backend/internal/platform/cache"
	"github.com/locnguyen0904/devhub/backend/internal/platform/database"
)

// tagName groups the post operations in the OpenAPI spec.
const (
	tagName  = "posts"
	pathByID = "/api/v1/posts/{id}"
)

// Module wires the post module.
type Module struct {
	handler *Handler
	svc     Service
	log     *slog.Logger
}

// New builds the module from its dependencies. users and tags arrive as the
// consumer-defined ports, renderer as the markdown port, cache for the hot feed.
func New(db *database.DB, users authorFinder, tags tagLinker, renderer markdownRenderer, c *cache.Client, log *slog.Logger) *Module {
	svc := newService(db, newRepository(db), users, tags, renderer, c, log)
	return &Module{handler: newHandler(svc), svc: svc, log: log}
}

// RunViewFlusher periodically writes buffered views to Postgres until ctx is
// cancelled, then does one final flush so the last window is not lost. Run it
// in a goroutine; it blocks until shutdown.
func (m *Module) RunViewFlusher(ctx context.Context) {
	ticker := time.NewTicker(viewFlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// context.WithoutCancel: the app context is already cancelled, but the
			// final flush still needs to reach Redis and Postgres.
			if err := m.svc.FlushViews(context.WithoutCancel(ctx)); err != nil {
				m.log.Error("final view flush", slog.String("error", err.Error()))
			}
			return
		case <-ticker.C:
			if err := m.svc.FlushViews(ctx); err != nil {
				m.log.ErrorContext(ctx, "view flush", slog.String("error", err.Error()))
			}
		}
	}
}

// Register mounts the post operations. Public reads carry no security; writes
// require a bearer token, enforced by auth.RequireIdentity inside the handlers.
func (m *Module) Register(api huma.API) {
	bearer := []map[string][]string{{"bearer": {}}}

	huma.Register(api, huma.Operation{
		OperationID: "listFeed",
		Method:      http.MethodGet,
		Path:        "/api/v1/posts",
		Summary:     "List published posts, newest first (cursor paginated)",
		Tags:        []string{tagName},
	}, m.handler.feed)

	huma.Register(api, huma.Operation{
		OperationID:   "recordView",
		Method:        http.MethodPost,
		Path:          "/api/v1/posts/{id}/views",
		Summary:       "Record a view (buffered, fire-and-forget)",
		Tags:          []string{tagName},
		DefaultStatus: http.StatusNoContent,
	}, m.handler.recordView)

	huma.Register(api, huma.Operation{
		OperationID: "searchPosts",
		Method:      http.MethodGet,
		Path:        "/api/v1/search",
		Summary:     "Full-text search over published posts",
		Tags:        []string{tagName},
	}, m.handler.search)

	huma.Register(api, huma.Operation{
		OperationID: "getMyPosts",
		Method:      http.MethodGet,
		Path:        "/api/v1/me/posts",
		Summary:     "List the current user's posts",
		Tags:        []string{tagName},
		Security:    bearer,
	}, m.handler.myPosts)

	huma.Register(api, huma.Operation{
		OperationID: "getPostBySlug",
		Method:      http.MethodGet,
		Path:        "/api/v1/posts/by-slug/{username}/{slug}",
		Summary:     "Get a published post by its public URL",
		Tags:        []string{tagName},
	}, m.handler.getBySlug)

	huma.Register(api, huma.Operation{
		OperationID: "getPost",
		Method:      http.MethodGet,
		Path:        pathByID,
		Summary:     "Get a post by id (for the editor)",
		Tags:        []string{tagName},
	}, m.handler.getByID)

	huma.Register(api, huma.Operation{
		OperationID:   "createPost",
		Method:        http.MethodPost,
		Path:          "/api/v1/posts",
		Summary:       "Create a draft post",
		Tags:          []string{tagName},
		Security:      bearer,
		DefaultStatus: http.StatusCreated,
	}, m.handler.create)

	huma.Register(api, huma.Operation{
		OperationID: "updatePost",
		Method:      http.MethodPatch,
		Path:        pathByID,
		Summary:     "Update a post you authored",
		Tags:        []string{tagName},
		Security:    bearer,
	}, m.handler.update)

	huma.Register(api, huma.Operation{
		OperationID: "deletePost",
		Method:      http.MethodDelete,
		Path:        pathByID,
		Summary:     "Delete a post you authored",
		Tags:        []string{tagName},
		Security:    bearer,
	}, m.handler.delete)

	huma.Register(api, huma.Operation{
		OperationID: "publishPost",
		Method:      http.MethodPost,
		Path:        "/api/v1/posts/{id}/publish",
		Summary:     "Publish a draft",
		Tags:        []string{tagName},
		Security:    bearer,
	}, m.handler.publish)

	huma.Register(api, huma.Operation{
		OperationID: "unpublishPost",
		Method:      http.MethodPost,
		Path:        "/api/v1/posts/{id}/unpublish",
		Summary:     "Return a post to draft",
		Tags:        []string{tagName},
		Security:    bearer,
	}, m.handler.unpublish)
}
