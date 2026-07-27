// Package reaction owns reactions and bookmarks — the engagement a reader has
// with a post. It maintains the post's reaction_count and answers the post
// module's questions about a viewer's own reactions and bookmarks.
package reaction

// Reaction kinds. A single post carries a total reaction_count across kinds.
const (
	KindLike      = "like"
	KindUnicorn   = "unicorn"
	KindMindBlown = "mind_blown"
)

// isAllowedKind reports whether kind is a real reaction. Validated before any
// write so a bad kind is a clean 422 rather than a database constraint error.
func isAllowedKind(kind string) bool {
	switch kind {
	case KindLike, KindUnicorn, KindMindBlown:
		return true
	default:
		return false
	}
}

// State is a post's reaction total plus the kinds the current viewer has given,
// returned after a toggle so the client can reconcile its optimistic update.
type State struct {
	Count   int
	Reacted []string
}
