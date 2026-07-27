package comment

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/locnguyen0904/devhub/backend/internal/modules/user"
	"github.com/locnguyen0904/devhub/backend/internal/platform/database"
	"github.com/locnguyen0904/devhub/backend/internal/platform/httpx"
)

const (
	minBodyLen = 1
	maxBodyLen = 5000
)

// WithAuthor is a single comment plus its author, returned by create and update.
type WithAuthor struct {
	Comment Comment
	Author  user.Brief
}

// Node is a comment in the tree: the comment, its author, and its replies. A
// deleted node carries a zero Author and an empty body; the handler renders it
// as a placeholder.
type Node struct {
	Comment Comment
	Author  user.Brief
	Replies []Node
}

// commentStore is the persistence the service needs, behind an interface so the
// authorization and edit-window rules can be unit-tested without a database.
type commentStore interface {
	createTx(ctx context.Context, tx pgx.Tx, c Comment) (Comment, error)
	getByID(ctx context.Context, id uuid.UUID) (Comment, error)
	listForPost(ctx context.Context, postID uuid.UUID) ([]Comment, error)
	update(ctx context.Context, id, authorID uuid.UUID, markdown, html string) (Comment, error)
	softDeleteTx(ctx context.Context, tx pgx.Tx, id, authorID uuid.UUID) (bool, error)
	adjustCountTx(ctx context.Context, tx pgx.Tx, postID uuid.UUID, delta int32) error
}

// Service is the comment API.
type Service interface {
	Create(ctx context.Context, actorID uuid.UUID, p CreateParams) (WithAuthor, error)
	Tree(ctx context.Context, postID uuid.UUID) ([]Node, error)
	Update(ctx context.Context, actorID, commentID uuid.UUID, bodyMarkdown string) (WithAuthor, error)
	Delete(ctx context.Context, actorID, commentID uuid.UUID) error
}

type service struct {
	db       *database.DB
	repo     commentStore
	users    authorFinder
	renderer markdownRenderer
	now      func() time.Time
}

func newService(db *database.DB, repo commentStore, users authorFinder, renderer markdownRenderer) *service {
	return &service{db: db, repo: repo, users: users, renderer: renderer, now: time.Now}
}

func (s *service) Create(ctx context.Context, actorID uuid.UUID, p CreateParams) (WithAuthor, error) {
	body := strings.TrimSpace(p.BodyMarkdown)
	if len(body) < minBodyLen || len(body) > maxBodyLen {
		return WithAuthor{}, httpx.Invalid("Invalid comment", map[string]string{
			"body_markdown": "must be between 1 and 5000 characters",
		})
	}

	depth := 0
	if p.ParentID != nil {
		parent, err := s.repo.getByID(ctx, *p.ParentID)
		if err != nil {
			return WithAuthor{}, err
		}
		if parent.Deleted || parent.PostID != p.PostID {
			return WithAuthor{}, httpx.NotFound("Parent comment not found")
		}
		depth = parent.Depth + 1
		if depth > maxDepth {
			return WithAuthor{}, httpx.Invalid("Cannot nest that deep", map[string]string{
				"parent_id": "replies can only be one level deep",
			})
		}
	}

	html, err := s.renderer.Render(body)
	if err != nil {
		return WithAuthor{}, err
	}

	var created Comment
	txErr := s.db.InTx(ctx, func(tx pgx.Tx) error {
		created, err = s.repo.createTx(ctx, tx, Comment{
			ID:           uuid.Must(uuid.NewV7()),
			PostID:       p.PostID,
			AuthorID:     actorID,
			ParentID:     p.ParentID,
			BodyMarkdown: body,
			BodyHTML:     html,
			Depth:        depth,
		})
		if err != nil {
			return err
		}
		return s.repo.adjustCountTx(ctx, tx, p.PostID, 1)
	})
	if txErr != nil {
		return WithAuthor{}, txErr
	}

	author, err := s.oneAuthor(ctx, actorID)
	if err != nil {
		return WithAuthor{}, err
	}
	return WithAuthor{Comment: created, Author: author}, nil
}

func (s *service) Tree(ctx context.Context, postID uuid.UUID) ([]Node, error) {
	comments, err := s.repo.listForPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	if len(comments) == 0 {
		return []Node{}, nil
	}

	authors, err := s.authorsFor(ctx, comments)
	if err != nil {
		return nil, err
	}
	return buildTree(comments, authors), nil
}

func (s *service) Update(ctx context.Context, actorID, commentID uuid.UUID, bodyMarkdown string) (WithAuthor, error) {
	body := strings.TrimSpace(bodyMarkdown)
	if len(body) < minBodyLen || len(body) > maxBodyLen {
		return WithAuthor{}, httpx.Invalid("Invalid comment", map[string]string{
			"body_markdown": "must be between 1 and 5000 characters",
		})
	}

	current, err := s.repo.getByID(ctx, commentID)
	if err != nil {
		return WithAuthor{}, err
	}
	// Hide non-ownership as not-found so comment ids cannot be probed.
	if current.AuthorID != actorID || current.Deleted {
		return WithAuthor{}, httpx.NotFound("Comment not found")
	}
	if s.now().After(current.CreatedAt.Add(editWindow)) {
		return WithAuthor{}, httpx.New(httpx.CodeForbidden, "The edit window has passed", nil)
	}

	html, err := s.renderer.Render(body)
	if err != nil {
		return WithAuthor{}, err
	}

	updated, err := s.repo.update(ctx, commentID, actorID, body, html)
	if err != nil {
		return WithAuthor{}, err
	}
	author, err := s.oneAuthor(ctx, actorID)
	if err != nil {
		return WithAuthor{}, err
	}
	return WithAuthor{Comment: updated, Author: author}, nil
}

func (s *service) Delete(ctx context.Context, actorID, commentID uuid.UUID) error {
	current, err := s.repo.getByID(ctx, commentID)
	if err != nil {
		return err
	}
	if current.AuthorID != actorID {
		return httpx.NotFound("Comment not found")
	}

	return s.db.InTx(ctx, func(tx pgx.Tx) error {
		changed, err := s.repo.softDeleteTx(ctx, tx, commentID, actorID)
		if err != nil {
			return err
		}
		// Only adjust the count when a live comment was actually removed, so a
		// repeated delete stays idempotent.
		if !changed {
			return nil
		}
		return s.repo.adjustCountTx(ctx, tx, current.PostID, -1)
	})
}

// buildTree groups a flat, chronologically-ordered comment list into two levels.
// Deleted replies (leaves) are dropped; a deleted top-level comment is kept only
// while it still has visible replies to anchor.
func buildTree(comments []Comment, authors map[uuid.UUID]user.Brief) []Node {
	repliesByParent := make(map[uuid.UUID][]Comment)
	var tops []Comment
	for _, c := range comments {
		if c.Depth == 0 {
			tops = append(tops, c)
		} else if c.ParentID != nil {
			repliesByParent[*c.ParentID] = append(repliesByParent[*c.ParentID], c)
		}
	}

	nodes := make([]Node, 0, len(tops))
	for _, top := range tops {
		var replies []Node
		for _, reply := range repliesByParent[top.ID] {
			if reply.Deleted {
				continue // a deleted leaf is noise, not a placeholder
			}
			replies = append(replies, Node{Comment: reply, Author: authors[reply.AuthorID]})
		}

		if top.Deleted && len(replies) == 0 {
			continue // deleted and nothing hangs off it
		}
		author := user.Brief{}
		if !top.Deleted {
			author = authors[top.AuthorID]
		}
		nodes = append(nodes, Node{Comment: top, Author: author, Replies: replies})
	}
	return nodes
}

func (s *service) authorsFor(ctx context.Context, comments []Comment) (map[uuid.UUID]user.Brief, error) {
	ids := make([]uuid.UUID, 0, len(comments))
	for _, c := range comments {
		if !c.Deleted {
			ids = append(ids, c.AuthorID)
		}
	}
	if len(ids) == 0 {
		return map[uuid.UUID]user.Brief{}, nil
	}
	return s.users.FindBrief(ctx, ids)
}

func (s *service) oneAuthor(ctx context.Context, id uuid.UUID) (user.Brief, error) {
	briefs, err := s.users.FindBrief(ctx, []uuid.UUID{id})
	if err != nil {
		return user.Brief{}, err
	}
	return briefs[id], nil
}
