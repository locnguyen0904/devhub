package post

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/locnguyen0904/devhub/backend/internal/modules/tag"
	"github.com/locnguyen0904/devhub/backend/internal/modules/user"
	"github.com/locnguyen0904/devhub/backend/internal/platform/database/sqlcgen"
	"github.com/locnguyen0904/devhub/backend/internal/platform/httpx"
)

// fakeStore serves one configurable post and records publish calls. Methods the
// tests below do not exercise return zero values.
type fakeStore struct {
	post          Post
	getErr        error
	publishCalled bool
}

func (f *fakeStore) createTx(context.Context, pgx.Tx, sqlcgen.CreatePostTxParams) (Post, error) {
	return f.post, nil
}
func (f *fakeStore) getByID(context.Context, uuid.UUID) (Post, error) { return f.post, f.getErr }
func (f *fakeStore) getPublishedBySlug(context.Context, string, string) (Post, error) {
	return f.post, f.getErr
}
func (f *fakeStore) slugTaken(context.Context, uuid.UUID, string) (bool, error) { return false, nil }
func (f *fakeStore) update(context.Context, sqlcgen.UpdatePostParams) (Post, error) {
	return f.post, nil
}

func (f *fakeStore) publish(_ context.Context, _, _ uuid.UUID, _ string) (Post, error) {
	f.publishCalled = true
	p := f.post
	p.Status = StatusPublished
	return p, nil
}

func (f *fakeStore) unpublish(context.Context, uuid.UUID, uuid.UUID) (Post, error) {
	return f.post, nil
}
func (f *fakeStore) softDelete(context.Context, uuid.UUID, uuid.UUID) error { return nil }
func (f *fakeStore) feed(context.Context, *feedCursor, int32) ([]Post, error) {
	return []Post{f.post}, nil
}

func (f *fakeStore) myPosts(context.Context, uuid.UUID, string, int32) ([]Post, error) {
	return []Post{f.post}, nil
}

type stubUsers struct{}

func (stubUsers) FindBrief(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]user.Brief, error) {
	m := make(map[uuid.UUID]user.Brief, len(ids))
	for _, id := range ids {
		m[id] = user.Brief{ID: id, Username: "loc"}
	}
	return m, nil
}

type stubTags struct{}

func (stubTags) EnsureAndAttachTx(context.Context, pgx.Tx, uuid.UUID, []string) ([]tag.Tag, error) {
	return nil, nil
}

func (stubTags) TagsForPosts(context.Context, []uuid.UUID) (map[uuid.UUID][]tag.Tag, error) {
	return map[uuid.UUID][]tag.Tag{}, nil
}

func newTestService(store postStore) *service {
	return &service{repo: store, users: stubUsers{}, tags: stubTags{}}
}

func TestPublishRejectsNonAuthor(t *testing.T) {
	author := uuid.New()
	stranger := uuid.New()
	store := &fakeStore{post: Post{ID: uuid.New(), AuthorID: author, Title: "T", Status: StatusDraft}}
	svc := newTestService(store)

	_, err := svc.Publish(context.Background(), stranger, store.post.ID)

	// A non-author must see not-found, not forbidden, so draft ids cannot be probed.
	var h *httpx.Error
	if !errors.As(err, &h) || h.Code != httpx.CodeNotFound {
		t.Errorf("Publish(stranger) error = %v, want not_found", err)
	}
	if store.publishCalled {
		t.Error("Publish(stranger) must not reach the publish query")
	}
}

func TestPublishIsIdempotent(t *testing.T) {
	author := uuid.New()
	store := &fakeStore{post: Post{ID: uuid.New(), AuthorID: author, Title: "T", Status: StatusPublished}}
	svc := newTestService(store)

	m, err := svc.Publish(context.Background(), author, store.post.ID)
	if err != nil {
		t.Fatalf("Publish(already published) error = %v", err)
	}
	if store.publishCalled {
		t.Error("Publishing an already-published post must not run the publish query again")
	}
	if m.Post.Status != StatusPublished {
		t.Errorf("status = %q, want %q", m.Post.Status, StatusPublished)
	}
}

func TestPublishRejectsEmptyTitle(t *testing.T) {
	author := uuid.New()
	store := &fakeStore{post: Post{ID: uuid.New(), AuthorID: author, Title: "   ", Status: StatusDraft}}
	svc := newTestService(store)

	_, err := svc.Publish(context.Background(), author, store.post.ID)

	var h *httpx.Error
	if !errors.As(err, &h) || h.Code != httpx.CodeValidationFailed {
		t.Errorf("Publish(empty title) error = %v, want validation_failed", err)
	}
}

func TestPublishSucceedsForAuthor(t *testing.T) {
	author := uuid.New()
	store := &fakeStore{post: Post{ID: uuid.New(), AuthorID: author, Title: "Real title", Status: StatusDraft}}
	svc := newTestService(store)

	m, err := svc.Publish(context.Background(), author, store.post.ID)
	if err != nil {
		t.Fatalf("Publish(author) error = %v", err)
	}
	if !store.publishCalled {
		t.Error("Publish(author) should run the publish query")
	}
	if m.Post.Status != StatusPublished {
		t.Errorf("status = %q, want %q", m.Post.Status, StatusPublished)
	}
}
