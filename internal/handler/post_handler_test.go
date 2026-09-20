package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/ssr0016/ssr-blog/internal/apperror"
	"github.com/ssr0016/ssr-blog/internal/model"
	"github.com/ssr0016/ssr-blog/internal/repository"
	"github.com/ssr0016/ssr-blog/internal/service"
	"github.com/ssr0016/ssr-blog/internal/validator"
)

// ============================================================
// Test environment
// ============================================================

type publicPostEnv struct {
	handler http.Handler
	repo    *repository.MockPostRepo
	base    time.Time
	next    int
}

// newPublicPostEnv wires the real service around a mock repository. There is deliberately no
// session manager and no auth middleware: the public handler must not depend on either.
func newPublicPostEnv(t *testing.T) *publicPostEnv {
	t.Helper()

	repo := repository.NewMockPostRepo()
	h := NewPostHandler(service.NewPostService(repo))

	e := echo.New()
	e.Validator = validator.New()
	e.HTTPErrorHandler = apperror.ErrorHandler
	e.GET("/posts", h.List)
	e.GET("/posts/:slug", h.Get)

	return &publicPostEnv{
		handler: e,
		repo:    repo,
		base:    time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC),
	}
}

// seed stores a post. Published posts get a strictly increasing published_at.
func (env *publicPostEnv) seed(t *testing.T, title, slug, status string) *model.Post {
	t.Helper()
	p := model.Post{
		Title:         title,
		Slug:          slug,
		Content:       "SECRET-BODY of " + slug,
		Excerpt:       "excerpt of " + slug,
		CoverImageURL: "https://example.com/" + slug + ".png",
		Status:        status,
	}
	if status == model.PostStatusPublished {
		env.next++
		at := env.base.Add(time.Duration(env.next) * time.Minute)
		p.PublishedAt = &at
	}
	created, err := env.repo.Create(context.Background(), p)
	if err != nil {
		t.Fatalf("seed %s: %v", slug, err)
	}
	return created
}

func (env *publicPostEnv) get(t *testing.T, path string, mutate ...func(*http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
	for _, m := range mutate {
		m(req)
	}
	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, req)
	return rec
}

func listItems(t *testing.T, rec *httptest.ResponseRecorder) []map[string]any {
	t.Helper()
	raw, ok := decodeMap(t, rec)["data"].([]any)
	if !ok {
		t.Fatalf("data is not an array\nbody: %s", rec.Body.String())
	}
	out := make([]map[string]any, 0, len(raw))
	for _, r := range raw {
		out = append(out, r.(map[string]any))
	}
	return out
}

// ============================================================
// GET /posts
// ============================================================

func TestPostHandler_List_ReturnsSummariesNewestFirstWithoutContent(t *testing.T) {
	env := newPublicPostEnv(t)
	env.seed(t, "First", "first", model.PostStatusPublished)
	env.seed(t, "Hidden Draft", "hidden-draft", model.PostStatusDraft)
	env.seed(t, "Second", "second", model.PostStatusPublished)

	rec := env.get(t, "/posts")

	requireStatus(t, rec, http.StatusOK, "")
	if ct := rec.Header().Get(echo.HeaderContentType); !strings.HasPrefix(ct, echo.MIMEApplicationJSON) {
		t.Errorf("Content-Type = %q", ct)
	}
	items := listItems(t, rec)
	if len(items) != 2 || items[0]["slug"] != "second" || items[1]["slug"] != "first" {
		t.Fatalf("items = %v, want [second first]", items)
	}
	if strings.Contains(rec.Body.String(), "SECRET-BODY") {
		t.Error("list must not include post content")
	}
	for _, key := range []string{"content", "status", "id", "deleted_at", "created_at", "updated_at"} {
		if _, present := items[0][key]; present {
			t.Errorf("list item has %q, which is not part of the public summary", key)
		}
	}
	for key, want := range map[string]any{
		"title": "Second", "slug": "second", "excerpt": "excerpt of second", "cover_image_url": "https://example.com/second.png",
	} {
		if items[0][key] != want {
			t.Errorf("%s = %v, want %v", key, items[0][key], want)
		}
	}
	if items[0]["published_at"] == nil {
		t.Error("published_at must be present")
	}

	meta := decodeMap(t, rec)["meta"].(map[string]any)
	want := map[string]float64{"page": 1, "limit": 20, "total": 2, "total_pages": 1}
	for k, v := range want {
		if meta[k] != v {
			t.Errorf("meta.%s = %v, want %v", k, meta[k], v)
		}
	}
}

func TestPostHandler_List_EmptyIsOKWithEmptyArray(t *testing.T) {
	env := newPublicPostEnv(t)
	env.seed(t, "Only Draft", "only-draft", model.PostStatusDraft)

	rec := env.get(t, "/posts")

	requireStatus(t, rec, http.StatusOK, "")
	if !strings.Contains(rec.Body.String(), `"data":[]`) {
		t.Errorf("body = %s, want data to be [] and not null", rec.Body.String())
	}
	meta := decodeMap(t, rec)["meta"].(map[string]any)
	if meta["total"] != float64(0) || meta["page"] != float64(1) || meta["limit"] != float64(20) || meta["total_pages"] != float64(0) {
		t.Errorf("meta = %v", meta)
	}
}

func TestPostHandler_List_PageBeyondTheEndIsEmptyData(t *testing.T) {
	env := newPublicPostEnv(t)
	env.seed(t, "One", "one", model.PostStatusPublished)
	env.seed(t, "Two", "two", model.PostStatusPublished)
	env.seed(t, "Three", "three", model.PostStatusPublished)

	rec := env.get(t, "/posts?page=5&limit=2")

	requireStatus(t, rec, http.StatusOK, "")
	if items := listItems(t, rec); len(items) != 0 {
		t.Errorf("items = %v, want none", items)
	}
	if !strings.Contains(rec.Body.String(), `"data":[]`) {
		t.Errorf("body = %s, want data to be []", rec.Body.String())
	}
	meta := decodeMap(t, rec)["meta"].(map[string]any)
	if meta["page"] != float64(5) || meta["limit"] != float64(2) || meta["total"] != float64(3) || meta["total_pages"] != float64(2) {
		t.Errorf("meta = %v", meta)
	}
}

func TestPostHandler_List_Pagination(t *testing.T) {
	env := newPublicPostEnv(t)
	for i := 1; i <= 5; i++ {
		env.seed(t, "P"+strconv.Itoa(i), "p"+strconv.Itoa(i), model.PostStatusPublished)
	}

	rec := env.get(t, "/posts?page=2&limit=2")

	requireStatus(t, rec, http.StatusOK, "")
	items := listItems(t, rec)
	if len(items) != 2 || items[0]["slug"] != "p3" || items[1]["slug"] != "p2" {
		t.Errorf("items = %v, want [p3 p2]", items)
	}
}

func TestPostHandler_List_ClampsOutOfRangeParams(t *testing.T) {
	env := newPublicPostEnv(t)
	env.seed(t, "One", "one", model.PostStatusPublished)

	for _, tc := range []struct {
		query     string
		wantPage  float64
		wantLimit float64
	}{
		{"?limit=1000", 1, 100},
		{"?limit=0", 1, 20},
		{"?page=0", 1, 20},
		{"?page=-3&limit=abc", 1, 20},
	} {
		rec := env.get(t, "/posts"+tc.query)
		requireStatus(t, rec, http.StatusOK, "")
		meta := decodeMap(t, rec)["meta"].(map[string]any)
		if meta["page"] != tc.wantPage || meta["limit"] != tc.wantLimit {
			t.Errorf("%s: meta = %v, want page %v limit %v", tc.query, meta, tc.wantPage, tc.wantLimit)
		}
	}
}

func TestPostHandler_List_RepoErrorIs500WithoutLeaking(t *testing.T) {
	env := newPublicPostEnv(t)
	env.repo.ListPublishedFunc = func(context.Context, int, int) ([]model.PostSummary, int64, error) {
		return nil, 0, errors.New("db down secret-detail")
	}

	rec := env.get(t, "/posts")

	requireStatus(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR")
	if strings.Contains(rec.Body.String(), "secret-detail") {
		t.Errorf("body leaks the cause: %s", rec.Body.String())
	}
}

// ============================================================
// GET /posts/:slug
// ============================================================

func TestPostHandler_Get_ReturnsFullPostWithoutDeletedAt(t *testing.T) {
	env := newPublicPostEnv(t)
	created := env.seed(t, "Hello World", "hello-world", model.PostStatusPublished)

	rec := env.get(t, "/posts/hello-world")

	requireStatus(t, rec, http.StatusOK, "")
	body := decodeMap(t, rec)
	if body["content"] != "SECRET-BODY of hello-world" {
		t.Errorf("content = %v, want the full body", body["content"])
	}
	if _, present := body["deleted_at"]; present {
		t.Error("deleted_at must never be exposed")
	}
	for key, want := range map[string]any{
		"id": float64(created.ID), "title": "Hello World", "slug": "hello-world", "status": "published",
		"excerpt": "excerpt of hello-world", "cover_image_url": "https://example.com/hello-world.png",
	} {
		if body[key] != want {
			t.Errorf("%s = %v, want %v", key, body[key], want)
		}
	}
	if body["published_at"] == nil {
		t.Error("published_at must be set")
	}
}

// Draft, soft-deleted and missing slugs must be indistinguishable: same status, same bytes.
func TestPostHandler_Get_NotFoundBodiesAreByteIdentical(t *testing.T) {
	env := newPublicPostEnv(t)
	env.seed(t, "Secret Draft", "secret-draft", model.PostStatusDraft)
	gone := env.seed(t, "Gone Post", "gone-post", model.PostStatusPublished)
	env.repo.MarkDeleted(gone.ID)

	draft := env.get(t, "/posts/secret-draft")
	deleted := env.get(t, "/posts/gone-post")
	missing := env.get(t, "/posts/never-existed")

	requireStatus(t, draft, http.StatusNotFound, "NOT_FOUND")
	requireStatus(t, deleted, http.StatusNotFound, "NOT_FOUND")
	requireStatus(t, missing, http.StatusNotFound, "NOT_FOUND")

	if draft.Body.String() != missing.Body.String() {
		t.Errorf("draft body %q != missing body %q", draft.Body.String(), missing.Body.String())
	}
	if deleted.Body.String() != missing.Body.String() {
		t.Errorf("deleted body %q != missing body %q", deleted.Body.String(), missing.Body.String())
	}
	if draft.Header().Get(echo.HeaderContentType) != missing.Header().Get(echo.HeaderContentType) ||
		deleted.Header().Get(echo.HeaderContentType) != missing.Header().Get(echo.HeaderContentType) {
		t.Error("Content-Type differs between not-found responses")
	}
	for _, leak := range []string{"secret-draft", "gone-post", "Secret Draft", "SECRET-BODY"} {
		if strings.Contains(missing.Body.String(), leak) {
			t.Errorf("not-found body echoes %q", leak)
		}
	}
}

func TestPostHandler_Get_RepoErrorIs500WithoutLeaking(t *testing.T) {
	env := newPublicPostEnv(t)
	env.repo.GetPublishedBySlugFunc = func(context.Context, string) (*model.Post, error) {
		return nil, errors.New("db down secret-detail")
	}

	rec := env.get(t, "/posts/anything")

	requireStatus(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR")
	if strings.Contains(rec.Body.String(), "secret-detail") {
		t.Errorf("body leaks the cause: %s", rec.Body.String())
	}
}

// ============================================================
// No auth
// ============================================================

func TestPostHandler_WorksWithoutSessionAndIgnoresAuthInput(t *testing.T) {
	env := newPublicPostEnv(t)
	env.seed(t, "Open Post", "open-post", model.PostStatusPublished)
	env.seed(t, "Closed Draft", "closed-draft", model.PostStatusDraft)

	withJunkAuth := func(req *http.Request) {
		req.Header.Set("Authorization", "Bearer not-a-real-token")
		req.AddCookie(&http.Cookie{Name: "session", Value: "not-a-real-session"})
	}

	for _, path := range []string{"/posts", "/posts/open-post"} {
		requireStatus(t, env.get(t, path), http.StatusOK, "")
		requireStatus(t, env.get(t, path, withJunkAuth), http.StatusOK, "")
	}

	// Credentials never widen what is visible: a draft is still not found.
	requireStatus(t, env.get(t, "/posts/closed-draft", withJunkAuth), http.StatusNotFound, "NOT_FOUND")
}
