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

// recordView buffers a view. It is public and fire-and-forget: an invalid id
// still returns 204 so a broken client cannot learn which posts exist, and a
// buffer failure is swallowed because a missed view is not worth a user error.
func (h *Handler) recordView(ctx context.Context, in *IDInput) (*struct{}, error) {
	id, err := parsePostID(in.ID)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	_ = h.svc.RecordView(ctx, id)
	return nil, nil
}

func (h *Handler) feed(ctx context.Context, in *FeedInput) (*FeedOutput, error) {
	posts, next, err := h.svc.Feed(ctx, FeedQuery{
		Sort:   in.Sort,
		Tag:    in.Tag,
		Cursor: in.Cursor,
		Limit:  in.Limit,
	})
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	out := &FeedOutput{}
	out.Body.Data = toCardViews(posts)
	out.Body.Page.NextCursor = next
	out.Body.Page.HasMore = next != ""
	return out, nil
}

func (h *Handler) search(ctx context.Context, in *SearchInput) (*SearchOutput, error) {
	results, err := h.svc.Search(ctx, in.Query, in.Limit)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	out := &SearchOutput{}
	out.Body.Data = make([]SearchHitView, 0, len(results))
	for _, r := range results {
		out.Body.Data = append(out.Body.Data, SearchHitView{
			Post:     toCardView(r.WithMeta),
			Headline: r.Headline,
		})
	}
	return out, nil
}

func (h *Handler) myPosts(ctx context.Context, in *MyPostsInput) (*CardListOutput, error) {
	actor, err := auth.RequireIdentity(ctx)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	posts, err := h.svc.MyPosts(ctx, actor.UserID, in.Status, in.Limit)
	if err != nil {
		return nil, httpx.ToHuma(err)
	}
	out := &CardListOutput{}
	out.Body.Data = toCardViews(posts)
	return out, nil
}
