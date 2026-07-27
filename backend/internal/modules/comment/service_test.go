package comment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/locnguyen0904/devhub/backend/internal/modules/user"
	"github.com/locnguyen0904/devhub/backend/internal/platform/httpx"
)

type fakeStore struct {
	comment Comment
	getErr  error
	updated bool
}

func (f *fakeStore) createTx(context.Context, pgx.Tx, Comment) (Comment, error) {
	return f.comment, nil
}

func (f *fakeStore) getByID(context.Context, uuid.UUID) (Comment, error) {
	return f.comment, f.getErr
}
func (f *fakeStore) listForPost(context.Context, uuid.UUID) ([]Comment, error) { return nil, nil }
func (f *fakeStore) update(_ context.Context, _, _ uuid.UUID, _, html string) (Comment, error) {
	f.updated = true
	c := f.comment
	c.BodyHTML = html
	return c, nil
}

func (f *fakeStore) softDeleteTx(context.Context, pgx.Tx, uuid.UUID, uuid.UUID) (bool, error) {
	return true, nil
}
func (f *fakeStore) adjustCountTx(context.Context, pgx.Tx, uuid.UUID, int32) error { return nil }

type stubUsers struct{}

func (stubUsers) FindBrief(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]user.Brief, error) {
	m := make(map[uuid.UUID]user.Brief, len(ids))
	for _, id := range ids {
		m[id] = user.Brief{ID: id, Username: "loc"}
	}
	return m, nil
}

type stubRenderer struct{}

func (stubRenderer) Render(s string) (string, error) { return "<p>" + s + "</p>", nil }

func newTestService(store *fakeStore, now time.Time) *service {
	return &service{
		repo:     store,
		users:    stubUsers{},
		renderer: stubRenderer{},
		now:      func() time.Time { return now },
	}
}

func TestUpdateRejectsNonAuthor(t *testing.T) {
	author := uuid.New()
	store := &fakeStore{comment: Comment{ID: uuid.New(), AuthorID: author, CreatedAt: time.Now()}}
	svc := newTestService(store, time.Now())

	_, err := svc.Update(context.Background(), uuid.New(), store.comment.ID, "hi there")

	var h *httpx.Error
	if !errors.As(err, &h) || h.Code != httpx.CodeNotFound {
		t.Errorf("Update(non-author) error = %v, want not_found", err)
	}
	if store.updated {
		t.Error("Update(non-author) must not write")
	}
}

func TestUpdateRejectsPastEditWindow(t *testing.T) {
	author := uuid.New()
	posted := time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)
	store := &fakeStore{comment: Comment{ID: uuid.New(), AuthorID: author, CreatedAt: posted}}
	// 31 minutes later — past the 30-minute window.
	svc := newTestService(store, posted.Add(31*time.Minute))

	_, err := svc.Update(context.Background(), author, store.comment.ID, "edited")

	var h *httpx.Error
	if !errors.As(err, &h) || h.Code != httpx.CodeForbidden {
		t.Errorf("Update(past window) error = %v, want forbidden", err)
	}
}

func TestUpdateAllowsWithinWindow(t *testing.T) {
	author := uuid.New()
	posted := time.Date(2026, 7, 24, 10, 0, 0, 0, time.UTC)
	store := &fakeStore{comment: Comment{ID: uuid.New(), AuthorID: author, CreatedAt: posted}}
	svc := newTestService(store, posted.Add(5*time.Minute))

	w, err := svc.Update(context.Background(), author, store.comment.ID, "edited")
	if err != nil {
		t.Fatalf("Update(within window) error = %v", err)
	}
	if !store.updated {
		t.Error("Update(within window) should write")
	}
	if w.Comment.BodyHTML != "<p>edited</p>" {
		t.Errorf("body = %q, want re-rendered", w.Comment.BodyHTML)
	}
}

func TestBuildTreeGroupsAndPrunes(t *testing.T) {
	top := uuid.New()
	topDeleted := uuid.New()
	topDeletedNoReplies := uuid.New()
	authorID := uuid.New()
	reply1 := uuid.New()
	deletedReply := uuid.New()

	comments := []Comment{
		{ID: top, AuthorID: authorID, Depth: 0},
		{ID: reply1, AuthorID: authorID, Depth: 1, ParentID: &top},
		{ID: deletedReply, AuthorID: authorID, Depth: 1, ParentID: &top, Deleted: true},
		{ID: topDeleted, AuthorID: authorID, Depth: 0, Deleted: true},
		{ID: uuid.New(), AuthorID: authorID, Depth: 1, ParentID: &topDeleted},
		{ID: topDeletedNoReplies, AuthorID: authorID, Depth: 0, Deleted: true},
	}
	authors := map[uuid.UUID]user.Brief{authorID: {ID: authorID, Username: "loc"}}

	tree := buildTree(comments, authors)

	// Expect: top (1 visible reply, deleted reply pruned), topDeleted (kept as
	// placeholder because it has a reply). topDeletedNoReplies dropped.
	if len(tree) != 2 {
		t.Fatalf("tree has %d top-level nodes, want 2: %+v", len(tree), tree)
	}
	if tree[0].Comment.ID != top || len(tree[0].Replies) != 1 {
		t.Errorf("first node = %v with %d replies, want top with 1", tree[0].Comment.ID, len(tree[0].Replies))
	}
	if tree[1].Comment.ID != topDeleted || !tree[1].Comment.Deleted {
		t.Errorf("second node = %v, want the deleted top with replies", tree[1].Comment.ID)
	}
}
