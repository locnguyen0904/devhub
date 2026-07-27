package post

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gosimple/slug"
	"github.com/jackc/pgx/v5"
	"github.com/locnguyen0904/devhub/backend/internal/modules/tag"
	"github.com/locnguyen0904/devhub/backend/internal/modules/user"
	"github.com/locnguyen0904/devhub/backend/internal/platform/cache"
	"github.com/locnguyen0904/devhub/backend/internal/platform/database"
	"github.com/locnguyen0904/devhub/backend/internal/platform/database/sqlcgen"
	"github.com/locnguyen0904/devhub/backend/internal/platform/httpx"
	"github.com/locnguyen0904/devhub/backend/internal/platform/markdown"
	"github.com/locnguyen0904/devhub/backend/internal/platform/random"
)

// Feed orderings accepted by FeedQuery.Sort.
const (
	SortLatest = "latest"
	SortHot    = "hot"
)

const (
	defaultFeedLimit = 20
	maxFeedLimit     = 50
	slugRetries      = 5

	// hotCacheKey is a prefix; the limit is appended so pages of different sizes
	// do not share one cache entry.
	hotCacheKey = "feed:hot"
	hotCacheTTL = 5 * time.Minute
	minQueryLen = 2

	// viewsKey is the Redis hash buffering pending view increments per post id.
	viewsKey = "post:views"
	// viewFlushInterval is how often buffered views are written to Postgres.
	viewFlushInterval = 60 * time.Second
)

// WithMeta is a post enriched with its author and tags, ready for a response.
type WithMeta struct {
	Post   Post
	Author user.Brief
	Tags   []tag.Tag
}

// postStore is the persistence the service needs, behind an interface so the
// ownership and publish rules can be unit-tested without a database.
type postStore interface {
	createTx(ctx context.Context, tx pgx.Tx, p sqlcgen.CreatePostTxParams) (Post, error)
	getByID(ctx context.Context, id uuid.UUID) (Post, error)
	getPublishedBySlug(ctx context.Context, username, slug string) (Post, error)
	slugTaken(ctx context.Context, authorID uuid.UUID, slug string) (bool, error)
	update(ctx context.Context, p sqlcgen.UpdatePostParams) (Post, error)
	publish(ctx context.Context, id, authorID uuid.UUID, slug string) (Post, error)
	unpublish(ctx context.Context, id, authorID uuid.UUID) (Post, error)
	softDelete(ctx context.Context, id, authorID uuid.UUID) error
	feed(ctx context.Context, cursor *feedCursor, limit int32) ([]Post, error)
	feedByTag(ctx context.Context, tag string, cursor *feedCursor, limit int32) ([]Post, error)
	hotFeed(ctx context.Context, limit int32) ([]Post, error)
	search(ctx context.Context, query string, limit int32) ([]SearchHit, error)
	myPosts(ctx context.Context, authorID uuid.UUID, statusFilter string, limit int32) ([]Post, error)
	addViews(ctx context.Context, ids []uuid.UUID, deltas []int64) error
}

// Service is the post API.
type Service interface {
	Create(ctx context.Context, actorID uuid.UUID, p CreateParams) (WithMeta, error)
	Update(ctx context.Context, actorID, postID uuid.UUID, p UpdateParams) (WithMeta, error)
	Publish(ctx context.Context, actorID, postID uuid.UUID) (WithMeta, error)
	Unpublish(ctx context.Context, actorID, postID uuid.UUID) (WithMeta, error)
	Delete(ctx context.Context, actorID, postID uuid.UUID) error
	GetByID(ctx context.Context, postID uuid.UUID) (WithMeta, error)
	GetBySlug(ctx context.Context, username, slug string) (WithMeta, error)
	Feed(ctx context.Context, q FeedQuery) ([]WithMeta, string, error)
	Search(ctx context.Context, query string, limit int) ([]SearchResult, error)
	MyPosts(ctx context.Context, actorID uuid.UUID, statusFilter string, limit int) ([]WithMeta, error)
	// RecordView buffers one view in Redis; the flusher writes it to Postgres.
	RecordView(ctx context.Context, postID uuid.UUID) error
	// FlushViews drains the buffered views into Postgres. Called on a timer.
	FlushViews(ctx context.Context) error
}

// FeedQuery selects and pages the feed.
type FeedQuery struct {
	Sort   string // "latest" (default) or "hot"
	Tag    string // optional; filters latest by tag
	Cursor string
	Limit  int
}

// SearchResult is a matched post with its highlighted snippet.
type SearchResult struct {
	WithMeta
	Headline string
}

type service struct {
	db       *database.DB
	repo     postStore
	users    authorFinder
	tags     tagLinker
	renderer markdownRenderer
	cache    *cache.Client
	log      *slog.Logger
}

func newService(db *database.DB, repo postStore, users authorFinder, tags tagLinker, renderer markdownRenderer, c *cache.Client, log *slog.Logger) *service {
	return &service{db: db, repo: repo, users: users, tags: tags, renderer: renderer, cache: c, log: log}
}

func (s *service) Create(ctx context.Context, actorID uuid.UUID, p CreateParams) (WithMeta, error) {
	if strings.TrimSpace(p.Title) == "" {
		return WithMeta{}, httpx.Invalid("Title is required", map[string]string{"title": "must not be empty"})
	}

	html, err := s.renderer.Render(p.BodyMarkdown)
	if err != nil {
		return WithMeta{}, err
	}

	slugValue, err := s.uniqueSlug(ctx, actorID, p.Title, "")
	if err != nil {
		return WithMeta{}, err
	}

	var (
		created Post
		tags    []tag.Tag
	)
	txErr := s.db.InTx(ctx, func(tx pgx.Tx) error {
		created, err = s.repo.createTx(ctx, tx, sqlcgen.CreatePostTxParams{
			ID:             uuid.Must(uuid.NewV7()),
			AuthorID:       actorID,
			Slug:           slugValue,
			Title:          p.Title,
			Subtitle:       p.Subtitle,
			BodyMarkdown:   p.BodyMarkdown,
			BodyHtml:       html,
			CoverImageUrl:  p.CoverImage,
			ReadingMinutes: int32(markdown.ReadingMinutes(p.BodyMarkdown)),
			CanonicalUrl:   p.CanonicalURL,
		})
		if err != nil {
			return err
		}
		tags, err = s.tags.EnsureAndAttachTx(ctx, tx, created.ID, p.Tags)
		return err
	})
	if txErr != nil {
		return WithMeta{}, txErr
	}

	author, err := s.oneAuthor(ctx, actorID)
	if err != nil {
		return WithMeta{}, err
	}
	return WithMeta{Post: created, Author: author, Tags: tags}, nil
}

func (s *service) Update(ctx context.Context, actorID, postID uuid.UUID, p UpdateParams) (WithMeta, error) {
	params := sqlcgen.UpdatePostParams{ID: postID, AuthorID: actorID}
	params.Title = p.Title
	params.Subtitle = p.Subtitle
	params.CoverImageUrl = p.CoverImage

	// Re-render only when the body changed; body_html must never drift from
	// body_markdown, so the two are always written together.
	if p.BodyMarkdown != nil {
		html, err := s.renderer.Render(*p.BodyMarkdown)
		if err != nil {
			return WithMeta{}, err
		}
		minutes := int32(markdown.ReadingMinutes(*p.BodyMarkdown))
		params.BodyMarkdown = p.BodyMarkdown
		params.BodyHtml = &html
		params.ReadingMinutes = &minutes
	}

	updated, err := s.repo.update(ctx, params)
	if err != nil {
		return WithMeta{}, err
	}
	return s.enrichOne(ctx, updated)
}

// Publish is idempotent: publishing an already-published post returns it
// unchanged rather than erroring, so a retried request is safe.
func (s *service) Publish(ctx context.Context, actorID, postID uuid.UUID) (WithMeta, error) {
	current, err := s.repo.getByID(ctx, postID)
	if err != nil {
		return WithMeta{}, err
	}
	if current.AuthorID != actorID {
		// Reported as not-found so a caller cannot probe others' draft ids.
		return WithMeta{}, httpx.NotFound("Post not found")
	}
	if current.Status == StatusPublished {
		return s.enrichOne(ctx, current)
	}
	if strings.TrimSpace(current.Title) == "" {
		return WithMeta{}, httpx.Invalid("Cannot publish without a title", map[string]string{"title": "must not be empty"})
	}

	published, err := s.repo.publish(ctx, postID, actorID, current.Slug)
	if err != nil {
		return WithMeta{}, err
	}
	return s.enrichOne(ctx, published)
}

func (s *service) Unpublish(ctx context.Context, actorID, postID uuid.UUID) (WithMeta, error) {
	unpublished, err := s.repo.unpublish(ctx, postID, actorID)
	if err != nil {
		return WithMeta{}, err
	}
	return s.enrichOne(ctx, unpublished)
}

func (s *service) Delete(ctx context.Context, actorID, postID uuid.UUID) error {
	return s.repo.softDelete(ctx, postID, actorID)
}

func (s *service) GetByID(ctx context.Context, postID uuid.UUID) (WithMeta, error) {
	p, err := s.repo.getByID(ctx, postID)
	if err != nil {
		return WithMeta{}, err
	}
	return s.enrichOne(ctx, p)
}

func (s *service) GetBySlug(ctx context.Context, username, slugValue string) (WithMeta, error) {
	p, err := s.repo.getPublishedBySlug(ctx, username, slugValue)
	if err != nil {
		return WithMeta{}, err
	}
	return s.enrichOne(ctx, p)
}

// Feed serves the latest or hot feed, optionally filtered by tag.
func (s *service) Feed(ctx context.Context, q FeedQuery) ([]WithMeta, string, error) {
	if q.Sort == SortHot {
		return s.hotFeed(ctx, clampFeedLimit(q.Limit))
	}
	return s.latestFeed(ctx, q.Tag, q.Cursor, clampFeedLimit(q.Limit))
}

// hotFeed returns the ranked feed, cached in Redis so the ranking scan runs at
// most once per TTL. Hot is a single page — the score changes with time, so a
// stable cursor is not meaningful.
func (s *service) hotFeed(ctx context.Context, limit int32) ([]WithMeta, string, error) {
	// Key by limit: a page cached at limit=20 must not be served to a request
	// asking for limit=50, which would get fewer rows than it asked for.
	key := fmt.Sprintf("%s:%d", hotCacheKey, limit)
	if cached, ok := s.readHotCache(ctx, key); ok {
		return cached, "", nil
	}

	posts, err := s.repo.hotFeed(ctx, limit)
	if err != nil {
		return nil, "", err
	}
	enriched, err := s.enrich(ctx, posts)
	if err != nil {
		return nil, "", err
	}
	s.writeHotCache(ctx, key, enriched)
	return enriched, "", nil
}

func (s *service) latestFeed(ctx context.Context, tagName, rawCursor string, limit int32) ([]WithMeta, string, error) {
	cursor, err := decodeCursor(rawCursor)
	if err != nil {
		return nil, "", err
	}

	var posts []Post
	if tagName != "" {
		posts, err = s.repo.feedByTag(ctx, tagName, cursor, limit)
	} else {
		posts, err = s.repo.feed(ctx, cursor, limit)
	}
	if err != nil {
		return nil, "", err
	}

	enriched, err := s.enrich(ctx, posts)
	if err != nil {
		return nil, "", err
	}

	// The next cursor points at the last row returned; empty when the page did
	// not fill, signalling the end of the feed.
	var next string
	if int32(len(posts)) == limit {
		last := posts[len(posts)-1]
		if last.PublishedAt != nil {
			next = encodeCursor(feedCursor{publishedAt: *last.PublishedAt, id: last.ID})
		}
	}
	return enriched, next, nil
}

// Search runs a full-text query and returns matches with highlighted snippets.
func (s *service) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	query = strings.TrimSpace(query)
	if len(query) < minQueryLen {
		return nil, httpx.Invalid("Query too short", map[string]string{"q": "must be at least 2 characters"})
	}

	hits, err := s.repo.search(ctx, query, clampFeedLimit(limit))
	if err != nil {
		return nil, err
	}
	if len(hits) == 0 {
		return []SearchResult{}, nil
	}

	posts := make([]Post, 0, len(hits))
	for _, h := range hits {
		posts = append(posts, h.Post)
	}
	enriched, err := s.enrich(ctx, posts)
	if err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(enriched))
	for i, m := range enriched {
		// Sanitize the snippet here: it comes from raw markdown, so only the <b>
		// match markers are safe to render.
		results = append(results, SearchResult{
			WithMeta: m,
			Headline: s.renderer.SanitizeHeadline(hits[i].Headline),
		})
	}
	return results, nil
}

// RecordView buffers a view in Redis rather than writing to Postgres per hit,
// so a popular post does not hammer the database. It is fire-and-forget from
// the client's side.
func (s *service) RecordView(ctx context.Context, postID uuid.UUID) error {
	return s.cache.HIncrBy(ctx, viewsKey, postID.String(), 1)
}

// FlushViews drains the buffered view counts and applies them in one UPDATE.
// A drain-then-apply crash loses that window's views, which the counter's
// tolerance allows.
func (s *service) FlushViews(ctx context.Context) error {
	drained, err := s.cache.DrainHash(ctx, viewsKey)
	if err != nil {
		return err
	}
	if len(drained) == 0 {
		return nil
	}

	ids := make([]uuid.UUID, 0, len(drained))
	deltas := make([]int64, 0, len(drained))
	for field, n := range drained {
		id, perr := uuid.Parse(field)
		if perr != nil {
			continue
		}
		ids = append(ids, id)
		deltas = append(deltas, n)
	}
	if len(ids) == 0 {
		return nil
	}
	return s.repo.addViews(ctx, ids, deltas)
}

// readHotCache returns the cached hot feed, or ok=false to recompute. A cache
// read failure is treated as a miss so the feed degrades to a live query rather
// than an error.
func (s *service) readHotCache(ctx context.Context, key string) ([]WithMeta, bool) {
	raw, found, err := s.cache.Get(ctx, key)
	if err != nil || !found {
		return nil, false
	}
	var cached []WithMeta
	if err := json.Unmarshal([]byte(raw), &cached); err != nil {
		return nil, false
	}
	return cached, true
}

func (s *service) writeHotCache(ctx context.Context, key string, feed []WithMeta) {
	raw, err := json.Marshal(feed)
	if err != nil {
		return
	}
	if err := s.cache.Set(ctx, key, string(raw), hotCacheTTL); err != nil {
		// A cache write failure must not fail the request; the next reader just
		// recomputes. Log it so a broken Redis is visible.
		s.log.WarnContext(ctx, "write hot feed cache", slog.String("error", err.Error()))
	}
}

func (s *service) MyPosts(ctx context.Context, actorID uuid.UUID, statusFilter string, limit int) ([]WithMeta, error) {
	switch statusFilter {
	case "", "all":
		statusFilter = "all"
	case StatusDraft, StatusPublished:
	default:
		return nil, httpx.Invalid("Invalid status filter", map[string]string{"status": "must be draft, published, or all"})
	}

	posts, err := s.repo.myPosts(ctx, actorID, statusFilter, clampFeedLimit(limit))
	if err != nil {
		return nil, err
	}
	return s.enrich(ctx, posts)
}

// uniqueSlug derives a slug from the title and disambiguates it within the
// author's posts. currentSlug lets an update keep its slug when the title is
// unchanged, avoiding a needless suffix.
func (s *service) uniqueSlug(ctx context.Context, authorID uuid.UUID, title, currentSlug string) (string, error) {
	base := slug.Make(title)
	if base == "" {
		base = "post"
	}
	if base == currentSlug {
		return base, nil
	}

	candidate := base
	for range slugRetries {
		taken, err := s.repo.slugTaken(ctx, authorID, candidate)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
		suffix, err := random.Hex(3)
		if err != nil {
			return "", err
		}
		candidate = base + "-" + suffix
	}
	return candidate, nil
}

// enrich attaches author and tags to a slice of posts using batch lookups, so
// the whole page costs two extra queries rather than two per post.
func (s *service) enrich(ctx context.Context, posts []Post) ([]WithMeta, error) {
	if len(posts) == 0 {
		return []WithMeta{}, nil
	}

	authorIDs := make([]uuid.UUID, 0, len(posts))
	postIDs := make([]uuid.UUID, 0, len(posts))
	for _, p := range posts {
		authorIDs = append(authorIDs, p.AuthorID)
		postIDs = append(postIDs, p.ID)
	}

	authors, err := s.users.FindBrief(ctx, authorIDs)
	if err != nil {
		return nil, err
	}
	tagsByPost, err := s.tags.TagsForPosts(ctx, postIDs)
	if err != nil {
		return nil, err
	}

	out := make([]WithMeta, 0, len(posts))
	for _, p := range posts {
		out = append(out, WithMeta{
			Post:   p,
			Author: authors[p.AuthorID],
			Tags:   tagsByPost[p.ID],
		})
	}
	return out, nil
}

func (s *service) enrichOne(ctx context.Context, p Post) (WithMeta, error) {
	enriched, err := s.enrich(ctx, []Post{p})
	if err != nil {
		return WithMeta{}, err
	}
	return enriched[0], nil
}

func (s *service) oneAuthor(ctx context.Context, id uuid.UUID) (user.Brief, error) {
	briefs, err := s.users.FindBrief(ctx, []uuid.UUID{id})
	if err != nil {
		return user.Brief{}, err
	}
	return briefs[id], nil
}

func clampFeedLimit(limit int) int32 {
	if limit <= 0 {
		return defaultFeedLimit
	}
	if limit > maxFeedLimit {
		return maxFeedLimit
	}
	return int32(limit)
}
