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

// publishedOnly is the public visibility filter: a post is public only when it is published and
// not soft-deleted. Every public read must use it, so drafts, unpublished and deleted posts are
// excluded by the same predicate that reads by slug and lists.
func publishedOnly() sq.Sqlizer {
	return sq.And{sq.Eq{"status": model.PostStatusPublished}, notDeleted()}
}

// ListPublished returns one page of public post summaries, newest first (published_at DESC, id DESC),
// and the total number of public posts. A page past the end returns an empty, non-nil slice.
func (r *PostRepo) ListPublished(ctx context.Context, page, limit int) ([]model.PostSummary, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if page < 1 {
		page = 1
	}

	countSQL, countArgs, err := r.db.Builder.
		Select("COUNT(*)").
		From("posts").
		Where(publishedOnly()).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count query: %w", err)
	}

	var total int64
	if err := r.db.Pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count published posts: %w", err)
	}

	offset := (page - 1) * limit
	query, args, err := r.db.Builder.
		Select("title", "slug", "excerpt", "cover_image_url", "published_at").
		From("posts").
		Where(publishedOnly()).
		OrderBy("published_at DESC", "id DESC").
		Limit(uint64(limit)).   // #nosec G115 - limit validated
		Offset(uint64(offset)). // #nosec G115 - offset validated
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query published posts: %w", err)
	}
	defer rows.Close()

	items := make([]model.PostSummary, 0, limit)
	for rows.Next() {
		var s model.PostSummary
		if err := rows.Scan(&s.Title, &s.Slug, &s.Excerpt, &s.CoverImageURL, &s.PublishedAt); err != nil {
			return nil, 0, fmt.Errorf("scan post summary: %w", err)
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate published posts: %w", err)
	}
	return items, total, nil
}

// GetPublishedBySlug returns a public post, or nil, nil if there is none. One predicate decides
// visibility, so a draft, a soft-deleted post and a slug that never existed are the same result.
func (r *PostRepo) GetPublishedBySlug(ctx context.Context, slug string) (*model.Post, error) {
	query, args, err := r.db.Builder.
		Select(postColumns).
		From("posts").
		Where(sq.Eq{"slug": slug}).
		Where(publishedOnly()).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	return scanPost(r.db.Pool.QueryRow(ctx, query, args...))
}

// ListAdmin returns one page of non-deleted posts of any status as admin summaries (no content),
// newest first (created_at DESC, id DESC), and the total for the same filter. An empty status
// means all statuses. A page past the end returns an empty, non-nil slice.
func (r *PostRepo) ListAdmin(ctx context.Context, status string, page, limit int) ([]model.AdminPostSummary, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if page < 1 {
		page = 1
	}

	filter := sq.And{notDeleted()}
	if status != "" {
		filter = append(filter, sq.Eq{"status": status})
	}

	countSQL, countArgs, err := r.db.Builder.
		Select("COUNT(*)").
		From("posts").
		Where(filter).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count query: %w", err)
	}

	var total int64
	if err := r.db.Pool.QueryRow(ctx, countSQL, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count posts: %w", err)
	}

	offset := (page - 1) * limit
	query, args, err := r.db.Builder.
		Select("id", "title", "slug", "excerpt", "cover_image_url", "status", "published_at", "created_at", "updated_at").
		From("posts").
		Where(filter).
		OrderBy("created_at DESC", "id DESC").
		Limit(uint64(limit)).   // #nosec G115 - limit validated
		Offset(uint64(offset)). // #nosec G115 - offset validated
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.db.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query posts: %w", err)
	}
	defer rows.Close()

	items := make([]model.AdminPostSummary, 0, limit)
	for rows.Next() {
		var s model.AdminPostSummary
		if err := rows.Scan(&s.ID, &s.Title, &s.Slug, &s.Excerpt, &s.CoverImageURL, &s.Status, &s.PublishedAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan admin post summary: %w", err)
		}
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate posts: %w", err)
	}
	return items, total, nil
}

// Update replaces the editable fields of a non-deleted post (title, content, excerpt, cover image,
// status, published_at) and returns the stored row. It never writes slug: a slug is fixed at creation,
// so p.Slug is ignored. A missing or soft-deleted id returns ErrPostNotFound.
func (r *PostRepo) Update(ctx context.Context, p model.Post) (*model.Post, error) {
	query, args, err := r.db.Builder.
		Update("posts").
		Set("title", p.Title).
		Set("content", p.Content).
		Set("excerpt", p.Excerpt).
		Set("cover_image_url", p.CoverImageURL).
		Set("status", p.Status).
		Set("published_at", p.PublishedAt).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": p.ID}).
		Where(notDeleted()).
		Suffix("RETURNING " + postColumns).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	updated, err := scanPost(r.db.Pool.QueryRow(ctx, query, args...))
	if err != nil {
		return nil, fmt.Errorf("update post: %w", err)
	}
	if updated == nil {
		return nil, fmt.Errorf("update post: %w", ErrPostNotFound)
	}
	return updated, nil
}
