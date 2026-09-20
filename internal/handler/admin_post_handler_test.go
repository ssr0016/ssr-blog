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
	e.GET("/admin/posts", h.List)
	e.PUT("/admin/posts/:id", h.Update)
	e.GET("/admin/posts/:id", h.Get)
	e.DELETE("/admin/posts/:id", h.Delete)

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

func (env *postEnv) list(query string) *httptest.ResponseRecorder {
	path := "/admin/posts"
	if query != "" {
		path += "?" + query
	}
	return env.do(http.MethodGet, path, "")
}

func (env *postEnv) put(id, body string) *httptest.ResponseRecorder {
	return env.do(http.MethodPut, "/admin/posts/"+id, body)
}

func (env *postEnv) get(id string) *httptest.ResponseRecorder {
	return env.do(http.MethodGet, "/admin/posts/"+id, "")
}

func (env *postEnv) del(id string) *httptest.ResponseRecorder {
	return env.do(http.MethodDelete, "/admin/posts/"+id, "")
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

// ============================================================
// GET /admin/posts
// ============================================================

// seedAdminPosts creates a draft and a published post through the handler, in that order.
func seedAdminPosts(t *testing.T, env *postEnv) {
	t.Helper()
	requireStatus(t, env.post(`{"title":"Draft One","content":"SECRET-BODY-1","excerpt":"e1"}`), http.StatusCreated, "")
	requireStatus(t, env.post(`{"title":"Live One","content":"SECRET-BODY-2","status":"published"}`), http.StatusCreated, "")
}

func adminListItems(t *testing.T, rec *httptest.ResponseRecorder) []map[string]any {
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

func TestAdminPostHandler_List_ReturnsAllStatusesNewestFirstWithoutContent(t *testing.T) {
	env := newPostEnv(t)
	seedAdminPosts(t, env)

	rec := env.list("")

	requireStatus(t, rec, http.StatusOK, "")
	if ct := rec.Header().Get(echo.HeaderContentType); !strings.HasPrefix(ct, echo.MIMEApplicationJSON) {
		t.Errorf("Content-Type = %q", ct)
	}
	items := adminListItems(t, rec)
	if len(items) != 2 || items[0]["slug"] != "live-one" || items[1]["slug"] != "draft-one" {
		t.Fatalf("items = %v, want [live-one draft-one]", items)
	}
	if strings.Contains(rec.Body.String(), "SECRET-BODY") {
		t.Error("list must not include post content")
	}
	for _, key := range []string{"content", "deleted_at"} {
		if _, present := items[0][key]; present {
			t.Errorf("list item has %q", key)
		}
	}
	for _, key := range []string{"id", "title", "slug", "excerpt", "cover_image_url", "status", "published_at", "created_at", "updated_at"} {
		if _, present := items[0][key]; !present {
			t.Errorf("list item is missing %q", key)
		}
	}
	if items[0]["status"] != "published" || items[1]["status"] != "draft" {
		t.Errorf("statuses = %v, %v", items[0]["status"], items[1]["status"])
	}
	if items[1]["published_at"] != nil {
		t.Errorf("draft published_at = %v, want null", items[1]["published_at"])
	}

	meta := decodeMap(t, rec)["meta"].(map[string]any)
	for k, v := range map[string]float64{"page": 1, "limit": 20, "total": 2, "total_pages": 1} {
		if meta[k] != v {
			t.Errorf("meta.%s = %v, want %v", k, meta[k], v)
		}
	}
}

func TestAdminPostHandler_List_StatusFilter(t *testing.T) {
	env := newPostEnv(t)
	seedAdminPosts(t, env)

	for status, wantSlug := range map[string]string{"draft": "draft-one", "published": "live-one"} {
		rec := env.list("status=" + status)
		requireStatus(t, rec, http.StatusOK, "")
		items := adminListItems(t, rec)
		if len(items) != 1 || items[0]["slug"] != wantSlug {
			t.Errorf("status=%s: items = %v, want only %s", status, items, wantSlug)
		}
		if total := decodeMap(t, rec)["meta"].(map[string]any)["total"]; total != float64(1) {
			t.Errorf("status=%s: meta.total = %v, want 1", status, total)
		}
	}

	rec := env.list("status=")
	requireStatus(t, rec, http.StatusOK, "")
	if items := adminListItems(t, rec); len(items) != 2 {
		t.Errorf("empty status: %d items, want all 2", len(items))
	}
}

func TestAdminPostHandler_List_UnknownStatusIs422(t *testing.T) {
	env := newPostEnv(t)
	seedAdminPosts(t, env)

	for _, status := range []string{"bogus", "Draft", "deleted", "all"} {
		rec := env.list("status=" + status)
		requireStatus(t, rec, http.StatusUnprocessableEntity, "VALIDATION_ERROR")
		if strings.Contains(rec.Body.String(), "SECRET-BODY") {
			t.Errorf("status=%s: error body carries post data", status)
		}
	}
}

func TestAdminPostHandler_List_EmptyIsOKWithEmptyArray(t *testing.T) {
	env := newPostEnv(t)

	rec := env.list("")

	requireStatus(t, rec, http.StatusOK, "")
	if !strings.Contains(rec.Body.String(), `"data":[]`) {
		t.Errorf("body = %s, want data to be [] and not null", rec.Body.String())
	}
	meta := decodeMap(t, rec)["meta"].(map[string]any)
	if meta["total"] != float64(0) || meta["page"] != float64(1) || meta["limit"] != float64(20) || meta["total_pages"] != float64(0) {
		t.Errorf("meta = %v", meta)
	}
}

func TestAdminPostHandler_List_PageBeyondTheEndIsEmptyData(t *testing.T) {
	env := newPostEnv(t)
	seedAdminPosts(t, env)

	rec := env.list("page=5&limit=1")

	requireStatus(t, rec, http.StatusOK, "")
	if !strings.Contains(rec.Body.String(), `"data":[]`) {
		t.Errorf("body = %s, want data to be []", rec.Body.String())
	}
	meta := decodeMap(t, rec)["meta"].(map[string]any)
	if meta["page"] != float64(5) || meta["limit"] != float64(1) || meta["total"] != float64(2) || meta["total_pages"] != float64(2) {
		t.Errorf("meta = %v", meta)
	}
}

// Out-of-range paging is clamped, not rejected, exactly like GET /admin/roles.
func TestAdminPostHandler_List_ClampsOutOfRangeParams(t *testing.T) {
	env := newPostEnv(t)
	seedAdminPosts(t, env)

	for _, tc := range []struct {
		query     string
		wantPage  float64
		wantLimit float64
	}{
		{"limit=0", 1, 20},
		{"limit=1000", 1, 100},
		{"limit=-5", 1, 20},
		{"page=0", 1, 20},
		{"page=-3&limit=abc", 1, 20},
	} {
		rec := env.list(tc.query)
		requireStatus(t, rec, http.StatusOK, "")
		meta := decodeMap(t, rec)["meta"].(map[string]any)
		if meta["page"] != tc.wantPage || meta["limit"] != tc.wantLimit {
			t.Errorf("%s: meta = %v, want page %v limit %v", tc.query, meta, tc.wantPage, tc.wantLimit)
		}
	}
}

func TestAdminPostHandler_List_Pagination(t *testing.T) {
	env := newPostEnv(t)
	for i := 1; i <= 5; i++ {
		requireStatus(t, env.post(`{"title":"P`+strconv.Itoa(i)+`","content":"C"}`), http.StatusCreated, "")
	}

	rec := env.list("page=2&limit=2")

	requireStatus(t, rec, http.StatusOK, "")
	items := adminListItems(t, rec)
	if len(items) != 2 || items[0]["slug"] != "p3" || items[1]["slug"] != "p2" {
		t.Errorf("items = %v, want [p3 p2]", items)
	}
}

func TestAdminPostHandler_List_RepoErrorIs500WithoutLeaking(t *testing.T) {
	env := newPostEnv(t)
	env.repo.ListAdminFunc = func(context.Context, string, int, int) ([]model.AdminPostSummary, int64, error) {
		return nil, 0, errors.New("db down secret-detail")
	}

	rec := env.list("")

	requireStatus(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR")
	if strings.Contains(rec.Body.String(), "secret-detail") {
		t.Errorf("body leaks the cause: %s", rec.Body.String())
	}
}

// Audit events are for writes only; a read must not write one.
func TestAdminPostHandler_List_WritesNoAuditEvent(t *testing.T) {
	env := newPostEnv(t)
	seedAdminPosts(t, env)
	logs := captureLogs(t)

	requireStatus(t, env.list(""), http.StatusOK, "")

	if got := logs.auditRecords(t); len(got) != 0 {
		t.Errorf("audit records = %v, want none for a read", got)
	}
}

// ============================================================
// PUT /admin/posts/:id
// ============================================================

const validUpdate = `{"title":"Edited Title","content":"edited body","excerpt":"edited","cover_image_url":"https://example.com/e.png","status":"draft"}`

// seedDraft creates a draft titled "Original Title" (id 1 in a fresh env).
func seedDraft(t *testing.T, env *postEnv) map[string]any {
	t.Helper()
	rec := env.post(`{"title":"Original Title","content":"original body"}`)
	requireStatus(t, rec, http.StatusCreated, "")
	return decodeMap(t, rec)
}

func TestAdminPostHandler_Update_Success(t *testing.T) {
	env := newPostEnv(t)
	created := seedDraft(t, env)

	rec := env.put("1", validUpdate)

	requireStatus(t, rec, http.StatusOK, "")
	if ct := rec.Header().Get(echo.HeaderContentType); !strings.HasPrefix(ct, echo.MIMEApplicationJSON) {
		t.Errorf("Content-Type = %q", ct)
	}
	body := decodeMap(t, rec)
	for k, v := range map[string]any{
		"id": created["id"], "title": "Edited Title", "slug": "original-title", "content": "edited body",
		"excerpt": "edited", "cover_image_url": "https://example.com/e.png", "status": "draft",
	} {
		if body[k] != v {
			t.Errorf("%s = %v, want %v", k, body[k], v)
		}
	}
	if body["published_at"] != nil {
		t.Errorf("published_at = %v, want null for a draft", body["published_at"])
	}
	if _, present := body["deleted_at"]; present {
		t.Error("deleted_at must never be exposed")
	}

	// The change is really stored.
	got := decodeMap(t, env.get("1"))
	if got["title"] != "Edited Title" || got["slug"] != "original-title" {
		t.Errorf("stored post = %v", got)
	}
}

func TestAdminPostHandler_Update_SlugInBodyIsIgnored(t *testing.T) {
	env := newPostEnv(t)
	seedDraft(t, env)

	rec := env.put("1", `{"title":"Edited Title","content":"c","status":"draft","slug":"hacked","id":999,"published_at":"2020-01-01T00:00:00Z","deleted_at":"2020-01-01T00:00:00Z"}`)

	requireStatus(t, rec, http.StatusOK, "")
	body := decodeMap(t, rec)
	if body["slug"] != "original-title" {
		t.Errorf("slug = %v, want original-title: a slug never changes", body["slug"])
	}
	if body["id"] != float64(1) {
		t.Errorf("id = %v, the body must not be able to change it", body["id"])
	}
	if body["published_at"] != nil {
		t.Errorf("published_at = %v, a client-supplied value must be ignored", body["published_at"])
	}
	if got := decodeMap(t, env.get("1")); got["slug"] != "original-title" {
		t.Errorf("stored slug = %v", got["slug"])
	}
	if hacked, _ := env.repo.GetPublishedBySlug(context.Background(), "hacked"); hacked != nil {
		t.Error("nothing may be reachable under the client-supplied slug")
	}
}

func TestAdminPostHandler_Update_PublishStampsAndUnpublishClearsPublishedAt(t *testing.T) {
	env := newPostEnv(t)
	seedDraft(t, env)

	pub := decodeMap(t, env.put("1", `{"title":"T","content":"c","status":"published"}`))
	if s, _ := pub["published_at"].(string); s == "" || pub["status"] != "published" {
		t.Fatalf("after publish: status = %v, published_at = %v", pub["status"], pub["published_at"])
	}
	stamp := pub["published_at"]

	edit := decodeMap(t, env.put("1", `{"title":"T edited","content":"c","status":"published"}`))
	if edit["published_at"] != stamp {
		t.Errorf("editing a published post moved published_at: %v -> %v", stamp, edit["published_at"])
	}

	unpub := decodeMap(t, env.put("1", `{"title":"T","content":"c","status":"draft"}`))
	if unpub["published_at"] != nil || unpub["status"] != "draft" {
		t.Errorf("after unpublish: status = %v, published_at = %v", unpub["status"], unpub["published_at"])
	}
}

func TestAdminPostHandler_Update_InvalidIDIs400(t *testing.T) {
	env := newPostEnv(t)
	seedDraft(t, env)

	for _, id := range []string{"abc", "0", "-1", "1.5", "99999999999999999999"} {
		rec := env.put(id, validUpdate)
		requireStatus(t, rec, http.StatusBadRequest, "BAD_REQUEST")
	}
	if got := decodeMap(t, env.get("1")); got["title"] != "Original Title" {
		t.Errorf("a rejected request changed the post: %v", got["title"])
	}
}

func TestAdminPostHandler_Update_UnknownAndDeletedAre404(t *testing.T) {
	env := newPostEnv(t)
	seedDraft(t, env)
	requireStatus(t, env.put("999", validUpdate), http.StatusNotFound, "NOT_FOUND")

	env.repo.MarkDeleted(1)
	requireStatus(t, env.put("1", validUpdate), http.StatusNotFound, "NOT_FOUND")
}

func TestAdminPostHandler_Update_ValidationFailuresAre422AndChangeNothing(t *testing.T) {
	long := func(n int) string { return strings.Repeat("a", n) }
	tests := []struct {
		name string
		body map[string]any
	}{
		{"missing title", map[string]any{"content": "c", "status": "draft"}},
		{"empty title", map[string]any{"title": "", "content": "c", "status": "draft"}},
		{"blank title", map[string]any{"title": "   ", "content": "c", "status": "draft"}},
		{"title too long", map[string]any{"title": long(201), "content": "c", "status": "draft"}},
		{"missing content", map[string]any{"title": "T", "status": "draft"}},
		{"blank content", map[string]any{"title": "T", "content": " \n\t ", "status": "draft"}},
		{"content too long", map[string]any{"title": "T", "content": long(100001), "status": "draft"}},
		{"excerpt too long", map[string]any{"title": "T", "content": "c", "excerpt": long(501), "status": "draft"}},
		{"cover url not http", map[string]any{"title": "T", "content": "c", "cover_image_url": "javascript:alert(1)", "status": "draft"}},
		{"cover url too long", map[string]any{"title": "T", "content": "c", "cover_image_url": "https://example.com/" + long(2048), "status": "draft"}},
		{"missing status", map[string]any{"title": "T", "content": "c"}},
		{"empty status", map[string]any{"title": "T", "content": "c", "status": ""}},
		{"unknown status", map[string]any{"title": "T", "content": "c", "status": "archived"}},
		{"status wrong case", map[string]any{"title": "T", "content": "c", "status": "Published"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newPostEnv(t)
			seedDraft(t, env)

			rec := env.put("1", mustJSON(t, tt.body))

			requireStatus(t, rec, http.StatusUnprocessableEntity, "VALIDATION_ERROR")
			if got := decodeMap(t, env.get("1")); got["title"] != "Original Title" || got["content"] != "original body" {
				t.Errorf("a rejected update changed the post: %v", got)
			}
		})
	}
}

func TestAdminPostHandler_Update_MalformedBodyIs400(t *testing.T) {
	env := newPostEnv(t)
	seedDraft(t, env)

	for _, body := range []string{`{"title":`, `not json`, `[1,2]`, ``} {
		rec := env.put("1", body)
		if rec.Code != http.StatusBadRequest && rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("body %q: status = %d, want a 4xx client error\n%s", body, rec.Code, rec.Body.String())
		}
	}
	requireStatus(t, env.put("1", `{"title":`), http.StatusBadRequest, "BAD_REQUEST")
}

func TestAdminPostHandler_Update_RepoErrorIs500WithoutLeaking(t *testing.T) {
	env := newPostEnv(t)
	seedDraft(t, env)
	env.repo.UpdateFunc = func(context.Context, model.Post) (*model.Post, error) {
		return nil, errors.New("db down secret-detail")
	}

	rec := env.put("1", validUpdate)

	requireStatus(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR")
	if strings.Contains(rec.Body.String(), "secret-detail") {
		t.Errorf("response leaks the cause: %s", rec.Body.String())
	}
}

func TestAdminPostHandler_Update_WritesAuditEventWithTheRightAction(t *testing.T) {
	tests := []struct {
		name       string
		startLive  bool
		body       string
		wantAction string
	}{
		{"edit a draft", false, `{"title":"T2","content":"c","status":"draft"}`, "post.update"},
		{"edit a published post", true, `{"title":"T2","content":"c","status":"published"}`, "post.update"},
		{"publish a draft", false, `{"title":"T","content":"c","status":"published"}`, "post.publish"},
		{"unpublish a published post", true, `{"title":"T","content":"c","status":"draft"}`, "post.unpublish"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := newPostEnv(t)
			status := "draft"
			if tt.startLive {
				status = "published"
			}
			requireStatus(t, env.post(`{"title":"T","content":"c","status":"`+status+`"}`), http.StatusCreated, "")
			logs := captureLogs(t)

			rec := env.put("1", tt.body)

			requireStatus(t, rec, http.StatusOK, "")
			records := logs.auditRecords(t)
			if len(records) != 1 {
				t.Fatalf("got %d audit records, want exactly 1: %v", len(records), records)
			}
			got := records[0]
			if got["audit"] != true || got["level"] != "INFO" {
				t.Errorf("audit = %v, level = %v", got["audit"], got["level"])
			}
			if got["action"] != tt.wantAction {
				t.Errorf("action = %v, want %s", got["action"], tt.wantAction)
			}
			if got["actor_user_id"] != float64(testActorID) {
				t.Errorf("actor_user_id = %v, want %d", got["actor_user_id"], testActorID)
			}
			if got["post_id"] != float64(1) || got["slug"] != "t" {
				t.Errorf("post_id = %v, slug = %v", got["post_id"], got["slug"])
			}
			for _, forbidden := range []string{"content", "title", "excerpt"} {
				if _, present := got[forbidden]; present {
					t.Errorf("audit event must not carry %q", forbidden)
				}
			}
		})
	}
}

func TestAdminPostHandler_Update_NoAuditEventWithoutASuccessfulWrite(t *testing.T) {
	env := newPostEnv(t)
	seedDraft(t, env)
	logs := captureLogs(t)

	env.put("abc", validUpdate)                                     // bad id
	env.put("999", validUpdate)                                     // not found
	env.put("1", `{"title":`)                                       // malformed
	env.put("1", `{"title":"","content":"c","status":"draft"}`)     // validation
	env.put("1", `{"title":"T","content":"c","status":"archived"}`) // bad status
	env.repo.UpdateFunc = func(context.Context, model.Post) (*model.Post, error) { return nil, errors.New("boom") }
	env.put("1", validUpdate) // repository failure

	if got := logs.auditRecords(t); len(got) != 0 {
		t.Errorf("failed updates wrote %d audit records, want 0: %v", len(got), got)
	}
}

// ============================================================
// DELETE /admin/posts/:id
// ============================================================

func TestAdminPostHandler_Delete_Success204WithEmptyBody(t *testing.T) {
	env := newPostEnv(t)
	seedDraft(t, env)

	rec := env.del("1")

	requireStatus(t, rec, http.StatusNoContent, "")
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
	// It is really gone from reads.
	requireStatus(t, env.get("1"), http.StatusNotFound, string(apperror.CodeNotFound))
}

func TestAdminPostHandler_Delete_IsSoftNotHard(t *testing.T) {
	env := newPostEnv(t)
	seedDraft(t, env)
	var soft []int64
	env.repo.SoftDeleteFunc = func(_ context.Context, id int64) error {
		soft = append(soft, id)
		return nil
	}

	requireStatus(t, env.del("1"), http.StatusNoContent, "")

	if len(soft) != 1 || soft[0] != 1 {
		t.Errorf("SoftDelete calls = %v, want [1]", soft)
	}
}

func TestAdminPostHandler_Delete_InvalidIDIs400(t *testing.T) {
	env := newPostEnv(t)
	seedDraft(t, env)
	env.repo.SoftDeleteFunc = func(context.Context, int64) error {
		t.Error("SoftDelete must not be called for a bad id")
		return nil
	}

	for _, id := range []string{"abc", "0", "-1", "1.5", "99999999999999999999"} {
		requireStatus(t, env.del(id), http.StatusBadRequest, "BAD_REQUEST")
	}
}

func TestAdminPostHandler_Delete_UnknownAndAlreadyDeletedAre404(t *testing.T) {
	env := newPostEnv(t)
	seedDraft(t, env)
	requireStatus(t, env.del("1"), http.StatusNoContent, "")

	unknown := env.del("999")
	again := env.del("1")

	requireStatus(t, unknown, http.StatusNotFound, string(apperror.CodeNotFound))
	requireStatus(t, again, http.StatusNotFound, string(apperror.CodeNotFound))
	if unknown.Body.String() != again.Body.String() {
		t.Errorf("bodies differ:\n%s\n%s", unknown.Body.String(), again.Body.String())
	}
}

func TestAdminPostHandler_Delete_RepoErrorIs500WithoutLeaking(t *testing.T) {
	env := newPostEnv(t)
	seedDraft(t, env)
	env.repo.SoftDeleteFunc = func(context.Context, int64) error { return errors.New("db down secret-detail") }

	rec := env.del("1")

	requireStatus(t, rec, http.StatusInternalServerError, "INTERNAL_ERROR")
	if strings.Contains(rec.Body.String(), "secret-detail") {
		t.Errorf("response leaks the cause: %s", rec.Body.String())
	}
}

func TestAdminPostHandler_Delete_WritesAuditEvent(t *testing.T) {
	env := newPostEnv(t)
	seedDraft(t, env) // slug "original-title"
	logs := captureLogs(t)

	rec := env.del("1")

	requireStatus(t, rec, http.StatusNoContent, "")
	records := logs.auditRecords(t)
	if len(records) != 1 {
		t.Fatalf("got %d audit records, want exactly 1: %v", len(records), records)
	}
	got := records[0]
	if got["audit"] != true || got["level"] != "INFO" {
		t.Errorf("audit = %v, level = %v", got["audit"], got["level"])
	}
	if got["action"] != "post.delete" {
		t.Errorf("action = %v, want post.delete", got["action"])
	}
	if got["actor_user_id"] != float64(testActorID) {
		t.Errorf("actor_user_id = %v, want %d", got["actor_user_id"], testActorID)
	}
	if got["post_id"] != float64(1) || got["slug"] != "original-title" {
		t.Errorf("post_id = %v, slug = %v", got["post_id"], got["slug"])
	}
	for _, forbidden := range []string{"content", "title", "excerpt"} {
		if _, present := got[forbidden]; present {
			t.Errorf("audit event must not carry %q", forbidden)
		}
	}
}

func TestAdminPostHandler_Delete_NoAuditEventWithoutASuccessfulWrite(t *testing.T) {
	env := newPostEnv(t)
	seedDraft(t, env)
	logs := captureLogs(t)

	env.del("abc") // bad id
	env.del("999") // not found
	env.repo.SoftDeleteFunc = func(context.Context, int64) error { return errors.New("boom") }
	env.del("1") // repository failure
	env.repo.SoftDeleteFunc = nil
	env.del("1") // succeeds, audited once...
	before := len(logs.auditRecords(t))
	env.del("1") // ...and the repeat is a 404 and adds nothing

	if before != 1 {
		t.Errorf("only the one successful delete should be audited, got %d records", before)
	}
	if got := len(logs.auditRecords(t)); got != before {
		t.Errorf("a repeated delete wrote %d audit records, want 0", got-before)
	}
}
