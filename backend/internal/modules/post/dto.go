package post

// AuthorView is the post author as sent to clients.
type AuthorView struct {
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

// TagView mirrors the tag module's client shape, duplicated to avoid a handler
// dependency on that package's DTO.
type TagView struct {
	Name     string  `json:"name"`
	ColorKey *string `json:"color_key,omitempty"`
}

// StatsView holds the post's counters.
type StatsView struct {
	Reactions int   `json:"reactions"`
	Comments  int   `json:"comments"`
	Views     int64 `json:"views"`
}

// View is the full post payload. The feed replaces body_html with an
// excerpt (see toCardView) to keep list responses small.
type View struct {
	ID             string           `json:"id"`
	Slug           string           `json:"slug"`
	URL            string           `json:"url"`
	Title          string           `json:"title"`
	Subtitle       *string          `json:"subtitle,omitempty"`
	BodyHTML       string           `json:"body_html,omitempty"`
	BodyMarkdown   string           `json:"body_markdown,omitempty"`
	Excerpt        string           `json:"excerpt,omitempty"`
	CoverImageURL  *string          `json:"cover_image_url,omitempty"`
	Status         string           `json:"status"`
	ReadingMinutes int              `json:"reading_minutes"`
	Tags           []TagView        `json:"tags" nullable:"false"`
	Author         AuthorView       `json:"author"`
	Stats          StatsView        `json:"stats"`
	Viewer         *ViewerStateView `json:"viewer_state,omitempty"`
	PublishedAt    *string          `json:"published_at,omitempty"`
	UpdatedAt      string           `json:"updated_at"`
}

// ViewerStateView is the signed-in reader's own engagement with a post. Present
// only on single-post reads by an authenticated viewer.
type ViewerStateView struct {
	Reacted    []string `json:"reacted" nullable:"false"`
	Bookmarked bool     `json:"bookmarked"`
}

// --- Create ---

// CreateInput is the body for creating a draft post.
type CreateInput struct {
	Body struct {
		Title        string   `json:"title" minLength:"1" maxLength:"200"`
		Subtitle     *string  `json:"subtitle,omitempty" maxLength:"300"`
		BodyMarkdown string   `json:"body_markdown" maxLength:"200000"`
		CoverImage   *string  `json:"cover_image_url,omitempty"`
		CanonicalURL *string  `json:"canonical_url,omitempty"`
		Tags         []string `json:"tags,omitempty" maxItems:"4"`
	}
}

// Output wraps a single post response.
type Output struct {
	Body View
}

// --- Update ---

// UpdateInput is the partial body for updating a post.
type UpdateInput struct {
	ID   string `path:"id"`
	Body struct {
		Title        *string `json:"title,omitempty" maxLength:"200"`
		Subtitle     *string `json:"subtitle,omitempty" maxLength:"300"`
		BodyMarkdown *string `json:"body_markdown,omitempty" maxLength:"200000"`
		CoverImage   *string `json:"cover_image_url,omitempty"`
	}
}

// --- ID / slug lookups ---

// IDInput carries a post id path parameter.
type IDInput struct {
	ID string `path:"id"`
}

// SlugInput carries the public username/slug path parameters.
type SlugInput struct {
	Username string `path:"username"`
	Slug     string `path:"slug"`
}

// --- Feed ---

// FeedInput is the cursor-paginated feed query.
type FeedInput struct {
	Sort   string `query:"sort" enum:"latest,hot" doc:"latest (default) or hot"`
	Tag    string `query:"tag" doc:"Filter latest by tag name"`
	Cursor string `query:"cursor"`
	Limit  int    `query:"limit" doc:"Page size (default 20, max 50)"`
}

// SearchInput is the full-text search query.
type SearchInput struct {
	Query string `query:"q" doc:"Search text, at least 2 characters"`
	Limit int    `query:"limit"`
}

// SearchOutput is the list of matches, each with a highlighted snippet.
type SearchOutput struct {
	Body struct {
		Data []SearchHitView `json:"data" nullable:"false"`
	}
}

// SearchHitView is one search result: the post card plus its snippet.
type SearchHitView struct {
	Post     View   `json:"post"`
	Headline string `json:"headline" doc:"Snippet with matched terms wrapped in <b>"`
}

// FeedOutput is a page of the feed with its next cursor.
type FeedOutput struct {
	Body struct {
		Data []View `json:"data" nullable:"false"`
		Page struct {
			NextCursor string `json:"next_cursor"`
			HasMore    bool   `json:"has_more"`
		} `json:"page"`
	}
}

// --- My posts ---

// MyPostsInput queries the current user's posts.
type MyPostsInput struct {
	Status string `query:"status" doc:"draft, published, or all (default all)"`
	Limit  int    `query:"limit"`
}

// CardListOutput is a plain list of post cards.
type CardListOutput struct {
	Body struct {
		Data []View `json:"data" nullable:"false"`
	}
}

// --- Delete ---

// DeleteOutput acknowledges a delete.
type DeleteOutput struct {
	Body struct {
		Status string `json:"status" example:"ok"`
	}
}
