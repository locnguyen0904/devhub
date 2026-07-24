package health

import (
	"context"
	"strings"

	"github.com/locnguyen0904/devhub/backend/internal/platform/httpx"
)

// Handler translates between HTTP and the service. It holds no business rules.
type Handler struct {
	svc Service
}

func newHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) live(_ context.Context, _ *struct{}) (*LiveOutput, error) {
	out := &LiveOutput{}
	out.Body.Status = "ok"
	return out, nil
}

func (h *Handler) ready(ctx context.Context, _ *struct{}) (*ReadyOutput, error) {
	status := h.svc.Check(ctx)

	if !status.Ready() {
		return nil, httpx.ToHuma(httpx.Unavailable(
			"dependencies unavailable: "+strings.Join(failedNames(status.Checks), ", "), nil))
	}

	out := &ReadyOutput{}
	out.Body.Status = "ok"
	out.Body.Checks = toCheckDTOs(status.Checks)
	return out, nil
}

func failedNames(checks []Check) []string {
	var names []string
	for _, c := range checks {
		if !c.OK {
			names = append(names, c.Name)
		}
	}
	return names
}
