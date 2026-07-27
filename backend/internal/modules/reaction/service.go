package reaction

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/locnguyen0904/devhub/backend/internal/platform/database"
	"github.com/locnguyen0904/devhub/backend/internal/platform/httpx"
)

const maxBookmarks = 100

// Service is the engagement API. React/Unreact and Bookmark/Unbookmark back the
// HTTP endpoints; ViewerState and BookmarkedPostIDs are consumed by the post
// module through its own port.
type Service interface {
	React(ctx context.Context, userID, postID uuid.UUID, kind string) (State, error)
	Unreact(ctx context.Context, userID, postID uuid.UUID, kind string) (State, error)
	Bookmark(ctx context.Context, userID, postID uuid.UUID) error
	Unbookmark(ctx context.Context, userID, postID uuid.UUID) error

	// ViewerState reports what the viewer has done to a post: which reaction
	// kinds they gave and whether they bookmarked it.
	ViewerState(ctx context.Context, userID, postID uuid.UUID) (reacted []string, bookmarked bool, err error)
	// BookmarkedPostIDs returns the viewer's saved post ids, newest first.
	BookmarkedPostIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

type service struct {
	db   *database.DB
	repo *repository
}

func newService(db *database.DB, repo *repository) *service {
	return &service{db: db, repo: repo}
}

func (s *service) React(ctx context.Context, userID, postID uuid.UUID, kind string) (State, error) {
	if !isAllowedKind(kind) {
		return State{}, invalidKind()
	}

	txErr := s.db.InTx(ctx, func(tx pgx.Tx) error {
		added, err := s.repo.addReactionTx(ctx, tx, userID, postID, kind)
		if err != nil {
			return err
		}
		// Only move the counter when the reaction was actually new, so a repeat
		// PUT is idempotent.
		if !added {
			return nil
		}
		return s.repo.adjustCountTx(ctx, tx, postID, 1)
	})
	if txErr != nil {
		return State{}, txErr
	}
	return s.state(ctx, userID, postID)
}

func (s *service) Unreact(ctx context.Context, userID, postID uuid.UUID, kind string) (State, error) {
	if !isAllowedKind(kind) {
		return State{}, invalidKind()
	}

	txErr := s.db.InTx(ctx, func(tx pgx.Tx) error {
		removed, err := s.repo.removeReactionTx(ctx, tx, userID, postID, kind)
		if err != nil {
			return err
		}
		if !removed {
			return nil
		}
		return s.repo.adjustCountTx(ctx, tx, postID, -1)
	})
	if txErr != nil {
		return State{}, txErr
	}
	return s.state(ctx, userID, postID)
}

func (s *service) Bookmark(ctx context.Context, userID, postID uuid.UUID) error {
	return s.repo.addBookmark(ctx, userID, postID)
}

func (s *service) Unbookmark(ctx context.Context, userID, postID uuid.UUID) error {
	return s.repo.removeBookmark(ctx, userID, postID)
}

func (s *service) ViewerState(ctx context.Context, userID, postID uuid.UUID) ([]string, bool, error) {
	reacted, err := s.repo.viewerReactions(ctx, userID, postID)
	if err != nil {
		return nil, false, err
	}
	bookmarked, err := s.repo.viewerBookmarked(ctx, userID, postID)
	if err != nil {
		return nil, false, err
	}
	return reacted, bookmarked, nil
}

func (s *service) BookmarkedPostIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	return s.repo.bookmarkedPostIDs(ctx, userID, maxBookmarks)
}

// state reads the fresh count and the viewer's reactions after a toggle.
func (s *service) state(ctx context.Context, userID, postID uuid.UUID) (State, error) {
	count, err := s.repo.reactionCount(ctx, postID)
	if err != nil {
		return State{}, err
	}
	reacted, err := s.repo.viewerReactions(ctx, userID, postID)
	if err != nil {
		return State{}, err
	}
	return State{Count: count, Reacted: reacted}, nil
}

func invalidKind() error {
	return httpx.Invalid("Unknown reaction", map[string]string{
		"kind": "must be like, unicorn or mind_blown",
	})
}
