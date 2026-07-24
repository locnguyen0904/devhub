package post

import (
	"regexp"
	"strings"
	"time"

	"github.com/locnguyen0904/devhub/backend/internal/modules/tag"
)

// excerptLength is how many characters of stripped body the feed shows.
const excerptLength = 200

var htmlTag = regexp.MustCompile(`<[^>]*>`)

// toFullView renders a post with its full HTML body, for the detail and editor
// endpoints.
func toFullView(m WithMeta) View {
	v := baseView(m)
	v.BodyHTML = m.Post.BodyHTML
	// The editor needs the source to edit; the feed card deliberately omits it.
	v.BodyMarkdown = m.Post.BodyMarkdown
	return v
}

// toCardView renders a post for a list, replacing the HTML body with a short
// text excerpt. Sending 20 full bodies would bloat the feed response.
func toCardView(m WithMeta) View {
	v := baseView(m)
	v.Excerpt = excerpt(m.Post.BodyHTML)
	return v
}

func toCardViews(posts []WithMeta) []View {
	views := make([]View, 0, len(posts))
	for _, m := range posts {
		views = append(views, toCardView(m))
	}
	return views
}

func baseView(m WithMeta) View {
	v := View{
		ID:             m.Post.ID.String(),
		Slug:           m.Post.Slug,
		URL:            "/@" + m.Author.Username + "/" + m.Post.Slug,
		Title:          m.Post.Title,
		Subtitle:       m.Post.Subtitle,
		CoverImageURL:  m.Post.CoverImageURL,
		Status:         m.Post.Status,
		ReadingMinutes: m.Post.ReadingMinutes,
		Tags:           toTagViews(m.Tags),
		Author: AuthorView{
			Username:    m.Author.Username,
			DisplayName: m.Author.DisplayName,
			AvatarURL:   m.Author.AvatarURL,
		},
		Stats: StatsView{
			Reactions: m.Post.ReactionCount,
			Comments:  m.Post.CommentCount,
			Views:     m.Post.ViewCount,
		},
		UpdatedAt: m.Post.UpdatedAt.Format(time.RFC3339),
	}
	if m.Post.PublishedAt != nil {
		published := m.Post.PublishedAt.Format(time.RFC3339)
		v.PublishedAt = &published
	}
	return v
}

func toTagViews(tags []tag.Tag) []TagView {
	views := make([]TagView, 0, len(tags))
	for _, t := range tags {
		views = append(views, TagView{Name: t.Name, ColorKey: t.ColorKey})
	}
	return views
}

// excerpt strips HTML tags and truncates to excerptLength on a word boundary.
func excerpt(html string) string {
	text := strings.TrimSpace(htmlTag.ReplaceAllString(html, " "))
	text = strings.Join(strings.Fields(text), " ")
	if len(text) <= excerptLength {
		return text
	}
	trimmed := text[:excerptLength]
	if idx := strings.LastIndex(trimmed, " "); idx > 0 {
		trimmed = trimmed[:idx]
	}
	return trimmed + "…"
}
