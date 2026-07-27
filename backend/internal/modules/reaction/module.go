package reaction

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/locnguyen0904/devhub/backend/internal/platform/database"
)

const tagName = "reactions"

// Module wires the reaction module.
type Module struct {
	handler *Handler
	Service Service // exported for the post module to consume
}

// New builds the module.
func New(db *database.DB) *Module {
	svc := newService(db, newRepository(db))
	return &Module{handler: newHandler(svc), Service: svc}
}

// Register mounts the reaction and bookmark endpoints. All require a bearer
// token — you cannot react on behalf of someone else.
func (m *Module) Register(api huma.API) {
	bearer := []map[string][]string{{"bearer": {}}}

	huma.Register(api, huma.Operation{
		OperationID: "addReaction",
		Method:      http.MethodPut,
		Path:        "/api/v1/posts/{id}/reactions/{kind}",
		Summary:     "React to a post (idempotent)",
		Tags:        []string{tagName},
		Security:    bearer,
	}, m.handler.react)

	huma.Register(api, huma.Operation{
		OperationID: "removeReaction",
		Method:      http.MethodDelete,
		Path:        "/api/v1/posts/{id}/reactions/{kind}",
		Summary:     "Remove your reaction (idempotent)",
		Tags:        []string{tagName},
		Security:    bearer,
	}, m.handler.unreact)

	huma.Register(api, huma.Operation{
		OperationID:   "addBookmark",
		Method:        http.MethodPut,
		Path:          "/api/v1/posts/{id}/bookmark",
		Summary:       "Save a post",
		Tags:          []string{tagName},
		Security:      bearer,
		DefaultStatus: http.StatusNoContent,
	}, m.handler.bookmark)

	huma.Register(api, huma.Operation{
		OperationID:   "removeBookmark",
		Method:        http.MethodDelete,
		Path:          "/api/v1/posts/{id}/bookmark",
		Summary:       "Unsave a post",
		Tags:          []string{tagName},
		Security:      bearer,
		DefaultStatus: http.StatusNoContent,
	}, m.handler.unbookmark)
}
