package service

import (
	"context"
	"errors"
	"time"

	"github.com/ssr0016/ssr-blog/internal/apperror"
	"github.com/ssr0016/ssr-blog/internal/model"
	"github.com/ssr0016/ssr-blog/internal/repository"
	"github.com/ssr0016/ssr-blog/pkg/metrics"
	"github.com/ssr0016/ssr-blog/pkg/pagination"
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

// ListPublished returns one page of public post summaries, newest first, and the total number of
// public posts. An empty result is not an error.
func (s *PostService) ListPublished(ctx context.Context, params pagination.Params) ([]model.PostSummary, int64, error) {
	items, total, err := s.postRepo.ListPublished(ctx, params.Page, params.Limit)
	if err != nil {
		return nil, 0, apperror.Internal("failed to list posts").WithError(err)
	}
	return items, total, nil
}

// GetPublishedBySlug returns a public post. A draft, a soft-deleted post and a slug that does not
// exist all return the same not-found error, so callers cannot tell them apart.
func (s *PostService) GetPublishedBySlug(ctx context.Context, slug string) (*model.Post, error) {
	if !validSlug(slug) {
		return nil, apperror.NotFound("post not found")
	}
	post, err := s.postRepo.GetPublishedBySlug(ctx, slug)
	if err != nil {
		return nil, apperror.Internal("failed to get post").WithError(err)
	}
	if post == nil {
		return nil, apperror.NotFound("post not found")
	}
	return post, nil
}

// ListAdmin returns one page of posts of any status for the admin, newest first, and the total for
// the same filter. status must be "" (all), "draft" or "published"; anything else is a validation
// error and the repository is not queried.
func (s *PostService) ListAdmin(ctx context.Context, status string, params pagination.Params) ([]model.AdminPostSummary, int64, error) {
	switch status {
	case "", model.PostStatusDraft, model.PostStatusPublished:
	default:
		return nil, 0, apperror.Validation("status must be draft or published")
	}

	items, total, err := s.postRepo.ListAdmin(ctx, status, params.Page, params.Limit)
	if err != nil {
		return nil, 0, apperror.Internal("failed to list posts").WithError(err)
	}
	return items, total, nil
}

// PostTransition says how an update changed a post's publication state. Callers use it to pick the
// audit action.
type PostTransition int

const (
	TransitionNone        PostTransition = iota // status did not change
	TransitionPublished                         // draft -> published
	TransitionUnpublished                       // published -> draft
)

// Update replaces the editable fields of a non-deleted post. The slug is never touched.
//
// published_at follows the status change: draft -> published stamps the clock (a re-publish gets a
// fresh time, not the old one), published -> published keeps the existing value however many other
// fields change, and any move to draft clears it. The current post is read first to know which case
// applies; if it is deleted in between, the repository reports not-found and so do we.
func (s *PostService) Update(ctx context.Context, id int64, req model.UpdatePostRequest) (*model.Post, PostTransition, error) {
	if req.Status != model.PostStatusDraft && req.Status != model.PostStatusPublished {
		return nil, TransitionNone, apperror.Validation("status must be draft or published")
	}

	current, err := s.postRepo.GetByID(ctx, id)
	if err != nil {
		return nil, TransitionNone, apperror.Internal("failed to get post").WithError(err)
	}
	if current == nil {
		return nil, TransitionNone, apperror.NotFound("post not found")
	}

	next := model.Post{
		ID:            current.ID,
		Title:         req.Title,
		Slug:          current.Slug,
		Content:       req.Content,
		Excerpt:       req.Excerpt,
		CoverImageURL: req.CoverImageURL,
		Status:        req.Status,
	}
	transition := TransitionNone
	switch {
	case req.Status == model.PostStatusPublished && current.Status != model.PostStatusPublished:
		publishedAt := s.clock()
		next.PublishedAt = &publishedAt
		transition = TransitionPublished
	case req.Status == model.PostStatusPublished:
		next.PublishedAt = current.PublishedAt
	case current.Status == model.PostStatusPublished:
		transition = TransitionUnpublished
	}

	updated, err := s.postRepo.Update(ctx, next)
	if errors.Is(err, repository.ErrPostNotFound) {
		return nil, TransitionNone, apperror.NotFound("post not found")
	}
	if err != nil {
		return nil, TransitionNone, apperror.Internal("failed to update post").WithError(err)
	}

	metrics.PostWritesTotal.WithLabelValues(string(metrics.PostOpUpdate)).Inc()
	switch transition {
	case TransitionPublished:
		metrics.PostWritesTotal.WithLabelValues(string(metrics.PostOpPublish)).Inc()
	case TransitionUnpublished:
		metrics.PostWritesTotal.WithLabelValues(string(metrics.PostOpUnpublish)).Inc()
	case TransitionNone:
	}
	return updated, transition, nil
}

// Delete soft-deletes a non-deleted post and returns it, so the caller can audit the slug. The post
// is read first to tell "not found" from a failure; if it is deleted in between, the repository
// reports not-found and so do we. The row is kept and its slug stays reserved. Only a successful
// delete is counted.
func (s *PostService) Delete(ctx context.Context, id int64) (*model.Post, error) {
	post, err := s.postRepo.GetByID(ctx, id)
	if err != nil {
		return nil, apperror.Internal("failed to get post").WithError(err)
	}
	if post == nil {
		return nil, apperror.NotFound("post not found")
	}

	err = s.postRepo.SoftDelete(ctx, id)
	if errors.Is(err, repository.ErrPostNotFound) {
		return nil, apperror.NotFound("post not found")
	}
	if err != nil {
		return nil, apperror.Internal("failed to delete post").WithError(err)
	}

	metrics.PostWritesTotal.WithLabelValues(string(metrics.PostOpDelete)).Inc()
	return post, nil
}
