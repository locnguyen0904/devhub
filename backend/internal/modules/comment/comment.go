// Package comment owns the comments table: a two-level discussion tree per post.
package comment

import (
	"time"

	"github.com/google/uuid"
)

// editWindow is how long after posting a comment may still be edited. Past it,
// edits are refused so a reply is not changed out from under the people who
// already responded to it.
const editWindow = 30 * time.Minute

// maxDepth is the deepest a comment may nest. Two levels (0 and 1) keep the
// whole tree fetchable in one query and readable on a phone.
const maxDepth = 1

// Comment is the domain model. Deleted marks a soft-deleted comment, which is
// kept in the tree only while it still has replies to anchor.
type Comment struct {
	ID           uuid.UUID
	PostID       uuid.UUID
	AuthorID     uuid.UUID
	ParentID     *uuid.UUID
	BodyMarkdown string
	BodyHTML     string
	Depth        int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Deleted      bool
}

// CreateParams is the validated input to post a comment.
type CreateParams struct {
	PostID       uuid.UUID
	ParentID     *uuid.UUID // nil for a top-level comment
	BodyMarkdown string
}
