package post

import (
	"context"

	"github.com/google/uuid"
	"github.com/locnguyen0904/devhub/backend/internal/modules/auth"
	"github.com/locnguyen0904/devhub/backend/internal/platform/httpx"
)

// Handler translates HTTP to the service. Ownership is enforced in the service
// and repository; the handler only supplies the acting user id.
type Handler struct {
	svc Service
}

func newHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) create(ctx context.Context, in *CreateInput) (*Output, error) {
	actor, err := auth.RequireIdentity(ctx)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}

	m, err := h.svc.Create(ctx, actor.UserID, CreateParams{
		Title:        in.Body.Title,
		Subtitle:     in.Body.Subtitle,
		BodyMarkdown: in.Body.BodyMarkdown,
		CoverImage:   in.Body.CoverImage,
		CanonicalURL: in.Body.CanonicalURL,
		Tags:         in.Body.Tags,
	})
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	return &Output{Body: toFullView(m)}, nil
}

func (h *Handler) update(ctx context.Context, in *UpdateInput) (*Output, error) {
	actor, err := auth.RequireIdentity(ctx)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	id, err := parsePostID(in.ID)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}

	m, err := h.svc.Update(ctx, actor.UserID, id, UpdateParams{
		Title:        in.Body.Title,
		Subtitle:     in.Body.Subtitle,
		BodyMarkdown: in.Body.BodyMarkdown,
		CoverImage:   in.Body.CoverImage,
	})
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	return &Output{Body: toFullView(m)}, nil
}

func (h *Handler) getByID(ctx context.Context, in *IDInput) (*Output, error) {
	id, err := parsePostID(in.ID)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	m, err := h.svc.GetByID(ctx, id)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	return &Output{Body: toFullView(m)}, nil
}

func (h *Handler) getBySlug(ctx context.Context, in *SlugInput) (*Output, error) {
	m, err := h.svc.GetBySlug(ctx, in.Username, in.Slug)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	return &Output{Body: toFullView(m)}, nil
}

func (h *Handler) publish(ctx context.Context, in *IDInput) (*Output, error) {
	actor, id, err := h.actorAndPost(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	m, err := h.svc.Publish(ctx, actor, id)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	return &Output{Body: toFullView(m)}, nil
}

func (h *Handler) unpublish(ctx context.Context, in *IDInput) (*Output, error) {
	actor, id, err := h.actorAndPost(ctx, in.ID)
	if err != nil {
		return nil, err
	}
	m, err := h.svc.Unpublish(ctx, actor, id)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	return &Output{Body: toFullView(m)}, nil
}

// actorAndPost resolves the authenticated user and the post id together, since
// every owner-scoped handler needs both. The returned error is already a huma
// error, ready to return.
func (h *Handler) actorAndPost(ctx context.Context, rawID string) (actor uuid.UUID, postID uuid.UUID, err error) {
	identity, err := auth.RequireIdentity(ctx)
	if err != nil {
		return uuid.Nil, uuid.Nil, httpx.ToHuma(err)
	}
	postID, err = parsePostID(rawID)
	if err != nil {
		return uuid.Nil, uuid.Nil, httpx.ToHuma(err)
	}
	return identity.UserID, postID, nil
}

func (h *Handler) delete(ctx context.Context, in *IDInput) (*DeleteOutput, error) {
	actor, err := auth.RequireIdentity(ctx)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	id, err := parsePostID(in.ID)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	if err := h.svc.Delete(ctx, actor.UserID, id); err != nil {
		return nil, httpx.ToHuma(err)
	}
	out := &DeleteOutput{}
	out.Body.Status = "ok"
	return out, nil
}

func (h *Handler) feed(ctx context.Context, in *FeedInput) (*FeedOutput, error) {
	posts, next, err := h.svc.Feed(ctx, in.Cursor, in.Limit)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	out := &FeedOutput{}
	out.Body.Data = toCardViews(posts)
	out.Body.Page.NextCursor = next
	out.Body.Page.HasMore = next != ""
	return out, nil
}

func (h *Handler) myPosts(ctx context.Context, in *MyPostsInput) (*ListOutput, error) {
	actor, err := auth.RequireIdentity(ctx)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	posts, err := h.svc.MyPosts(ctx, actor.UserID, in.Status, in.Limit)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	out := &ListOutput{}
	out.Body.Data = toCardViews(posts)
	return out, nil
}
