package comment

import (
	"time"

	"github.com/locnguyen0904/devhub/backend/internal/modules/user"
)

// CommentAuthor is the comment author as sent to clients. Named distinctly from
// the post module's AuthorView so the OpenAPI schema names stay unique.
type CommentAuthor struct {
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

// CommentView is one comment in the tree. A deleted comment has a null body and
// author but keeps its replies, so the thread stays intact.
type CommentView struct {
	ID           string         `json:"id"`
	BodyHTML     *string        `json:"body_html"`
	BodyMarkdown *string        `json:"body_markdown" doc:"Source for the author's edit form; null when deleted"`
	Author       *CommentAuthor `json:"author,omitempty" doc:"Absent on a deleted comment"`
	Deleted      bool           `json:"deleted"`
	CreatedAt    string         `json:"created_at"`
	UpdatedAt    string         `json:"updated_at"`
	Replies      []CommentView  `json:"replies" nullable:"false"`
}

// --- Inputs / outputs ---

// PostCommentsInput reads the post id from the path.
type PostCommentsInput struct {
	ID string `path:"id"`
}

// CommentTreeOutput is the whole discussion for a post.
type CommentTreeOutput struct {
	Body struct {
		Data []CommentView `json:"data" nullable:"false"`
	}
}

// CreateCommentInput posts a comment under a post, optionally replying.
type CreateCommentInput struct {
	ID   string `path:"id"`
	Body struct {
		BodyMarkdown string  `json:"body_markdown" minLength:"1" maxLength:"5000"`
		ParentID     *string `json:"parent_id,omitempty" doc:"Reply target; omit for a top-level comment"`
	}
}

// CommentOutput wraps a single comment.
type CommentOutput struct {
	Body CommentView
}

// UpdateCommentInput edits a comment's body.
type UpdateCommentInput struct {
	ID   string `path:"id"`
	Body struct {
		BodyMarkdown string `json:"body_markdown" minLength:"1" maxLength:"5000"`
	}
}

// CommentIDInput reads a comment id from the path.
type CommentIDInput struct {
	ID string `path:"id"`
}

// DeleteCommentOutput acknowledges a delete.
type DeleteCommentOutput struct {
	Body struct {
		Status string `json:"status" example:"ok"`
	}
}

func toTreeViews(nodes []Node) []CommentView {
	views := make([]CommentView, 0, len(nodes))
	for _, n := range nodes {
		views = append(views, toView(n.Comment, n.Author, toTreeViews(n.Replies)))
	}
	return views
}

func toSingleView(w WithAuthor) CommentView {
	return toView(w.Comment, w.Author, []CommentView{})
}

func toView(c Comment, author user.Brief, replies []CommentView) CommentView {
	v := CommentView{
		ID:        c.ID.String(),
		Deleted:   c.Deleted,
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
		UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
		Replies:   replies,
	}
	if !c.Deleted {
		html := c.BodyHTML
		v.BodyHTML = &html
		md := c.BodyMarkdown
		v.BodyMarkdown = &md
		v.Author = &CommentAuthor{
			Username:    author.Username,
			DisplayName: author.DisplayName,
			AvatarURL:   author.AvatarURL,
		}
	}
	return v
}
