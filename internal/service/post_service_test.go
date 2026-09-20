package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/ssr0016/ssr-blog/internal/apperror"
	"github.com/ssr0016/ssr-blog/internal/model"
	"github.com/ssr0016/ssr-blog/internal/repository"
	"github.com/ssr0016/ssr-blog/pkg/pagination"
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
	return writesTotal(t, "create")
}

// writesTotal reads blog_post_writes_total{operation=op} (0 if the series does not exist yet).
func writesTotal(t *testing.T, op string) float64 {
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
				if lp.GetName() == "operation" && lp.GetValue() == op {
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

// ============================================================
// ListPublished
// ============================================================

func TestPostService_ListPublished_ReturnsSummariesAndTotalFromRepo(t *testing.T) {
	want := []model.PostSummary{{Title: "A", Slug: "a"}, {Title: "B", Slug: "b"}}
	repo := repository.NewMockPostRepo()
	repo.ListPublishedFunc = func(_ context.Context, page, limit int) ([]model.PostSummary, int64, error) {
		if page != 2 || limit != 10 {
			t.Errorf("repo got page=%d limit=%d, want 2, 10", page, limit)
		}
		return want, 42, nil
	}
	svc := newTestPostService(repo)

	got, total, err := svc.ListPublished(context.Background(), pagination.Params{Page: 2, Limit: 10})
	if err != nil {
		t.Fatalf("ListPublished() error = %v", err)
	}
	if total != 42 || len(got) != 2 || got[0].Slug != "a" || got[1].Slug != "b" {
		t.Errorf("ListPublished() = %+v, %d", got, total)
	}
}

func TestPostService_ListPublished_EmptyIsNotAnError(t *testing.T) {
	svc := newTestPostService(repository.NewMockPostRepo())

	got, total, err := svc.ListPublished(context.Background(), pagination.Params{Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("ListPublished() error = %v", err)
	}
	if total != 0 || len(got) != 0 {
		t.Errorf("ListPublished() = %+v, %d, want empty", got, total)
	}
}

func TestPostService_ListPublished_RepoErrorBecomesInternalWithoutLeaking(t *testing.T) {
	cause := errors.New("db down secret-detail")
	repo := repository.NewMockPostRepo()
	repo.ListPublishedFunc = func(context.Context, int, int) ([]model.PostSummary, int64, error) { return nil, 0, cause }
	svc := newTestPostService(repo)

	_, _, err := svc.ListPublished(context.Background(), pagination.Params{Page: 1, Limit: 20})
	appErr := requireAppError(t, err, apperror.CodeInternal, http.StatusInternalServerError)
	if strings.Contains(appErr.Message, "secret-detail") {
		t.Errorf("client-facing message leaks the cause: %q", appErr.Message)
	}
	if !errors.Is(err, cause) {
		t.Error("cause must stay attached for server-side logging")
	}
}

// ============================================================
// GetPublishedBySlug
// ============================================================

func TestPostService_GetPublishedBySlug_Found(t *testing.T) {
	repo := repository.NewMockPostRepo()
	svc := newTestPostService(repo)
	if _, err := svc.Create(context.Background(), model.CreatePostRequest{Title: "Live Post", Content: "C", Status: model.PostStatusPublished}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := svc.GetPublishedBySlug(context.Background(), "live-post")
	if err != nil {
		t.Fatalf("GetPublishedBySlug() error = %v", err)
	}
	if got.Slug != "live-post" || got.Content != "C" {
		t.Errorf("GetPublishedBySlug() = %+v", got)
	}
}

// Draft, deleted and missing slugs all reach the service as "the repo returned nil", so they must
// produce the same error, message included.
func TestPostService_GetPublishedBySlug_NotFoundParity(t *testing.T) {
	repo := repository.NewMockPostRepo()
	svc := newTestPostService(repo)
	if _, err := svc.Create(context.Background(), model.CreatePostRequest{Title: "Secret Draft", Content: "C"}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	repo.GetPublishedBySlugFunc = nil

	var first *apperror.AppError
	for _, slug := range []string{"secret-draft", "missing", "deleted-one"} {
		got, err := svc.GetPublishedBySlug(context.Background(), slug)
		if got != nil {
			t.Errorf("%s: post = %+v, want nil", slug, got)
		}
		appErr := requireAppError(t, err, apperror.CodeNotFound, http.StatusNotFound)
		if first == nil {
			first = appErr
		} else if appErr.Message != first.Message {
			t.Errorf("%s: message %q differs from %q", slug, appErr.Message, first.Message)
		}
	}
}

func TestPostService_GetPublishedBySlug_NilFromRepoIsNotFound(t *testing.T) {
	repo := repository.NewMockPostRepo()
	repo.GetPublishedBySlugFunc = func(context.Context, string) (*model.Post, error) { return nil, nil }
	svc := newTestPostService(repo)

	got, err := svc.GetPublishedBySlug(context.Background(), "anything")
	if got != nil {
		t.Errorf("post = %+v, want nil", got)
	}
	_ = requireAppError(t, err, apperror.CodeNotFound, http.StatusNotFound)
}

func TestPostService_GetPublishedBySlug_RepoErrorBecomesInternal(t *testing.T) {
	cause := errors.New("db down secret-detail")
	repo := repository.NewMockPostRepo()
	repo.GetPublishedBySlugFunc = func(context.Context, string) (*model.Post, error) { return nil, cause }
	svc := newTestPostService(repo)

	_, err := svc.GetPublishedBySlug(context.Background(), "x")
	appErr := requireAppError(t, err, apperror.CodeInternal, http.StatusInternalServerError)
	if strings.Contains(appErr.Message, "secret-detail") {
		t.Errorf("client-facing message leaks the cause: %q", appErr.Message)
	}
	if !errors.Is(err, cause) {
		t.Error("cause must stay attached for server-side logging")
	}
}

// ============================================================
// ListAdmin
// ============================================================

func TestPostService_ListAdmin_PassesFilterAndPagingToRepo(t *testing.T) {
	want := []model.AdminPostSummary{{ID: 2, Slug: "b"}, {ID: 1, Slug: "a"}}
	repo := repository.NewMockPostRepo()
	repo.ListAdminFunc = func(_ context.Context, status string, page, limit int) ([]model.AdminPostSummary, int64, error) {
		if status != model.PostStatusDraft || page != 3 || limit != 10 {
			t.Errorf("repo got status=%q page=%d limit=%d, want draft, 3, 10", status, page, limit)
		}
		return want, 42, nil
	}
	svc := newTestPostService(repo)

	got, total, err := svc.ListAdmin(context.Background(), model.PostStatusDraft, pagination.Params{Page: 3, Limit: 10})
	if err != nil {
		t.Fatalf("ListAdmin() error = %v", err)
	}
	if total != 42 || len(got) != 2 || got[0].ID != 2 || got[1].ID != 1 {
		t.Errorf("ListAdmin() = %+v, %d", got, total)
	}
}

func TestPostService_ListAdmin_AcceptsEmptyDraftAndPublished(t *testing.T) {
	for _, status := range []string{"", model.PostStatusDraft, model.PostStatusPublished} {
		called := false
		repo := repository.NewMockPostRepo()
		repo.ListAdminFunc = func(_ context.Context, got string, _, _ int) ([]model.AdminPostSummary, int64, error) {
			called = true
			if got != status {
				t.Errorf("repo got status %q, want %q", got, status)
			}
			return nil, 0, nil
		}
		svc := newTestPostService(repo)

		if _, _, err := svc.ListAdmin(context.Background(), status, pagination.Params{Page: 1, Limit: 20}); err != nil {
			t.Errorf("status %q: error = %v", status, err)
		}
		if !called {
			t.Errorf("status %q: repo was not called", status)
		}
	}
}

func TestPostService_ListAdmin_RejectsUnknownStatusWithoutTouchingRepo(t *testing.T) {
	for _, status := range []string{"bogus", "Draft", "PUBLISHED", " draft", "draft ", "deleted", "all"} {
		repo := repository.NewMockPostRepo()
		repo.ListAdminFunc = func(context.Context, string, int, int) ([]model.AdminPostSummary, int64, error) {
			t.Errorf("status %q: repo must not be called for an invalid filter", status)
			return nil, 0, nil
		}
		svc := newTestPostService(repo)

		got, total, err := svc.ListAdmin(context.Background(), status, pagination.Params{Page: 1, Limit: 20})
		if got != nil || total != 0 {
			t.Errorf("status %q: got %+v, %d, want nothing", status, got, total)
		}
		_ = requireAppError(t, err, apperror.CodeValidation, http.StatusUnprocessableEntity)
	}
}

func TestPostService_ListAdmin_EmptyIsNotAnError(t *testing.T) {
	svc := newTestPostService(repository.NewMockPostRepo())

	got, total, err := svc.ListAdmin(context.Background(), "", pagination.Params{Page: 1, Limit: 20})
	if err != nil {
		t.Fatalf("ListAdmin() error = %v", err)
	}
	if total != 0 || len(got) != 0 {
		t.Errorf("ListAdmin() = %+v, %d, want empty", got, total)
	}
}

func TestPostService_ListAdmin_RepoErrorBecomesInternalWithoutLeaking(t *testing.T) {
	cause := errors.New("db down secret-detail")
	repo := repository.NewMockPostRepo()
	repo.ListAdminFunc = func(context.Context, string, int, int) ([]model.AdminPostSummary, int64, error) {
		return nil, 0, cause
	}
	svc := newTestPostService(repo)

	_, _, err := svc.ListAdmin(context.Background(), "", pagination.Params{Page: 1, Limit: 20})
	appErr := requireAppError(t, err, apperror.CodeInternal, http.StatusInternalServerError)
	if strings.Contains(appErr.Message, "secret-detail") {
		t.Errorf("client-facing message leaks the cause: %q", appErr.Message)
	}
	if !errors.Is(err, cause) {
		t.Error("cause must stay attached for server-side logging")
	}
}

// ============================================================
// Update
// ============================================================

var (
	t1 = fixedNow
	t2 = fixedNow.Add(time.Hour)
	t3 = fixedNow.Add(2 * time.Hour)
)

// seedForUpdate creates a post through the service at time at, with the given status.
func seedForUpdate(t *testing.T, repo repository.PostRepository, title, status string, at time.Time) *model.Post {
	t.Helper()
	svc := NewPostService(repo)
	svc.clock = func() time.Time { return at }
	created, err := svc.Create(context.Background(), model.CreatePostRequest{Title: title, Content: "old body", Status: status})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	return created
}

func updateReq(status string) model.UpdatePostRequest {
	return model.UpdatePostRequest{Title: "Edited Title", Content: "new body", Excerpt: "new excerpt", Status: status}
}

func TestPostService_Update_PublishedAtTransitions(t *testing.T) {
	tests := []struct {
		name        string
		from, to    string
		seedAt      time.Time // clock when the post was created
		updateAt    time.Time // clock when it is edited
		wantNil     bool
		wantAt      time.Time
		wantChange  PostTransition
		wantPublish float64 // expected increments of the publish counter
		wantUnpub   float64 // expected increments of the unpublish counter
	}{
		{"draft to published stamps the clock", model.PostStatusDraft, model.PostStatusPublished, t1, t2, false, t2, TransitionPublished, 1, 0},
		{"published to published keeps the original", model.PostStatusPublished, model.PostStatusPublished, t1, t2, false, t1, TransitionNone, 0, 0},
		{"published to draft clears it", model.PostStatusPublished, model.PostStatusDraft, t1, t2, true, time.Time{}, TransitionUnpublished, 0, 1},
		{"draft to draft stays empty", model.PostStatusDraft, model.PostStatusDraft, t1, t2, true, time.Time{}, TransitionNone, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := repository.NewMockPostRepo()
			seeded := seedForUpdate(t, repo, "Some Post", tt.from, tt.seedAt)
			svc := newTestPostService(repo)
			svc.clock = func() time.Time { return tt.updateAt }
			publishBefore, unpubBefore := writesTotal(t, "publish"), writesTotal(t, "unpublish")

			got, change, err := svc.Update(context.Background(), seeded.ID, updateReq(tt.to))
			if err != nil {
				t.Fatalf("Update() error = %v", err)
			}

			if tt.wantNil {
				if got.PublishedAt != nil {
					t.Errorf("published_at = %v, want nil", got.PublishedAt)
				}
			} else if got.PublishedAt == nil || !got.PublishedAt.Equal(tt.wantAt) {
				t.Errorf("published_at = %v, want %v", got.PublishedAt, tt.wantAt)
			}
			if change != tt.wantChange {
				t.Errorf("transition = %v, want %v", change, tt.wantChange)
			}
			if d := writesTotal(t, "publish") - publishBefore; d != tt.wantPublish {
				t.Errorf("publish counter moved by %v, want %v", d, tt.wantPublish)
			}
			if d := writesTotal(t, "unpublish") - unpubBefore; d != tt.wantUnpub {
				t.Errorf("unpublish counter moved by %v, want %v", d, tt.wantUnpub)
			}

			stored, _ := repo.GetByID(context.Background(), seeded.ID)
			if (stored.PublishedAt == nil) != tt.wantNil {
				t.Errorf("stored published_at = %v, wantNil = %v", stored.PublishedAt, tt.wantNil)
			}
		})
	}
}

func TestPostService_Update_RepublishGetsANewTimestampNotTheOldOne(t *testing.T) {
	repo := repository.NewMockPostRepo()
	seeded := seedForUpdate(t, repo, "Flip Flop", model.PostStatusPublished, t1)
	svc := newTestPostService(repo)

	svc.clock = func() time.Time { return t2 }
	if _, _, err := svc.Update(context.Background(), seeded.ID, updateReq(model.PostStatusDraft)); err != nil {
		t.Fatalf("unpublish: %v", err)
	}
	svc.clock = func() time.Time { return t3 }
	got, change, err := svc.Update(context.Background(), seeded.ID, updateReq(model.PostStatusPublished))
	if err != nil {
		t.Fatalf("republish: %v", err)
	}

	if got.PublishedAt == nil || !got.PublishedAt.Equal(t3) {
		t.Errorf("published_at = %v, want the new clock %v, not the original %v", got.PublishedAt, t3, t1)
	}
	if change != TransitionPublished {
		t.Errorf("transition = %v, want TransitionPublished", change)
	}
}

func TestPostService_Update_EditingAPublishedPostKeepsItsPublishedAtEvenWithAllFieldsChanged(t *testing.T) {
	repo := repository.NewMockPostRepo()
	seeded := seedForUpdate(t, repo, "Stable Date", model.PostStatusPublished, t1)
	svc := newTestPostService(repo)
	svc.clock = func() time.Time { return t3 }

	got, _, err := svc.Update(context.Background(), seeded.ID, model.UpdatePostRequest{
		Title: "Totally Different", Content: "other", Excerpt: "other", CoverImageURL: "https://example.com/x.png", Status: model.PostStatusPublished,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if got.PublishedAt == nil || !got.PublishedAt.Equal(t1) {
		t.Errorf("published_at = %v, want unchanged %v", got.PublishedAt, t1)
	}
}

func TestPostService_Update_SlugNeverChanges(t *testing.T) {
	repo := repository.NewMockPostRepo()
	seeded := seedForUpdate(t, repo, "Original Title", model.PostStatusDraft, t1)
	svc := newTestPostService(repo)

	got, _, err := svc.Update(context.Background(), seeded.ID, updateReq(model.PostStatusDraft))
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if got.Title != "Edited Title" {
		t.Fatalf("title = %q, the edit did not apply", got.Title)
	}
	if got.Slug != "original-title" {
		t.Errorf("slug = %q, want it frozen at original-title", got.Slug)
	}
}

func TestPostService_Update_AppliesEditableFieldsAndKeepsIdentity(t *testing.T) {
	repo := repository.NewMockPostRepo()
	seeded := seedForUpdate(t, repo, "Identity", model.PostStatusDraft, t1)
	svc := newTestPostService(repo)

	got, _, err := svc.Update(context.Background(), seeded.ID, model.UpdatePostRequest{
		Title: "New T", Content: "New C", Excerpt: "New E", CoverImageURL: "https://example.com/n.png", Status: model.PostStatusDraft,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if got.ID != seeded.ID || got.Title != "New T" || got.Content != "New C" || got.Excerpt != "New E" || got.CoverImageURL != "https://example.com/n.png" {
		t.Errorf("Update() = %+v", got)
	}
}

func TestPostService_Update_RecordsUpdateMetricOnSuccessOnly(t *testing.T) {
	repo := repository.NewMockPostRepo()
	seeded := seedForUpdate(t, repo, "Counted", model.PostStatusDraft, t1)
	svc := newTestPostService(repo)
	before := writesTotal(t, "update")

	if _, _, err := svc.Update(context.Background(), seeded.ID, updateReq(model.PostStatusDraft)); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if d := writesTotal(t, "update") - before; d != 1 {
		t.Errorf("update counter moved by %v, want 1", d)
	}

	// Failures must not count.
	_, _, _ = svc.Update(context.Background(), 99999, updateReq(model.PostStatusDraft))
	_, _, _ = svc.Update(context.Background(), seeded.ID, updateReq("bogus"))
	if d := writesTotal(t, "update") - before; d != 1 {
		t.Errorf("update counter moved by %v after failures, want still 1", d)
	}
}

func TestPostService_Update_MissingAndDeletedAreNotFound(t *testing.T) {
	repo := repository.NewMockPostRepo()
	gone := seedForUpdate(t, repo, "Gone", model.PostStatusDraft, t1)
	repo.MarkDeleted(gone.ID)
	svc := newTestPostService(repo)

	for name, id := range map[string]int64{"missing": 424242, "deleted": gone.ID} {
		got, change, err := svc.Update(context.Background(), id, updateReq(model.PostStatusDraft))
		if got != nil || change != TransitionNone {
			t.Errorf("%s: got %+v, %v, want nothing", name, got, change)
		}
		_ = requireAppError(t, err, apperror.CodeNotFound, http.StatusNotFound)
	}
}

func TestPostService_Update_RejectsUnknownStatusWithoutWriting(t *testing.T) {
	for _, status := range []string{"", "archived", "Published", " draft"} {
		repo := repository.NewMockPostRepo()
		seeded := seedForUpdate(t, repo, "Guarded", model.PostStatusDraft, t1)
		repo.UpdateFunc = func(context.Context, model.Post) (*model.Post, error) {
			t.Errorf("status %q: repo.Update must not be called", status)
			return nil, nil
		}
		svc := newTestPostService(repo)

		_, _, err := svc.Update(context.Background(), seeded.ID, updateReq(status))
		_ = requireAppError(t, err, apperror.CodeValidation, http.StatusUnprocessableEntity)
	}
}

// The post can be deleted by another request between the service's read and the repository's write.
func TestPostService_Update_RepoNotFoundAfterReadIsStillNotFound(t *testing.T) {
	repo := repository.NewMockPostRepo()
	seeded := seedForUpdate(t, repo, "Racy", model.PostStatusDraft, t1)
	repo.UpdateFunc = func(context.Context, model.Post) (*model.Post, error) {
		return nil, fmt.Errorf("update post: %w", repository.ErrPostNotFound)
	}
	svc := newTestPostService(repo)
	before := writesTotal(t, "update")

	_, _, err := svc.Update(context.Background(), seeded.ID, updateReq(model.PostStatusDraft))

	_ = requireAppError(t, err, apperror.CodeNotFound, http.StatusNotFound)
	if writesTotal(t, "update") != before {
		t.Error("a failed update must not be counted")
	}
}

func TestPostService_Update_RepoErrorsBecomeInternalWithoutLeaking(t *testing.T) {
	cause := errors.New("db down secret-detail")

	t.Run("read fails", func(t *testing.T) {
		repo := repository.NewMockPostRepo()
		repo.GetByIDFunc = func(context.Context, int64) (*model.Post, error) { return nil, cause }
		svc := newTestPostService(repo)

		_, _, err := svc.Update(context.Background(), 1, updateReq(model.PostStatusDraft))

		appErr := requireAppError(t, err, apperror.CodeInternal, http.StatusInternalServerError)
		if strings.Contains(appErr.Message, "secret-detail") {
			t.Errorf("client-facing message leaks the cause: %q", appErr.Message)
		}
		if !errors.Is(err, cause) {
			t.Error("cause must stay attached for server-side logging")
		}
	})

	t.Run("write fails", func(t *testing.T) {
		repo := repository.NewMockPostRepo()
		seeded := seedForUpdate(t, repo, "Write Fail", model.PostStatusDraft, t1)
		repo.UpdateFunc = func(context.Context, model.Post) (*model.Post, error) { return nil, cause }
		svc := newTestPostService(repo)
		before := writesTotal(t, "update")

		_, _, err := svc.Update(context.Background(), seeded.ID, updateReq(model.PostStatusDraft))

		appErr := requireAppError(t, err, apperror.CodeInternal, http.StatusInternalServerError)
		if strings.Contains(appErr.Message, "secret-detail") {
			t.Errorf("client-facing message leaks the cause: %q", appErr.Message)
		}
		if !errors.Is(err, cause) {
			t.Error("cause must stay attached for server-side logging")
		}
		if writesTotal(t, "update") != before {
			t.Error("a failed update must not be counted")
		}
	})
}

// The real repository ignores the slug on update, but the service must still hand it the stored
// one so the two layers agree and a repo that did write it could never rename a post.
func TestPostService_Update_PassesTheStoredSlugToTheRepo(t *testing.T) {
	repo := repository.NewMockPostRepo()
	seeded := seedForUpdate(t, repo, "Original Title", model.PostStatusDraft, t1)
	var sent model.Post
	repo.UpdateFunc = func(_ context.Context, p model.Post) (*model.Post, error) {
		sent = p
		return &p, nil
	}
	svc := newTestPostService(repo)

	if _, _, err := svc.Update(context.Background(), seeded.ID, updateReq(model.PostStatusDraft)); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if sent.Slug != "original-title" || sent.ID != seeded.ID {
		t.Errorf("repo received id=%d slug=%q, want id=%d slug=original-title", sent.ID, sent.Slug, seeded.ID)
	}
}
