package tag

import (
	"context"
	"regexp"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/locnguyen0904/devhub/backend/internal/platform/httpx"
)

// tagNamePattern matches the tags.name CHECK constraint. Validating here gives a
// clean 422 instead of letting the database constraint surface as a 500.
var tagNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,29}$`)

// Service is the tag API. EnsureAndAttachTx is what post calls while creating a
// post; the read methods back the tag endpoints.
type Service interface {
	// EnsureAndAttachTx creates any missing tags, attaches them to the post, and
	// bumps their post counts — all within the caller's transaction so tagging
	// commits atomically with the post.
	EnsureAndAttachTx(ctx context.Context, tx pgx.Tx, postID uuid.UUID, names []string) ([]Tag, error)
	// TagsForPosts resolves tags for many posts at once.
	TagsForPosts(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]Tag, error)
	Search(ctx context.Context, query string, limit int) ([]Tag, error)
	Popular(ctx context.Context, limit int) ([]Tag, error)
}

type service struct {
	repo *repository
}

func newService(repo *repository) *service {
	return &service{repo: repo}
}

func (s *service) EnsureAndAttachTx(ctx context.Context, tx pgx.Tx, postID uuid.UUID, names []string) ([]Tag, error) {
	if len(names) > maxTagsPerPost {
		return nil, httpx.Invalid("Too many tags", map[string]string{
			"tags": "a post may have at most 4 tags",
		})
	}

	seen := make(map[string]struct{}, len(names))
	tags := make([]Tag, 0, len(names))
	ids := make([]uuid.UUID, 0, len(names))

	for i, name := range names {
		if !tagNamePattern.MatchString(name) {
			return nil, httpx.Invalid("Invalid tag", map[string]string{
				"tags": "tag '" + name + "' must be lowercase letters, digits or hyphens",
			})
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}

		t, err := s.repo.upsertTx(ctx, tx, name)
		if err != nil {
			return nil, err
		}
		if err := s.repo.attachTx(ctx, tx, postID, t.ID, i); err != nil {
			return nil, err
		}
		tags = append(tags, t)
		ids = append(ids, t.ID)
	}

	if len(ids) > 0 {
		if err := s.repo.incrementCountsTx(ctx, tx, ids); err != nil {
			return nil, err
		}
	}
	return tags, nil
}

func (s *service) TagsForPosts(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]Tag, error) {
	if len(postIDs) == 0 {
		return map[uuid.UUID][]Tag{}, nil
	}
	return s.repo.forPosts(ctx, postIDs)
}

func (s *service) Search(ctx context.Context, query string, limit int) ([]Tag, error) {
	return s.repo.search(ctx, query, clampLimit(limit))
}

func (s *service) Popular(ctx context.Context, limit int) ([]Tag, error) {
	return s.repo.popular(ctx, clampLimit(limit))
}

func clampLimit(limit int) int32 {
	const defaultLimit, maxLimit = 10, 50
	if limit <= 0 {
		return defaultLimit
	}
	if limit > maxLimit {
		return maxLimit
	}
	return int32(limit)
}
