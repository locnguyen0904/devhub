package post

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
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
}

// New builds the module from its dependencies. users and tags arrive as the
// consumer-defined ports, renderer as the markdown port.
func New(db *database.DB, users authorFinder, tags tagLinker, renderer markdownRenderer) *Module {
	svc := newService(db, newRepository(db), users, tags, renderer)
	return &Module{handler: newHandler(svc)}
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
