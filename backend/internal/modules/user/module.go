package user

import "github.com/locnguyen0904/devhub/backend/internal/platform/database"

// Module wires the user module. It registers no HTTP routes in Phase 1 — the
// public /me and /users/{username} endpoints arrive with their own phase. For
// now it exists so auth can provision and look up users through Service.
type Module struct {
	Service Service
}

// New builds the module.
func New(db *database.DB) *Module {
	return &Module{Service: newService(newRepository(db))}
}
