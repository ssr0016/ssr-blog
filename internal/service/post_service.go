package service

import (
	"context"
	"errors"
	"time"

	"github.com/ssr0016/ssr-blog/internal/apperror"
	"github.com/ssr0016/ssr-blog/internal/model"
	"github.com/ssr0016/ssr-blog/internal/repository"
	"github.com/ssr0016/ssr-blog/pkg/metrics"
)

// maxSlugAttempts caps the slug retry loop: base, base-2, ... base-100.
const maxSlugAttempts = 100

type PostService struct {
	postRepo repository.PostRepository
	clock    func() time.Time
}

func NewPostService(postRepo repository.PostRepository) *PostService {
	return &PostService{
		postRepo: postRepo,
		clock:    time.Now,
	}
}

// Create makes a new post. The slug is derived from the title; if it is taken (by any post,
// including soft-deleted ones) a numeric suffix is appended and the insert is retried. The
// database unique index is the source of truth, so concurrent creates cannot produce duplicates.
func (s *PostService) Create(ctx context.Context, req model.CreatePostRequest) (*model.Post, error) {
	status := req.Status
	if status == "" {
		status = model.PostStatusDraft
	}
	if status != model.PostStatusDraft && status != model.PostStatusPublished {
		return nil, apperror.Validation("status must be draft or published")
	}

	base := Slugify(req.Title)
	if base == "" {
		return nil, apperror.Validation("title has no usable characters for a slug")
	}

	post := model.Post{
		Title:         req.Title,
		Content:       req.Content,
		Excerpt:       req.Excerpt,
		CoverImageURL: req.CoverImageURL,
		Status:        status,
	}
	if status == model.PostStatusPublished {
		publishedAt := s.clock()
		post.PublishedAt = &publishedAt
	}

	for n := 1; n <= maxSlugAttempts; n++ {
		post.Slug = WithSuffix(base, n)

		created, err := s.postRepo.Create(ctx, post)
		if errors.Is(err, repository.ErrSlugTaken) {
			continue
		}
		if err != nil {
			return nil, apperror.Internal("failed to create post").WithError(err)
		}

		metrics.PostSlugAttempts.Observe(float64(n))
		metrics.PostWritesTotal.WithLabelValues(string(metrics.PostOpCreate)).Inc()
		return created, nil
	}

	return nil, apperror.Internal("slug collision limit reached")
}

// GetByID returns a non-deleted post of any status, or a not-found error.
func (s *PostService) GetByID(ctx context.Context, id int64) (*model.Post, error) {
	post, err := s.postRepo.GetByID(ctx, id)
	if err != nil {
		return nil, apperror.Internal("failed to get post").WithError(err)
	}
	if post == nil {
		return nil, apperror.NotFound("post not found")
	}
	return post, nil
}
