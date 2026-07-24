package tag

import (
	"context"

	"github.com/locnguyen0904/devhub/backend/internal/platform/httpx"
)

// Handler serves the public tag endpoints.
type Handler struct {
	svc Service
}

func newHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// Chip is a tag as sent to clients. Named Chip rather than View to keep the
// OpenAPI schema name unique across modules (post has its own View).
type Chip struct {
	Name     string  `json:"name" example:"go"`
	ColorKey *string `json:"color_key,omitempty" doc:"Palette key; nil means derive from name"`
}

// ListInput is the shared query for search and popular.
type ListInput struct {
	Query string `query:"q" doc:"Prefix to autocomplete; empty for popular"`
	Limit int    `query:"limit" doc:"Max results (default 10, max 50)"`
}

// ListOutput is the tag list body.
type ListOutput struct {
	Body struct {
		Tags []Chip `json:"tags" nullable:"false"`
	}
}

func (h *Handler) list(ctx context.Context, in *ListInput) (*ListOutput, error) {
	var (
		tags []Tag
		err  error
	)
	if in.Query == "" {
		tags, err = h.svc.Popular(ctx, in.Limit)
	} else {
		tags, err = h.svc.Search(ctx, in.Query, in.Limit)
	}
	if err != nil {
		return nil, httpx.ToHuma(err)
	}

	out := &ListOutput{}
	out.Body.Tags = toChips(tags)
	return out, nil
}

func toChips(tags []Tag) []Chip {
	chips := make([]Chip, 0, len(tags))
	for _, t := range tags {
		chips = append(chips, Chip{Name: t.Name, ColorKey: t.ColorKey})
	}
	return chips
}
