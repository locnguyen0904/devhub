package comment

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/locnguyen0904/devhub/backend/internal/platform/database"
)

const tagName = "comments"

// Module wires the comment module.
type Module struct {
	handler *Handler
}

// New builds the module. users and renderer arrive as consumer-defined ports.
func New(db *database.DB, users authorFinder, renderer markdownRenderer) *Module {
	svc := newService(db, newRepository(db), users, renderer)
	return &Module{handler: newHandler(svc)}
}

// Register mounts the comment endpoints. Reads are public; writes need a bearer
// token, enforced by auth.RequireIdentity inside the handlers.
func (m *Module) Register(api huma.API) {
	bearer := []map[string][]string{{"bearer": {}}}

	huma.Register(api, huma.Operation{
		OperationID: "getComments",
		Method:      http.MethodGet,
		Path:        "/api/v1/posts/{id}/comments",
		Summary:     "Get a post's comment tree",
		Tags:        []string{tagName},
	}, m.handler.tree)

	huma.Register(api, huma.Operation{
		OperationID:   "createComment",
		Method:        http.MethodPost,
		Path:          "/api/v1/posts/{id}/comments",
		Summary:       "Comment on a post, or reply to a comment",
		Tags:          []string{tagName},
		Security:      bearer,
		DefaultStatus: http.StatusCreated,
	}, m.handler.create)

	huma.Register(api, huma.Operation{
		OperationID: "updateComment",
		Method:      http.MethodPatch,
		Path:        "/api/v1/comments/{id}",
		Summary:     "Edit a comment within 30 minutes of posting",
		Tags:        []string{tagName},
		Security:    bearer,
	}, m.handler.update)

	huma.Register(api, huma.Operation{
		OperationID: "deleteComment",
		Method:      http.MethodDelete,
		Path:        "/api/v1/comments/{id}",
		Summary:     "Delete a comment you authored",
		Tags:        []string{tagName},
		Security:    bearer,
	}, m.handler.delete)
}
