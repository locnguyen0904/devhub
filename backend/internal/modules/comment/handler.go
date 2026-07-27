package comment

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

func (h *Handler) tree(ctx context.Context, in *PostCommentsInput) (*CommentTreeOutput, error) {
	postID, err := parseID(in.ID, "post")
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	nodes, err := h.svc.Tree(ctx, postID)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	out := &CommentTreeOutput{}
	out.Body.Data = toTreeViews(nodes)
	return out, nil
}

func (h *Handler) create(ctx context.Context, in *CreateCommentInput) (*CommentOutput, error) {
	actor, err := auth.RequireIdentity(ctx)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	postID, err := parseID(in.ID, "post")
	if err != nil {
		return nil, httpx.ToHuma(err)
	}

	var parentID *uuid.UUID
	if in.Body.ParentID != nil {
		pid, perr := parseID(*in.Body.ParentID, "parent")
		if perr != nil {
			return nil, httpx.ToHuma(perr)
		}
		parentID = &pid
	}

	w, err := h.svc.Create(ctx, actor.UserID, CreateParams{
		PostID:       postID,
		ParentID:     parentID,
		BodyMarkdown: in.Body.BodyMarkdown,
	})
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	return &CommentOutput{Body: toSingleView(w)}, nil
}

func (h *Handler) update(ctx context.Context, in *UpdateCommentInput) (*CommentOutput, error) {
	actor, err := auth.RequireIdentity(ctx)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	id, err := parseID(in.ID, "comment")
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	w, err := h.svc.Update(ctx, actor.UserID, id, in.Body.BodyMarkdown)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	return &CommentOutput{Body: toSingleView(w)}, nil
}

func (h *Handler) delete(ctx context.Context, in *CommentIDInput) (*DeleteCommentOutput, error) {
	actor, err := auth.RequireIdentity(ctx)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	id, err := parseID(in.ID, "comment")
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	if err := h.svc.Delete(ctx, actor.UserID, id); err != nil {
		return nil, httpx.ToHuma(err)
	}
	out := &DeleteCommentOutput{}
	out.Body.Status = "ok"
	return out, nil
}

func parseID(raw, what string) (uuid.UUID, error) {
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, httpx.New(httpx.CodeInvalidRequest, "Invalid "+what+" id", fmt.Errorf("parse %s id: %w", what, err))
	}
	return id, nil
}
