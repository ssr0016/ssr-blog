-- +goose Up
-- +goose StatementBegin
CREATE TABLE posts (
    id              BIGSERIAL PRIMARY KEY,
    title           VARCHAR(200)  NOT NULL,
    slug            VARCHAR(200)  NOT NULL,
    content         TEXT          NOT NULL,
    excerpt         VARCHAR(500)  NOT NULL DEFAULT '',
    cover_image_url VARCHAR(2048) NOT NULL DEFAULT '',
    status          VARCHAR(16)   NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published')),
    published_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

-- Full (not partial) unique index: soft-deleted posts keep their slug reserved.
CREATE UNIQUE INDEX idx_posts_slug ON posts(slug);

-- Public list: published, not deleted, newest first.
CREATE INDEX idx_posts_public_list ON posts(published_at DESC, id DESC)
    WHERE status = 'published' AND deleted_at IS NULL;

-- Admin list: all non-deleted posts, newest first. Status filter applies on this scan.
CREATE INDEX idx_posts_admin_list ON posts(created_at DESC, id DESC)
    WHERE deleted_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS posts;
-- +goose StatementEnd
