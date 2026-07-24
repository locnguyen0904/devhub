// Package health reports whether the API and its dependencies can serve
// traffic. It is also the reference shape for every other module: ports,
// repository, service, handler, module.
package health

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/locnguyen0904/devhub/backend/internal/platform/database"
)

// Module is the only place that knows how this module is assembled.
type Module struct {
	handler *Handler
	Service Service // exported so other modules can depend on it later
}

// New wires the module. redis is passed as a Pinger because that is the whole
// of what this module needs from it.
func New(db *database.DB, redis Pinger) *Module {
	svc := newService(newRepository(db), redis)
	return &Module{handler: newHandler(svc), Service: svc}
}

// Register declares this module's operations on the huma API. Registration is
// startup code rather than comments, so the OpenAPI spec cannot describe a
// route the server does not actually serve.
func (m *Module) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "getLiveness",
		Method:      http.MethodGet,
		Path:        "/api/v1/healthz",
		Summary:     "Liveness probe",
		Tags:        []string{"health"},
	}, m.handler.live)

	huma.Register(api, huma.Operation{
		OperationID: "getReadiness",
		Method:      http.MethodGet,
		Path:        "/api/v1/readyz",
		Summary:     "Readiness probe",
		Description: "Returns 503 with code service_unavailable when a dependency does not answer.",
		Tags:        []string{"health"},
	}, m.handler.ready)
}
