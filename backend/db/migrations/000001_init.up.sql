-- Phase 1 schema (Blog/Posts). Design and rationale: docs/02-data-model.md

CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ---------------------------------------------------------------- users

CREATE TABLE users (
    id              UUID PRIMARY KEY,
    username        CITEXT NOT NULL UNIQUE,
    email           CITEXT UNIQUE,
    display_name    TEXT NOT NULL,
    avatar_url      TEXT,
    bio             TEXT CHECK (length(bio) <= 300),
    website_url     TEXT,
    github_username TEXT,
    location        TEXT,
    role            TEXT NOT NULL DEFAULT 'user'
                    CHECK (role IN ('user', 'moderator', 'admin')),
    is_suspended    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT username_format CHECK (username ~ '^[a-zA-Z0-9_-]{3,30}$')
);

CREATE TABLE oauth_accounts (
    id               UUID PRIMARY KEY,
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider         TEXT NOT NULL CHECK (provider IN ('github')),
    provider_user_id TEXT NOT NULL,
    access_token_enc BYTEA,
    scopes           TEXT[],
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (provider, provider_user_id)
);

CREATE INDEX idx_oauth_accounts_user ON oauth_accounts(user_id);

CREATE TABLE refresh_tokens (
    id         UUID PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash BYTEA NOT NULL UNIQUE,
    family_id  UUID NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    user_agent TEXT,
    ip         INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_user   ON refresh_tokens(user_id) WHERE revoked_at IS NULL;
CREATE INDEX idx_refresh_tokens_expiry ON refresh_tokens(expires_at);

-- ---------------------------------------------------------------- posts

CREATE TABLE posts (
    id              UUID PRIMARY KEY,
    author_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    slug            TEXT NOT NULL,
    title           TEXT NOT NULL CHECK (length(title) BETWEEN 1 AND 200),
    subtitle        TEXT CHECK (length(subtitle) <= 300),
    body_markdown   TEXT NOT NULL,
    body_html       TEXT NOT NULL,
    cover_image_url TEXT,
    status          TEXT NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft', 'published', 'unlisted')),
    reading_minutes INT NOT NULL DEFAULT 1,
    canonical_url   TEXT,

    comment_count   INT NOT NULL DEFAULT 0 CHECK (comment_count  >= 0),
    reaction_count  INT NOT NULL DEFAULT 0 CHECK (reaction_count >= 0),
    view_count      BIGINT NOT NULL DEFAULT 0 CHECK (view_count  >= 0),

    published_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ,

    CONSTRAINT published_needs_timestamp
        CHECK (status <> 'published' OR published_at IS NOT NULL),

    -- 'simple' config, not 'english': Postgres ships no Vietnamese dictionary,
    -- and the English stemmer mangles Vietnamese words.
    search_vector tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('simple', coalesce(title, '')),         'A') ||
        setweight(to_tsvector('simple', coalesce(subtitle, '')),      'B') ||
        setweight(to_tsvector('simple', coalesce(body_markdown, '')), 'C')
    ) STORED
);

-- Slug only needs to be unique per author — the URL is /@username/slug.
CREATE UNIQUE INDEX uq_posts_author_slug
    ON posts(author_id, slug) WHERE deleted_at IS NULL;

-- All three below are partial indexes: drafts and deleted posts stay out of the
-- feed index, the hottest query in the whole system.
CREATE INDEX idx_posts_feed
    ON posts(published_at DESC, id DESC)
    WHERE status = 'published' AND deleted_at IS NULL;

CREATE INDEX idx_posts_author_published
    ON posts(author_id, published_at DESC)
    WHERE status = 'published' AND deleted_at IS NULL;

CREATE INDEX idx_posts_author_drafts
    ON posts(author_id, updated_at DESC)
    WHERE status = 'draft' AND deleted_at IS NULL;

CREATE INDEX idx_posts_search ON posts USING GIN(search_vector);

-- ---------------------------------------------------------------- tags

CREATE TABLE tags (
    id          UUID PRIMARY KEY,
    name        CITEXT NOT NULL UNIQUE CHECK (name ~ '^[a-z0-9][a-z0-9-]{0,29}$'),
    description TEXT,
    -- A colour key, not a hex value: no single hex reads well on both light and
    -- dark backgrounds. Palette in docs/06-design-system.md §4.
    color_key   TEXT CHECK (color_key IN
                ('blue', 'violet', 'emerald', 'amber', 'rose', 'cyan', 'orange', 'teal')),
    post_count  INT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tags_name_trgm ON tags USING GIN(name gin_trgm_ops);
CREATE INDEX idx_tags_popular   ON tags(post_count DESC);

CREATE TABLE post_tags (
    post_id  UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    tag_id   UUID NOT NULL REFERENCES tags(id)  ON DELETE CASCADE,
    position SMALLINT NOT NULL DEFAULT 0,
    PRIMARY KEY (post_id, tag_id)
);

-- The primary key serves "tags of a post"; this index serves the reverse,
-- "posts by tag", which is the direction the tag page uses.
CREATE INDEX idx_post_tags_tag ON post_tags(tag_id, post_id);

-- ---------------------------------------------------------------- comments

CREATE TABLE comments (
    id            UUID PRIMARY KEY,
    post_id       UUID NOT NULL REFERENCES posts(id)    ON DELETE CASCADE,
    author_id     UUID NOT NULL REFERENCES users(id)    ON DELETE CASCADE,
    parent_id     UUID          REFERENCES comments(id) ON DELETE CASCADE,
    body_markdown TEXT NOT NULL CHECK (length(body_markdown) BETWEEN 1 AND 5000),
    body_html     TEXT NOT NULL,
    -- Fixed at 2 levels: lets a post's whole comment tree load in one query.
    depth         SMALLINT NOT NULL DEFAULT 0 CHECK (depth IN (0, 1)),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX idx_comments_post   ON comments(post_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_comments_parent ON comments(parent_id) WHERE parent_id IS NOT NULL;

-- ---------------------------------------------------------------- reactions

-- The composite primary key makes "one user, one reaction kind, one post, once"
-- an invariant at the database layer, independent of any check in code.
CREATE TABLE reactions (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_id    UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    kind       TEXT NOT NULL CHECK (kind IN ('like', 'unicorn', 'mind_blown')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, post_id, kind)
);

CREATE INDEX idx_reactions_post ON reactions(post_id);

CREATE TABLE bookmarks (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_id    UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, post_id)
);

CREATE INDEX idx_bookmarks_user ON bookmarks(user_id, created_at DESC);
