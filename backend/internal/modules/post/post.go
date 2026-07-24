// Package post owns the posts table: authoring, publishing, and the feed.
package post

import (
	"time"

	"github.com/google/uuid"
)

// Status values for a post.
const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusUnlisted  = "unlisted"
)

// Post is the domain model, mapped from the sqlc row in the repository.
type Post struct {
	ID             uuid.UUID
	AuthorID       uuid.UUID
	Slug           string
	Title          string
	Subtitle       *string
	BodyMarkdown   string
	BodyHTML       string
	CoverImageURL  *string
	Status         string
	ReadingMinutes int
	CommentCount   int
	ReactionCount  int
	ViewCount      int64
	PublishedAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CreateParams is the validated input to create a post.
type CreateParams struct {
	Title        string
	Subtitle     *string
	BodyMarkdown string
	CoverImage   *string
	CanonicalURL *string
	Tags         []string
}

// UpdateParams carries only the fields a PATCH changes; nil means "leave as is".
type UpdateParams struct {
	Title        *string
	Subtitle     *string
	BodyMarkdown *string
	CoverImage   *string
}
