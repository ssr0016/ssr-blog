package repository

import (
	"context"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ssr0016/ssr-blog/internal/database"
	"github.com/ssr0016/ssr-blog/internal/model"
)

var (
	// ErrSlugTaken is returned by Create when the slug violates the unique index.
	// Soft-deleted posts keep their slug reserved, so they count as taken.
	ErrSlugTaken = errors.New("post slug already taken")

	// ErrPostNotFound is returned by write methods that match no non-deleted post.
	// Read methods return nil, nil instead.
	ErrPostNotFound = errors.New("post not found")
)

const (
	pgUniqueViolation = "23505"
	slugIndexName     = "idx_posts_slug"
)

type PostRepo struct{ db *database.DB }

func NewPostRepo(db *database.DB) *PostRepo { return &PostRepo{db: db} }

const postColumns = "id, title, slug, content, excerpt, cover_image_url, status, published_at, created_at, updated_at, deleted_at"

// notDeleted is the soft-delete filter. Every SELECT, UPDATE and DELETE on posts must use it:
//
//	.Where(notDeleted())
func notDeleted() sq.Sqlizer {
	return sq.Eq{"deleted_at": nil}
}

// scanPost scans a single post from a row.
func scanPost(row pgx.Row) (*model.Post, error) {
	var p model.Post
	err := row.Scan(&p.ID, &p.Title, &p.Slug, &p.Content, &p.Excerpt, &p.CoverImageURL, &p.Status, &p.PublishedAt, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan post: %w", err)
	}
	return &p, nil
}

// isSlugTaken reports whether err is a unique violation on the slug index.
func isSlugTaken(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation && pgErr.ConstraintName == slugIndexName
}

// Create inserts a post and returns the stored row. The caller supplies the slug, status and
// published_at; Status must be "draft" or "published". A slug collision returns ErrSlugTaken.
func (r *PostRepo) Create(ctx context.Context, p model.Post) (*model.Post, error) {
	query, args, err := r.db.Builder.
		Insert("posts").
		Columns("title", "slug", "content", "excerpt", "cover_image_url", "status", "published_at").
		Values(p.Title, p.Slug, p.Content, p.Excerpt, p.CoverImageURL, p.Status, p.PublishedAt).
		Suffix("RETURNING " + postColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	created, err := scanPost(r.db.Pool.QueryRow(ctx, query, args...))
	if err != nil {
		if isSlugTaken(err) {
			return nil, fmt.Errorf("create post: %w", ErrSlugTaken)
		}
		return nil, fmt.Errorf("create post: %w", err)
	}
	return created, nil
}

// GetByID returns a non-deleted post of any status, or nil, nil if there is none.
func (r *PostRepo) GetByID(ctx context.Context, id int64) (*model.Post, error) {
	query, args, err := r.db.Builder.
		Select(postColumns).
		From("posts").
		Where(sq.Eq{"id": id}).
		Where(notDeleted()).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	return scanPost(r.db.Pool.QueryRow(ctx, query, args...))
}
