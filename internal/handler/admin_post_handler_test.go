package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/alexedwards/scs/v2"
	"github.com/labstack/echo/v4"

	"github.com/ssr0016/ssr-blog/internal/apperror"
	"github.com/ssr0016/ssr-blog/internal/middleware"
	"github.com/ssr0016/ssr-blog/internal/model"
	"github.com/ssr0016/ssr-blog/internal/repository"
	"github.com/ssr0016/ssr-blog/internal/service"
	"github.com/ssr0016/ssr-blog/internal/validator"
)

const testActorID int64 = 42

// ============================================================
// Test environment
// ============================================================

type postEnv struct {
	ctx     context.Context
	handler http.Handler
	repo    *repository.MockPostRepo
}

// newPostEnv wires the real service and validator around a mock repository. A tiny middleware
// stands in for RequireAuth by putting testActorID into the session; the session manager wraps
// Echo because scs panics when a context has no session data.
func newPostEnv(t *testing.T) *postEnv {
	t.Helper()

	sm := scs.New()
	repo := repository.NewMockPostRepo()
	h := NewAdminPostHandler(service.NewPostService(repo), sm)

	e := echo.New()
	e.Validator = validator.New()
	e.HTTPErrorHandler = apperror.ErrorHandler
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			sm.Put(c.Request().Context(), middleware.UserIDKey, testActorID)
			return next(c)
		}
	})
	e.POST("/admin/posts", h.Create)
	e.GET("/admin/posts/:id", h.Get)

	return &postEnv{ctx: t.Context(), handler: sm.LoadAndSave(e), repo: repo}
}

func (env *postEnv) do(method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(env.ctx, method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	rec := httptest.NewRecorder()
	env.handler.ServeHTTP(rec, req)
	return rec
}

func (env *postEnv) post(body string) *httptest.ResponseRecorder {
	return env.do(http.MethodPost, "/admin/posts", body)
}

func (env *postEnv) get(id string) *httptest.ResponseRecorder {
	return env.do(http.MethodGet, "/admin/posts/"+id, "")
}

// nothingStored reports whether the repository holds no post at all (mock ids start at 1).
func (env *postEnv) nothingStored(t *testing.T) bool {
	t.Helper()
	got, err := env.repo.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	return got == nil
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

func decodeMap(t *testing.T, rec *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("response is not a JSON object: %v\nbody: %s", err, rec.Body.String())
	}
	return m
}

func requireStatus(t *testing.T, rec *httptest.ResponseRecorder, status int, errCode string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, status, rec.Body.String())
	}
	if errCode != "" {
		if got := decodeMap(t, rec)["error"]; got != errCode {
			t.Fatalf("error code = %v, want %s\nbody: %s", got, errCode, rec.Body.String())
		}
	}
}

// ============================================================
// Audit log capture
// ============================================================

type logCapture struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (l *logCapture) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buf.Write(p)
}

// captureLogs redirects the default slog logger for the test. Tests using it must not run in parallel.
func captureLogs(t *testing.T) *logCapture {
	t.Helper()
	prev := slog.Default()
	lc := &logCapture{}
	slog.SetDefault(slog.New(slog.NewJSONHandler(lc, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })
	return lc
}

func (l *logCapture) auditRecords(t *testing.T) []map[string]any {
	t.Helper()
	l.mu.Lock()
	defer l.mu.Unlock()

	var out []map[string]any
	for line := range strings.SplitSeq(strings.TrimSpace(l.buf.String()), "\n") {
		if line == "" {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("log line is not JSON: %q", line)
		}
		if rec["msg"] == "audit" {
			out = append(out, rec)
		}
	}
	return out
}

// ============================================================
// POST /admin/posts: success
// ============================================================

func TestAdminPostHandler_Create_Success(t *testing.T) {
	env := newPostEnv(t)

	content := "<script>alert(1)</script>\n\n**md**"
	rec := env.post(mustJSON(t, map[string]any{
		"title":           "Hello World",
		"content":         content,
		"excerpt":         "short",
		"cover_image_url": "https://example.com/a.png",
	}))

	requireStatus(t, rec, http.StatusCreated, "")
	if ct := rec.Header().Get(echo.HeaderContentType); !strings.HasPrefix(ct, echo.MIMEApplicationJSON) {
		t.Errorf("Content-Type = %q", ct)
	}
	body := decodeMap(t, rec)
	if body["id"] == nil || body["id"].(float64) == 0 {
		t.Errorf("id = %v, want non-zero", body["id"])
	}
	want := map[string]any{
		"title":           "Hello World",
		"slug":            "hello-world",
		"content":         content,
		"excerpt":         "short",
		"cover_image_url": "https://example.com/a.png",
		"status":          "draft",
	}
	for k, v := range want {
		if body[k] != v {
			t.Errorf("%s = %v, want %v", k, body[k], v)
		}
	}
	if v, present := body["published_at"]; !present || v != nil {
		t.Errorf("published_at = %v (present=%v), want explicit null for a draft", v, present)
	}
	if _, present := body["deleted_at"]; present {
		t.Error("deleted_at must never be exposed")
	}
	for _, k := range []string{"created_at", "updated_at"} {
		if s, _ := body[k].(string); s == "" {
			t.Errorf("%s missing", k)
		}
	}
}

func TestAdminPostHandler_Create_Published(t *testing.T) {
	env := newPostEnv(t)

	rec := env.post(`{"title":"Live","content":"C","status":"published"}`)

	requireStatus(t, rec, http.StatusCreated, "")
	body := decodeMap(t, rec)
	if body["status"] != "published" {
		t.Errorf("status = %v", body["status"])
	}
	if s, _ := body["published_at"].(string); s == "" {
		t.Errorf("published_at = %v, want a timestamp", body["published_at"])
	}
}

func TestAdminPostHandler_Create_IgnoresSystemFields(t *testing.T) {
	env := newPostEnv(t)

	rec := env.post(`{"title":"Real Title","content":"C","slug":"hacked","id":999,"published_at":"2020-01-01T00:00:00Z","deleted_at":"2020-01-01T00:00:00Z","status":"draft"}`)

	requireStatus(t, rec, http.StatusCreated, "")
	body := decodeMap(t, rec)
	if body["slug"] != "real-title" {
		t.Errorf("slug = %v, want it derived from the title", body["slug"])
	}
	if body["id"].(float64) == 999 {
		t.Error("client-supplied id must be ignored")
	}
	if body["published_at"] != nil {
		t.Errorf("published_at = %v, client-supplied value must be ignored", body["published_at"])
	}
}

func TestAdminPostHandler_Create_SameTitleTwiceBothSucceed(t *testing.T) {
	env := newPostEnv(t)
	body := `{"title":"Same","content":"C"}`

	first := decodeMap(t, env.post(body))
	rec := env.post(body)

	requireStatus(t, rec, http.StatusCreated, "")
	if first["slug"] != "same" || decodeMap(t, rec)["slug"] != "same-2" {
		t.Errorf("slugs = %v, %v", first["slug"], decodeMap(t, rec)["slug"])
	}
}

// ============================================================
// POST /admin/posts: validation (422), nothing stored
// ============================================================

func TestAdminPostHandler_Create_ValidationFailures(t *testing.T) {
	tests := []struct {
		name string
		body map[string]any
	}{
		{"title missing", map[string]any{"content": "C"}},
		{"title blank", map[string]any{"title": "   ", "content": "C"}},
		{"title 201 chars", map[string]any{"title": strings.Repeat("a", 201), "content": "C"}},
		{"content missing", map[string]any{"title": "T"}},
		{"content blank", map[string]any{"title": "T", "content": " \n\t "}},
		{"content 100001 chars", map[string]any{"title": "T", "content": strings.Repeat("a", 100001)}},
		{"excerpt 501 chars", map[string]any{"title": "T", "content": "C", "excerpt": strings.Repeat("a", 501)}},
		{"cover 2049 chars", map[string]any{"title": "T", "content": "C", "cover_image_url": "https://x.io/" + strings.Repeat("a", 2049-len("https://x.io/"))}},
		{"cover javascript scheme", map[string]any{"title": "T", "content": "C", "cover_image_url": "javascript:alert(1)"}},
		{"cover ftp scheme", map[string]any{"title": "T", "content": "C", "cover_image_url": "ftp://x.io/a.png"}},
		{"cover not a url", map[string]any{"title": "T", "content": "C", "cover_image_url": "not a url"}},
		{"status archived", map[string]any{"title": "T", "content": "C", "status": "archived"}},
		{"status wrong case", map[string]any{"title": "T", "content": "C", "status": "Published"}},
		{"title without letters or digits", map[string]any{"title": "!!!", "content": "C"}},
		{"non-latin title has no slug", map[string]any{"title": "日本語のタイトル", "content": "C"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newPostEnv(t)

			rec := env.post(mustJSON(t, tt.body))

			requireStatus(t, rec, http.StatusUnprocessableEntity, string(apperror.CodeValidation))
			if !env.nothingStored(t) {
				t.Error("a rejected request must not store anything")
			}
		})
	}
}

func TestAdminPostHandler_Create_EmptyBodyIsValidationError(t *testing.T) {
	env := newPostEnv(t)

	rec := env.do(http.MethodPost, "/admin/posts", "")

	requireStatus(t, rec, http.StatusUnprocessableEntity, string(apperror.CodeValidation))
	if !env.nothingStored(t) {
		t.Error("nothing may be stored")
	}
}

// Boundary values must be accepted: limits count characters, not bytes.
func TestAdminPostHandler_Create_AcceptsFieldsAtTheLimit(t *testing.T) {
	env := newPostEnv(t)

	rec := env.post(mustJSON(t, map[string]any{
		"title":           strings.Repeat("é", 200),
		"content":         strings.Repeat("日", 100000),
		"excerpt":         strings.Repeat("a", 500),
		"cover_image_url": "https://x.io/" + strings.Repeat("a", 2048-len("https://x.io/")),
	}))

	requireStatus(t, rec, http.StatusCreated, "")
	body := decodeMap(t, rec)
	if len(body["slug"].(string)) > service.MaxSlugLen {
		t.Errorf("slug len = %d, want <= %d", len(body["slug"].(string)), service.MaxSlugLen)
	}
}

// ============================================================
// POST /admin/posts: bad requests (400) and server errors
// ============================================================

func TestAdminPostHandler_Create_MalformedJSON(t *testing.T) {
	for name, body := range map[string]string{
		"truncated":     `{"title":"T","content":`,
		"not json":      `title=T&content=C`,
		"wrong type":    `{"title":123,"content":"C"}`,
		"array at root": `[]`,
	} {
		t.Run(name, func(t *testing.T) {
			env := newPostEnv(t)

			rec := env.post(body)

			requireStatus(t, rec, http.StatusBadRequest, string(apperror.CodeBadRequest))
			if !env.nothingStored(t) {
				t.Error("nothing may be stored")
			}
		})
	}
}

func TestAdminPostHandler_Create_UnsupportedContentType(t *testing.T) {
	env := newPostEnv(t)
	req := httptest.NewRequestWithContext(env.ctx, http.MethodPost, "/admin/posts", strings.NewReader(`{"title":"T","content":"C"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMETextPlain)
	rec := httptest.NewRecorder()

	env.handler.ServeHTTP(rec, req)

	requireStatus(t, rec, http.StatusBadRequest, string(apperror.CodeBadRequest))
	if !env.nothingStored(t) {
		t.Error("nothing may be stored")
	}
}

func TestAdminPostHandler_Create_RepoErrorIsGenericInternal(t *testing.T) {
	env := newPostEnv(t)
	env.repo.CreateFunc = func(context.Context, model.Post) (*model.Post, error) {
		return nil, errors.New("connection refused: 10.0.0.5:5437 secret-detail")
	}

	rec := env.post(`{"title":"T","content":"C"}`)

	requireStatus(t, rec, http.StatusInternalServerError, string(apperror.CodeInternal))
	if strings.Contains(rec.Body.String(), "secret-detail") || strings.Contains(rec.Body.String(), "5437") {
		t.Errorf("response leaks the cause: %s", rec.Body.String())
	}
}

// ============================================================
// GET /admin/posts/:id
// ============================================================

func TestAdminPostHandler_Get_Success(t *testing.T) {
	env := newPostEnv(t)
	created := decodeMap(t, env.post(`{"title":"Find Me","content":"# Body","excerpt":"e","status":"published"}`))

	rec := env.get(strconv.Itoa(int(created["id"].(float64))))

	requireStatus(t, rec, http.StatusOK, "")
	body := decodeMap(t, rec)
	for _, k := range []string{"id", "title", "slug", "content", "excerpt", "cover_image_url", "status", "published_at", "created_at", "updated_at"} {
		if _, present := body[k]; !present {
			t.Errorf("field %q missing from response", k)
		}
	}
	if _, present := body["deleted_at"]; present {
		t.Error("deleted_at must never be exposed")
	}
	if body["title"] != "Find Me" || body["slug"] != "find-me" || body["content"] != "# Body" || body["status"] != "published" {
		t.Errorf("unexpected body: %v", body)
	}
}

func TestAdminPostHandler_Get_ReturnsDrafts(t *testing.T) {
	env := newPostEnv(t)
	created := decodeMap(t, env.post(`{"title":"Secret Draft","content":"C"}`))

	rec := env.get(strconv.Itoa(int(created["id"].(float64))))

	requireStatus(t, rec, http.StatusOK, "")
	if decodeMap(t, rec)["status"] != "draft" {
		t.Error("admin must be able to view drafts")
	}
}

func TestAdminPostHandler_Get_InvalidID(t *testing.T) {
	for _, id := range []string{"abc", "1.5", "0", "-1", "99999999999999999999", "1%20OR%201=1"} {
		t.Run(id, func(t *testing.T) {
			env := newPostEnv(t)

			requireStatus(t, env.get(id), http.StatusBadRequest, string(apperror.CodeBadRequest))
		})
	}
}

func TestAdminPostHandler_Get_UnknownID(t *testing.T) {
	env := newPostEnv(t)

	requireStatus(t, env.get("999"), http.StatusNotFound, string(apperror.CodeNotFound))
}

// A soft-deleted post is invisible to the repository read, so the handler sees it exactly like an
// unknown id. The real filter is covered by the repository integration tests.
func TestAdminPostHandler_Get_DeletedPostLooksLikeUnknown(t *testing.T) {
	env := newPostEnv(t)
	env.repo.GetByIDFunc = func(context.Context, int64) (*model.Post, error) { return nil, nil }

	deleted := env.get("1")
	unknown := env.get("999")

	requireStatus(t, deleted, http.StatusNotFound, string(apperror.CodeNotFound))
	if deleted.Body.String() != unknown.Body.String() {
		t.Errorf("bodies differ:\n%s\n%s", deleted.Body.String(), unknown.Body.String())
	}
}

func TestAdminPostHandler_Get_RepoErrorIsGenericInternal(t *testing.T) {
	env := newPostEnv(t)
	env.repo.GetByIDFunc = func(context.Context, int64) (*model.Post, error) {
		return nil, errors.New("db down secret-detail")
	}

	rec := env.get("1")

	requireStatus(t, rec, http.StatusInternalServerError, string(apperror.CodeInternal))
	if strings.Contains(rec.Body.String(), "secret-detail") {
		t.Errorf("response leaks the cause: %s", rec.Body.String())
	}
}

// ============================================================
// Audit log
// ============================================================

func TestAdminPostHandler_Create_WritesAuditEvent(t *testing.T) {
	logs := captureLogs(t)
	env := newPostEnv(t)

	rec := env.post(`{"title":"Audited Post","content":"C"}`)

	requireStatus(t, rec, http.StatusCreated, "")
	created := decodeMap(t, rec)

	records := logs.auditRecords(t)
	if len(records) != 1 {
		t.Fatalf("got %d audit records, want exactly 1: %v", len(records), records)
	}
	got := records[0]
	if got["audit"] != true {
		t.Errorf("audit = %v, want true", got["audit"])
	}
	if got["actor_user_id"] != float64(testActorID) {
		t.Errorf("actor_user_id = %v, want %d", got["actor_user_id"], testActorID)
	}
	if got["action"] != "post.create" {
		t.Errorf("action = %v, want post.create", got["action"])
	}
	if got["post_id"] != created["id"] {
		t.Errorf("post_id = %v, want %v", got["post_id"], created["id"])
	}
	if got["slug"] != "audited-post" {
		t.Errorf("slug = %v, want audited-post", got["slug"])
	}
	if got["level"] != "INFO" {
		t.Errorf("level = %v, want INFO", got["level"])
	}
	for _, forbidden := range []string{"content", "title", "excerpt"} {
		if _, present := got[forbidden]; present {
			t.Errorf("audit event must not carry %q", forbidden)
		}
	}
}

func TestAdminPostHandler_NoAuditEventWithoutASuccessfulWrite(t *testing.T) {
	logs := captureLogs(t)
	env := newPostEnv(t)

	env.post(`{"title":"","content":"C"}`)    // validation failure
	env.post(`{"title":`)                     // malformed
	env.post(`{"title":"!!!","content":"C"}`) // no usable slug
	env.get("999")                            // not found
	env.get("abc")                            // bad id
	env.repo.CreateFunc = func(context.Context, model.Post) (*model.Post, error) { return nil, errors.New("boom") }
	env.post(`{"title":"T","content":"C"}`) // repository failure

	created := newPostEnv(t)
	created.post(`{"title":"Readable","content":"C"}`)
	before := len(logs.auditRecords(t))
	created.get("1") // a successful read is not an audited write

	if got := len(logs.auditRecords(t)); got != before {
		t.Errorf("GET wrote %d audit records, want 0", got-before)
	}
	if before != 1 {
		t.Errorf("only the one successful create should be audited, got %d records", before)
	}
}
