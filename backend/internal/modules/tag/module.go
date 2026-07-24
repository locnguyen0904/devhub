package tag

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/locnguyen0904/devhub/backend/internal/platform/database"
)

// tagName groups the tag operations in the OpenAPI spec.
const tagName = "tags"

// Module wires the tag module.
type Module struct {
	handler *Handler
	Service Service
}

// New builds the module.
func New(db *database.DB) *Module {
	svc := newService(newRepository(db))
	return &Module{handler: newHandler(svc), Service: svc}
}

// Register mounts the public tag endpoints.
func (m *Module) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "listTags",
		Method:      http.MethodGet,
		Path:        "/api/v1/tags",
		Summary:     "Autocomplete tags, or list popular tags when q is empty",
		Tags:        []string{tagName},
	}, m.handler.list)
}
