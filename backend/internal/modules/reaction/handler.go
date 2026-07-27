package reaction

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/locnguyen0904/devhub/backend/internal/modules/auth"
	"github.com/locnguyen0904/devhub/backend/internal/platform/httpx"
)

// Handler translates HTTP to the service.
type Handler struct {
	svc Service
}

func newHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// ReactInput targets a post and a reaction kind from the path.
type ReactInput struct {
	ID   string `path:"id"`
	Kind string `path:"kind"`
}

// ReactionStateOutput returns the fresh total and the viewer's own reactions so
// the client can reconcile an optimistic update.
type ReactionStateOutput struct {
	Body struct {
		ReactionCount int      `json:"reaction_count"`
		ViewerReacted []string `json:"viewer_reacted" nullable:"false"`
	}
}

// BookmarkInput targets a post from the path.
type BookmarkInput struct {
	ID string `path:"id"`
}

func (h *Handler) react(ctx context.Context, in *ReactInput) (*ReactionStateOutput, error) {
	actor, postID, err := h.actorAndPost(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	state, err := h.svc.React(ctx, actor, postID, in.Kind)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	return stateOutput(state), nil
}

func (h *Handler) unreact(ctx context.Context, in *ReactInput) (*ReactionStateOutput, error) {
	actor, postID, err := h.actorAndPost(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	state, err := h.svc.Unreact(ctx, actor, postID, in.Kind)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	return stateOutput(state), nil
}

func (h *Handler) bookmark(ctx context.Context, in *BookmarkInput) (*struct{}, error) {
	actor, postID, err := h.actorAndPost(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if err := h.svc.Bookmark(ctx, actor, postID); err != nil {
		return nil, httpx.ToHuma(err)
	}
	return nil, nil
}

func (h *Handler) unbookmark(ctx context.Context, in *BookmarkInput) (*struct{}, error) {
	actor, postID, err := h.actorAndPost(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	if err := h.svc.Unbookmark(ctx, actor, postID); err != nil {
		return nil, httpx.ToHuma(err)
	}
	return nil, nil
}

func (h *Handler) actorAndPost(ctx context.Context, rawID string) (uuid.UUID, uuid.UUID, error) {
	identity, err := auth.RequireIdentity(ctx)
	if err != nil {
		return uuid.Nil, uuid.Nil, httpx.ToHuma(err)
	}
	postID, err := uuid.Parse(rawID)
	if err != nil {
		return uuid.Nil, uuid.Nil, httpx.ToHuma(httpx.New(httpx.CodeInvalidRequest, "Invalid post id", fmt.Errorf("parse post id: %w", err)))
	}
	return identity.UserID, postID, nil
}

func stateOutput(state State) *ReactionStateOutput {
	out := &ReactionStateOutput{}
	out.Body.ReactionCount = state.Count
	out.Body.ViewerReacted = state.Reacted
	if out.Body.ViewerReacted == nil {
		out.Body.ViewerReacted = []string{}
	}
	return out
}
