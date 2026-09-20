package model

import "time"

// Post statuses.
const (
	PostStatusDraft     = "draft"
	PostStatusPublished = "published"
)

// Post is a blog post. Slug is generated from the title on creation and never changes.
// DeletedAt is set on soft delete; deleted posts are hidden from every read.
type Post struct {
	ID            int64      `db:"id"`
	Title         string     `db:"title"`
	Slug          string     `db:"slug"`
	Content       string     `db:"content"` // markdown source, stored as written
	Excerpt       string     `db:"excerpt"`
	CoverImageURL string     `db:"cover_image_url"`
	Status        string     `db:"status"`
	PublishedAt   *time.Time `db:"published_at"`
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"`
	DeletedAt     *time.Time `db:"deleted_at"`
}

// PostSummary is the public list item. It carries no content and no status.
type PostSummary struct {
	Title         string     `json:"title" db:"title"`
	Slug          string     `json:"slug" db:"slug"`
	Excerpt       string     `json:"excerpt" db:"excerpt"`
	CoverImageURL string     `json:"cover_image_url" db:"cover_image_url"`
	PublishedAt   *time.Time `json:"published_at" db:"published_at"`
}

// AdminPostSummary is the admin list item. It carries no content.
type AdminPostSummary struct {
	ID            int64      `json:"id" db:"id"`
	Title         string     `json:"title" db:"title"`
	Slug          string     `json:"slug" db:"slug"`
	Excerpt       string     `json:"excerpt" db:"excerpt"`
	CoverImageURL string     `json:"cover_image_url" db:"cover_image_url"`
	Status        string     `json:"status" db:"status"`
	PublishedAt   *time.Time `json:"published_at" db:"published_at"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// ============================================================
// Request DTOs
// ============================================================

// CreatePostRequest is the payload for creating a post.
// There is no slug field: the slug is always derived from the title.
// Status is optional and defaults to draft.
type CreatePostRequest struct {
	Title         string `json:"title" validate:"required,notblank,max=200"`
	Content       string `json:"content" validate:"required,notblank,max=100000"`
	Excerpt       string `json:"excerpt" validate:"max=500"`
	CoverImageURL string `json:"cover_image_url" validate:"omitempty,max=2048,http_url"`
	Status        string `json:"status" validate:"omitempty,oneof=draft published"`
}

// UpdatePostRequest is the payload for editing a post (full replace of the editable fields).
// There is no slug field: a slug never changes after creation.
type UpdatePostRequest struct {
	Title         string `json:"title" validate:"required,notblank,max=200"`
	Content       string `json:"content" validate:"required,notblank,max=100000"`
	Excerpt       string `json:"excerpt" validate:"max=500"`
	CoverImageURL string `json:"cover_image_url" validate:"omitempty,max=2048,http_url"`
	Status        string `json:"status" validate:"required,oneof=draft published"`
}

// ============================================================
// Responses
// ============================================================

// PostResponse is the API representation of a single post. deleted_at is never exposed.
type PostResponse struct {
	ID            int64      `json:"id"`
	Title         string     `json:"title"`
	Slug          string     `json:"slug"`
	Content       string     `json:"content"`
	Excerpt       string     `json:"excerpt"`
	CoverImageURL string     `json:"cover_image_url"`
	Status        string     `json:"status"`
	PublishedAt   *time.Time `json:"published_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ToResponse converts Post to PostResponse.
func (p *Post) ToResponse() PostResponse {
	return PostResponse{
		ID:            p.ID,
		Title:         p.Title,
		Slug:          p.Slug,
		Content:       p.Content,
		Excerpt:       p.Excerpt,
		CoverImageURL: p.CoverImageURL,
		Status:        p.Status,
		PublishedAt:   p.PublishedAt,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}
