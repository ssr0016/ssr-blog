package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ssr0016/ssr-blog/internal/apperror"
	"github.com/ssr0016/ssr-blog/internal/model"
	"github.com/ssr0016/ssr-blog/internal/repository"
)

var fixedNow = time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC)

func newTestPostService(repo repository.PostRepository) *PostService {
	svc := NewPostService(repo)
	svc.clock = func() time.Time { return fixedNow }
	return svc
}

// requireAppError asserts err is an *apperror.AppError with the given code and HTTP status.
func requireAppError(t *testing.T, err error, code apperror.Code, status int) *apperror.AppError {
	t.Helper()
	appErr, ok := apperror.As(err)
	if !ok {
		t.Fatalf("error = %v (%T), want *apperror.AppError", err, err)
	}
	if appErr.Code != code || appErr.HTTPStatus != status {
		t.Fatalf("error = %s/%d (%q), want %s/%d", appErr.Code, appErr.HTTPStatus, appErr.Message, code, status)
	}
	return appErr
}

// createWritesTotal reads blog_post_writes_total{operation="create"} from the default registry
// (0 if the series does not exist yet).
func createWritesTotal(t *testing.T) float64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, mf := range families {
		if mf.GetName() != "blog_post_writes_total" {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, lp := range m.GetLabel() {
				if lp.GetName() == "operation" && lp.GetValue() == "create" {
					return m.GetCounter().GetValue()
				}
			}
		}
	}
	return 0
}

// slugAttemptTotals returns the sample count and sum of blog_post_slug_attempts (0, 0 if it has no samples yet).
func slugAttemptTotals(t *testing.T) (count uint64, sum float64) {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	for _, mf := range families {
		if mf.GetName() != "blog_post_slug_attempts" {
			continue
		}
		for _, m := range mf.GetMetric() {
			return m.GetHistogram().GetSampleCount(), m.GetHistogram().GetSampleSum()
		}
	}
	return 0, 0
}

func TestNewPostService_UsesRealClock(t *testing.T) {
	svc := NewPostService(repository.NewMockPostRepo())
	if svc.clock == nil {
		t.Fatal("clock must default to time.Now")
	}
	if d := time.Since(svc.clock()); d < 0 || d > time.Minute {
		t.Errorf("clock() is %v away from now", d)
	}
}

// ============================================================
// Create: happy paths
// ============================================================

func TestPostService_Create_DefaultsToDraft(t *testing.T) {
	svc := newTestPostService(repository.NewMockPostRepo())

	post, err := svc.Create(context.Background(), model.CreatePostRequest{
		Title:   "Hello World",
		Content: "Body",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if post.Status != model.PostStatusDraft {
		t.Errorf("Status = %q, want draft", post.Status)
	}
	if post.PublishedAt != nil {
		t.Errorf("PublishedAt = %v, want nil for a draft", post.PublishedAt)
	}
	if post.Slug != "hello-world" {
		t.Errorf("Slug = %q, want hello-world", post.Slug)
	}
	if post.ID == 0 {
		t.Error("ID must be set by the repository")
	}
}

func TestPostService_Create_ExplicitDraftHasNoPublishedAt(t *testing.T) {
	svc := newTestPostService(repository.NewMockPostRepo())

	post, err := svc.Create(context.Background(), model.CreatePostRequest{
		Title: "T", Content: "C", Status: model.PostStatusDraft,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if post.PublishedAt != nil {
		t.Errorf("PublishedAt = %v, want nil", post.PublishedAt)
	}
}

func TestPostService_Create_PublishedSetsPublishedAtFromClock(t *testing.T) {
	svc := newTestPostService(repository.NewMockPostRepo())

	post, err := svc.Create(context.Background(), model.CreatePostRequest{
		Title: "T", Content: "C", Status: model.PostStatusPublished,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if post.Status != model.PostStatusPublished {
		t.Errorf("Status = %q, want published", post.Status)
	}
	if post.PublishedAt == nil || !post.PublishedAt.Equal(fixedNow) {
		t.Errorf("PublishedAt = %v, want %v", post.PublishedAt, fixedNow)
	}
}

func TestPostService_Create_PassesFieldsToRepoAsWritten(t *testing.T) {
	var got model.Post
	repo := repository.NewMockPostRepo()
	repo.CreateFunc = func(_ context.Context, p model.Post) (*model.Post, error) {
		got = p
		return &p, nil
	}
	svc := newTestPostService(repo)

	content := "<script>alert(1)</script>\n\n  **md**  "
	_, err := svc.Create(context.Background(), model.CreatePostRequest{
		Title:         "Café Au Lait",
		Content:       content,
		Excerpt:       "short",
		CoverImageURL: "https://example.com/a.png",
		Status:        model.PostStatusDraft,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if got.Title != "Café Au Lait" || got.Slug != "cafe-au-lait" {
		t.Errorf("title/slug = %q/%q", got.Title, got.Slug)
	}
	if got.Content != content {
		t.Errorf("Content = %q, want it stored exactly as written", got.Content)
	}
	if got.Excerpt != "short" || got.CoverImageURL != "https://example.com/a.png" {
		t.Errorf("excerpt/cover = %q/%q", got.Excerpt, got.CoverImageURL)
	}
}

// ============================================================
// Create: slug collisions
// ============================================================

func TestPostService_Create_SecondPostWithSameTitleGetsSuffix(t *testing.T) {
	svc := newTestPostService(repository.NewMockPostRepo())
	ctx := context.Background()
	req := model.CreatePostRequest{Title: "Same Title", Content: "C"}

	first, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	second, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("second Create() error = %v", err)
	}
	if first.Slug != "same-title" || second.Slug != "same-title-2" {
		t.Errorf("slugs = %q, %q, want same-title, same-title-2", first.Slug, second.Slug)
	}
}

func TestPostService_Create_CollisionAtBaseAndSecondGetsThird(t *testing.T) {
	repo := repository.NewMockPostRepo()
	ctx := context.Background()
	for _, slug := range []string{"hello-world", "hello-world-2"} {
		if _, err := repo.Create(ctx, model.Post{Title: "x", Slug: slug, Content: "c", Status: model.PostStatusDraft}); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	svc := newTestPostService(repo)

	post, err := svc.Create(ctx, model.CreatePostRequest{Title: "Hello World", Content: "C"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if post.Slug != "hello-world-3" {
		t.Errorf("Slug = %q, want hello-world-3", post.Slug)
	}
}

func TestPostService_Create_TriesCandidatesInOrder(t *testing.T) {
	var tried []string
	repo := repository.NewMockPostRepo()
	repo.CreateFunc = func(_ context.Context, p model.Post) (*model.Post, error) {
		tried = append(tried, p.Slug)
		if len(tried) < 4 {
			return nil, repository.ErrSlugTaken
		}
		return &p, nil
	}
	svc := newTestPostService(repo)

	post, err := svc.Create(context.Background(), model.CreatePostRequest{Title: "Order", Content: "C"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	want := []string{"order", "order-2", "order-3", "order-4"}
	if strings.Join(tried, ",") != strings.Join(want, ",") {
		t.Errorf("tried = %v, want %v", tried, want)
	}
	if post.Slug != "order-4" {
		t.Errorf("Slug = %q, want order-4", post.Slug)
	}
}

func TestPostService_Create_WrappedSlugTakenStillRetries(t *testing.T) {
	// The real repository wraps ErrSlugTaken with context; the service must use errors.Is.
	calls := 0
	repo := repository.NewMockPostRepo()
	repo.CreateFunc = func(_ context.Context, p model.Post) (*model.Post, error) {
		calls++
		if calls == 1 {
			return nil, errors.Join(errors.New("create post"), repository.ErrSlugTaken)
		}
		return &p, nil
	}
	svc := newTestPostService(repo)

	if _, err := svc.Create(context.Background(), model.CreatePostRequest{Title: "T", Content: "C"}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if calls != 2 {
		t.Errorf("repo called %d times, want 2", calls)
	}
}

func TestPostService_Create_GivesUpAfter100Attempts(t *testing.T) {
	calls := 0
	repo := repository.NewMockPostRepo()
	repo.CreateFunc = func(context.Context, model.Post) (*model.Post, error) {
		calls++
		return nil, repository.ErrSlugTaken
	}
	svc := newTestPostService(repo)

	post, err := svc.Create(context.Background(), model.CreatePostRequest{Title: "Busy", Content: "C"})
	if post != nil {
		t.Errorf("post = %+v, want nil", post)
	}
	appErr := requireAppError(t, err, apperror.CodeInternal, http.StatusInternalServerError)
	if appErr.Message != "slug collision limit reached" {
		t.Errorf("Message = %q", appErr.Message)
	}
	if calls != 100 {
		t.Errorf("repo called %d times, want exactly 100", calls)
	}
}

func TestPostService_Create_Succeeds_OnThe100thAttempt(t *testing.T) {
	calls := 0
	var lastSlug string
	repo := repository.NewMockPostRepo()
	repo.CreateFunc = func(_ context.Context, p model.Post) (*model.Post, error) {
		calls++
		lastSlug = p.Slug
		if calls < 100 {
			return nil, repository.ErrSlugTaken
		}
		return &p, nil
	}
	svc := newTestPostService(repo)

	if _, err := svc.Create(context.Background(), model.CreatePostRequest{Title: "Edge", Content: "C"}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if lastSlug != "edge-100" {
		t.Errorf("last slug = %q, want edge-100", lastSlug)
	}
}

func TestPostService_Create_LongTitleSlugStaysWithinColumnOnCollision(t *testing.T) {
	svc := newTestPostService(repository.NewMockPostRepo())
	ctx := context.Background()
	req := model.CreatePostRequest{Title: strings.Repeat("a", 200), Content: "C"}

	first, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	second, err := svc.Create(ctx, req)
	if err != nil {
		t.Fatalf("second Create() error = %v", err)
	}
	if first.Slug == second.Slug {
		t.Errorf("slugs must differ, both %q", first.Slug)
	}
	if len(second.Slug) > MaxSlugLen {
		t.Errorf("suffixed slug has len %d, want <= %d", len(second.Slug), MaxSlugLen)
	}
}

// ============================================================
// Create: rejections (repo must never be called)
// ============================================================

func TestPostService_Create_RejectsTitlesWithNoUsableSlug(t *testing.T) {
	titles := map[string]string{
		"punctuation only": "!!!",
		"japanese":         "日本語のタイトル",
		"arabic":           "مرحبا بالعالم",
		"emoji":            "🚀",
	}
	for name, title := range titles {
		t.Run(name, func(t *testing.T) {
			called := false
			repo := repository.NewMockPostRepo()
			repo.CreateFunc = func(context.Context, model.Post) (*model.Post, error) {
				called = true
				return nil, nil
			}
			svc := newTestPostService(repo)

			post, err := svc.Create(context.Background(), model.CreatePostRequest{Title: title, Content: "C"})
			if post != nil {
				t.Errorf("post = %+v, want nil", post)
			}
			appErr := requireAppError(t, err, apperror.CodeValidation, http.StatusUnprocessableEntity)
			if appErr.Message != "title has no usable characters for a slug" {
				t.Errorf("Message = %q", appErr.Message)
			}
			if called {
				t.Error("repository must not be called when the slug is empty")
			}
		})
	}
}

func TestPostService_Create_RejectsUnknownStatus(t *testing.T) {
	for _, status := range []string{"archived", "Published", " draft", "DRAFT"} {
		t.Run(status, func(t *testing.T) {
			called := false
			repo := repository.NewMockPostRepo()
			repo.CreateFunc = func(context.Context, model.Post) (*model.Post, error) {
				called = true
				return nil, nil
			}
			svc := newTestPostService(repo)

			_, err := svc.Create(context.Background(), model.CreatePostRequest{Title: "T", Content: "C", Status: status})
			_ = requireAppError(t, err, apperror.CodeValidation, http.StatusUnprocessableEntity)
			if called {
				t.Error("repository must not be called for an invalid status")
			}
		})
	}
}

// ============================================================
// Create: repository failures
// ============================================================

func TestPostService_Create_RepoErrorBecomesInternalWithoutLeaking(t *testing.T) {
	cause := errors.New("connection refused: 10.0.0.5:5437 secret-detail")
	repo := repository.NewMockPostRepo()
	repo.CreateFunc = func(context.Context, model.Post) (*model.Post, error) { return nil, cause }
	svc := newTestPostService(repo)

	post, err := svc.Create(context.Background(), model.CreatePostRequest{Title: "T", Content: "C"})
	if post != nil {
		t.Errorf("post = %+v, want nil", post)
	}
	appErr := requireAppError(t, err, apperror.CodeInternal, http.StatusInternalServerError)
	if strings.Contains(appErr.Message, "secret-detail") || strings.Contains(appErr.Message, "5437") {
		t.Errorf("client-facing message leaks the cause: %q", appErr.Message)
	}
	if !errors.Is(err, cause) {
		t.Error("cause must stay attached for server-side logging")
	}
}

func TestPostService_Create_NonCollisionErrorStopsRetrying(t *testing.T) {
	calls := 0
	repo := repository.NewMockPostRepo()
	repo.CreateFunc = func(context.Context, model.Post) (*model.Post, error) {
		calls++
		if calls == 1 {
			return nil, repository.ErrSlugTaken
		}
		return nil, errors.New("disk full")
	}
	svc := newTestPostService(repo)

	_, err := svc.Create(context.Background(), model.CreatePostRequest{Title: "T", Content: "C"})
	_ = requireAppError(t, err, apperror.CodeInternal, http.StatusInternalServerError)
	if calls != 2 {
		t.Errorf("repo called %d times, want 2 (no retry after a non-collision error)", calls)
	}
}

// ============================================================
// Create: metrics
// ============================================================

func TestPostService_Create_RecordsMetricsOnSuccess(t *testing.T) {
	repo := repository.NewMockPostRepo()
	ctx := context.Background()
	for _, slug := range []string{"metrics", "metrics-2"} {
		if _, err := repo.Create(ctx, model.Post{Title: "x", Slug: slug, Content: "c", Status: model.PostStatusDraft}); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	svc := newTestPostService(repo)

	writesBefore := createWritesTotal(t)
	countBefore, sumBefore := slugAttemptTotals(t)

	if _, err := svc.Create(ctx, model.CreatePostRequest{Title: "Metrics", Content: "C"}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if got := createWritesTotal(t) - writesBefore; got != 1 {
		t.Errorf("blog_post_writes_total{create} delta = %v, want 1", got)
	}
	countAfter, sumAfter := slugAttemptTotals(t)
	if countAfter-countBefore != 1 {
		t.Errorf("slug attempts sample delta = %d, want 1", countAfter-countBefore)
	}
	if sumAfter-sumBefore != 3 {
		t.Errorf("slug attempts observed %v, want 3 (base, -2, then -3)", sumAfter-sumBefore)
	}
}

func TestPostService_Create_NoWriteMetricOnFailure(t *testing.T) {
	repo := repository.NewMockPostRepo()
	repo.CreateFunc = func(context.Context, model.Post) (*model.Post, error) { return nil, errors.New("boom") }
	svc := newTestPostService(repo)

	writesBefore := createWritesTotal(t)
	countBefore, _ := slugAttemptTotals(t)

	_, _ = svc.Create(context.Background(), model.CreatePostRequest{Title: "T", Content: "C"})
	_, _ = svc.Create(context.Background(), model.CreatePostRequest{Title: "!!!", Content: "C"})

	if got := createWritesTotal(t) - writesBefore; got != 0 {
		t.Errorf("write counter moved by %v on failures, want 0", got)
	}
	if countAfter, _ := slugAttemptTotals(t); countAfter != countBefore {
		t.Errorf("slug attempts histogram moved on failures")
	}
}

// ============================================================
// GetByID
// ============================================================

func TestPostService_GetByID_Found(t *testing.T) {
	repo := repository.NewMockPostRepo()
	svc := newTestPostService(repo)
	created, err := svc.Create(context.Background(), model.CreatePostRequest{Title: "Find Me", Content: "C"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := svc.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.ID != created.ID || got.Slug != "find-me" {
		t.Errorf("GetByID() = %+v", got)
	}
}

func TestPostService_GetByID_NotFound(t *testing.T) {
	svc := newTestPostService(repository.NewMockPostRepo())

	got, err := svc.GetByID(context.Background(), 12345)
	if got != nil {
		t.Errorf("post = %+v, want nil", got)
	}
	_ = requireAppError(t, err, apperror.CodeNotFound, http.StatusNotFound)
}

func TestPostService_GetByID_RepoErrorBecomesInternal(t *testing.T) {
	cause := errors.New("db down secret-detail")
	repo := repository.NewMockPostRepo()
	repo.GetByIDFunc = func(context.Context, int64) (*model.Post, error) { return nil, cause }
	svc := newTestPostService(repo)

	_, err := svc.GetByID(context.Background(), 1)
	appErr := requireAppError(t, err, apperror.CodeInternal, http.StatusInternalServerError)
	if strings.Contains(appErr.Message, "secret-detail") {
		t.Errorf("client-facing message leaks the cause: %q", appErr.Message)
	}
	if !errors.Is(err, cause) {
		t.Error("cause must stay attached for server-side logging")
	}
}
