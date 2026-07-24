// Package tag owns the tags and post_tags tables. Tags are created on demand as
// authors type them, so there is no separate "create tag" step.
package tag

import "github.com/google/uuid"

// maxTagsPerPost mirrors Dev.to. More than a handful stops being categorisation
// and starts being keyword spam.
const maxTagsPerPost = 4

// Tag is the domain model. ColorKey is nil until an admin assigns one; the
// frontend then derives a colour from the name.
type Tag struct {
	ID       uuid.UUID
	Name     string
	ColorKey *string
}
