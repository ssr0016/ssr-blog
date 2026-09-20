package router_test

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/ssr0016/ssr-blog/internal/handler"
	ourmiddleware "github.com/ssr0016/ssr-blog/internal/middleware"
	"github.com/ssr0016/ssr-blog/internal/model"
	"github.com/ssr0016/ssr-blog/internal/repository"
	"github.com/ssr0016/ssr-blog/internal/router"
	"github.com/ssr0016/ssr-blog/internal/service"
	"github.com/ssr0016/ssr-blog/internal/validator"
)

// Actors. Anonymous is the zero value.
const (
	anonymous int64 = 0
	adminID   int64 = 1
	editorID  int64 = 2
	userID    int64 = 3
	noRoleID  int64 = 4
	ghostID   int64 = 99 // has a session but no user row
)

const (
	csrfCookie = "csrf_token"
	csrfHeader = "X-CSRF-Token"
	csrfToken  = "test-csrf-token"

	seededTitle = "Seeded Post Title"
)

// ============================================================
// Harness
// ============================================================

type harness struct {
	handler http.Handler
	sm      *scs.SessionManager
	posts   *repository.MockPostRepo
}

// newHarness builds the real middleware chain from cmd/api/main.go (CSRF, sessions, error handler,
// validator) and the real router.Setup, backed by mock repositories. Post 1 is already stored.
// A new harness has a fresh rate limiter, because Setup creates its limiters.
func newHarness(t *testing.T) *harness {
	t.Helper()

	sm := scs.New()

	roleByUser := map[int64]string{adminID: "admin", editorID: "editor", userID: "user", noRoleID: ""}
	users := repository.NewMockUserRepo()
	users.GetWithRoleFunc = func(_ context.Context, id int64) (*model.User, error) {
		name, ok := roleByUser[id]
		if !ok {
			return nil, nil
		}
		u := &model.User{ID: id}
		if name != "" {
			u.RoleID = id
			u.Role = &model.Role{ID: id, Name: name}
		}
		return u, nil
	}

	posts := repository.NewMockPostRepo()
	if _, err := posts.Create(context.Background(), model.Post{
		Title: seededTitle, Slug: "seeded-post-title", Content: "seeded body", Status: model.PostStatusDraft,
	}); err != nil {
		t.Fatalf("seed post: %v", err)
	}
	adminPosts := handler.NewAdminPostHandler(service.NewPostService(posts), sm)

	e := echo.New()
	e.Validator = validator.New()
	e.HTTPErrorHandler = apperror.ErrorHandler
	e.Use(ourmiddleware.CSRFProtection())

	// Handlers this test never reaches are nil; Echo only stores their method values.
	router.Setup(e, sm, users, nil, nil, nil, nil, nil, nil, adminPosts, handler.NewPostHandler(service.NewPostService(posts)))

	return &harness{handler: sm.LoadAndSave(e), sm: sm, posts: posts}
}

// sessionCookie returns a logged-in session cookie for the given user id.
func (h *harness) sessionCookie(t *testing.T, id int64) *http.Cookie {
	t.Helper()
	ctx, err := h.sm.Load(context.Background(), "")
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	h.sm.Put(ctx, ourmiddleware.UserIDKey, id)
	token, _, err := h.sm.Commit(ctx)
	if err != nil {
		t.Fatalf("commit session: %v", err)
	}
	return &http.Cookie{Name: h.sm.Cookie.Name, Value: token}
}

type call struct {
	method string
	path   string
	body   string
	actor  int64
	csrf   string // header value; empty means the header is omitted
}

// do sends a request. The CSRF cookie is always present (as after any earlier response); the
// header is sent only when call.csrf is set.
func (h *harness) do(t *testing.T, c call) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), c.method, c.path, strings.NewReader(c.body))
	if c.body != "" {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	req.AddCookie(&http.Cookie{Name: csrfCookie, Value: csrfToken})
	if c.csrf != "" {
		req.Header.Set(csrfHeader, c.csrf)
	}
	if c.actor != anonymous {
		req.AddCookie(h.sessionCookie(t, c.actor))
	}
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, req)
	return rec
}

// createPost is the standard admin write: valid session and valid CSRF token.
func (h *harness) createPost(t *testing.T, actor int64, title string) *httptest.ResponseRecorder {
	t.Helper()
	return h.do(t, call{
		method: http.MethodPost, path: "/api/v1/admin/posts", actor: actor, csrf: csrfToken,
		body: `{"title":"` + title + `","content":"body"}`,
	})
}

// updatePost is the standard admin edit of the seeded post (id 1): valid session and valid CSRF token.
func (h *harness) updatePost(t *testing.T, actor int64, body string) *httptest.ResponseRecorder {
	t.Helper()
	return h.do(t, call{method: http.MethodPut, path: "/api/v1/admin/posts/1", actor: actor, csrf: csrfToken, body: body})
}

const routerUpdateBody = `{"title":"Edited Through Router","content":"edited","status":"draft"}`

// seededUnchanged reports whether post 1 still has its original title, content and status.
func (h *harness) seededUnchanged(t *testing.T) bool {
	t.Helper()
	got, err := h.posts.GetByID(context.Background(), 1)
	if err != nil || got == nil {
		t.Fatalf("GetByID(1) = %v, %v", got, err)
	}
	return got.Title == seededTitle && got.Content == "seeded body" && got.Status == model.PostStatusDraft
}

// deletePost is the standard admin delete of the seeded post (id 1): valid session and valid CSRF token.
func (h *harness) deletePost(t *testing.T, actor int64) *httptest.ResponseRecorder {
	t.Helper()
	return h.deletePostID(t, actor, 1)
}

func (h *harness) deletePostID(t *testing.T, actor, id int64) *httptest.ResponseRecorder {
	t.Helper()
	return h.do(t, call{method: http.MethodDelete, path: "/api/v1/admin/posts/" + strconv.FormatInt(id, 10), actor: actor, csrf: csrfToken})
}

// seededPresent reports whether post 1 is still readable (not deleted).
func (h *harness) seededPresent(t *testing.T) bool {
	t.Helper()
	got, err := h.posts.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetByID(1): %v", err)
	}
	return got != nil
}

// requireSeededSoftDeleted proves a delete was a real soft delete: the row is still stored with
// deleted_at set, and it no longer reads. A hard delete or a no-op would fail one of the two.
func requireSeededSoftDeleted(t *testing.T, h *harness) {
	t.Helper()
	if !h.posts.IsSoftDeleted(1) {
		t.Error("post 1 must still be stored with deleted_at set (soft delete)")
	}
	if h.seededPresent(t) {
		t.Error("post 1 must no longer be readable after the delete")
	}
}

// nothingCreated reports whether the only stored post is still the seeded one (id 1).
func (h *harness) nothingCreated(t *testing.T) bool {
	t.Helper()
	got, err := h.posts.GetByID(context.Background(), 2)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	return got == nil
}

func errorCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("not a JSON error body: %v\n%s", err, rec.Body.String())
	}
	return body.Error
}

// ============================================================
// Audit log capture (default slog logger)
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
// RBAC matrix
// ============================================================

// TestAdminPostRoutes_RBACMatrix is the harness later tickets extend: add a route row and the
// full actor x route matrix runs for it.
func TestAdminPostRoutes_RBACMatrix(t *testing.T) {
	routes := []struct {
		name     string
		method   string
		path     string
		body     string
		okStatus int
		// afterOK, when set, checks the admin's success really did its job. A status code alone is
		// not proof: denied rows pass even without the route, so a write row needs a state check.
		afterOK func(t *testing.T, h *harness)
	}{
		{"POST /admin/posts", http.MethodPost, "/api/v1/admin/posts", `{"title":"New Post","content":"body"}`, http.StatusCreated, nil},
		{"GET /admin/posts/:id", http.MethodGet, "/api/v1/admin/posts/1", "", http.StatusOK, nil},
		{"GET /admin/posts", http.MethodGet, "/api/v1/admin/posts", "", http.StatusOK, nil},
		{"GET /admin/posts?status=draft", http.MethodGet, "/api/v1/admin/posts?status=draft", "", http.StatusOK, nil},
		{"PUT /admin/posts/:id", http.MethodPut, "/api/v1/admin/posts/1", `{"title":"Edited Title","content":"body","status":"published"}`, http.StatusOK, nil},
		{"DELETE /admin/posts/:id", http.MethodDelete, "/api/v1/admin/posts/1", "", http.StatusNoContent, requireSeededSoftDeleted},
	}
	actors := []struct {
		name       string
		id         int64
		wantStatus int // 0 means the route's okStatus
		wantCode   string
	}{
		{"anonymous", anonymous, http.StatusUnauthorized, "UNAUTHORIZED"},
		{"session for a user that no longer exists", ghostID, http.StatusUnauthorized, "UNAUTHORIZED"},
		{"user role", userID, http.StatusForbidden, "FORBIDDEN"},
		{"editor role", editorID, http.StatusForbidden, "FORBIDDEN"},
		{"user without a role", noRoleID, http.StatusForbidden, "FORBIDDEN"},
		{"admin", adminID, 0, ""},
	}

	for _, r := range routes {
		for _, a := range actors {
			t.Run(r.name+"/"+a.name, func(t *testing.T) {
				h := newHarness(t)

				rec := h.do(t, call{method: r.method, path: r.path, body: r.body, actor: a.id, csrf: csrfToken})

				want := a.wantStatus
				if want == 0 {
					want = r.okStatus
				}
				if rec.Code != want {
					t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, want, rec.Body.String())
				}
				if a.wantCode != "" {
					if got := errorCode(t, rec); got != a.wantCode {
						t.Errorf("error code = %q, want %q", got, a.wantCode)
					}
					if strings.Contains(rec.Body.String(), seededTitle) {
						t.Error("a denied response must not carry post data")
					}
					if !h.nothingCreated(t) {
						t.Error("a denied request must not change anything")
					}
					if !h.seededPresent(t) {
						t.Error("a denied request must not delete the seeded post")
					}
				} else if r.afterOK != nil {
					r.afterOK(t, h)
				}
			})
		}
	}
}

// ============================================================
// CSRF
// ============================================================

func TestAdminPostRoutes_CSRF(t *testing.T) {
	tests := []struct {
		name string
		call call
	}{
		{"admin, header missing", call{actor: adminID, csrf: ""}},
		{"admin, header does not match the cookie", call{actor: adminID, csrf: "another-token"}},
		{"anonymous, header missing (CSRF is checked before auth)", call{actor: anonymous, csrf: ""}},
		{"editor, header missing", call{actor: editorID, csrf: ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			c := tt.call
			c.method, c.path, c.body = http.MethodPost, "/api/v1/admin/posts", `{"title":"CSRF","content":"body"}`

			rec := h.do(t, c)

			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403\nbody: %s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), "CSRF") {
				t.Errorf("want a CSRF rejection, got: %s", rec.Body.String())
			}
			if !h.nothingCreated(t) {
				t.Error("nothing may be created without a valid CSRF token")
			}
		})
	}

	t.Run("admin with a matching token succeeds", func(t *testing.T) {
		h := newHarness(t)

		if rec := h.createPost(t, adminID, "With Token"); rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201\nbody: %s", rec.Code, rec.Body.String())
		}
	})
}

// ============================================================
// Rate limit
// ============================================================

func TestAdminPostRoutes_RateLimitOnCreate(t *testing.T) {
	h := newHarness(t)

	// Burst is 10: the first ten writes pass, the eleventh is rejected.
	for i := 1; i <= 10; i++ {
		rec := h.createPost(t, adminID, "Burst")
		if rec.Code != http.StatusCreated {
			t.Fatalf("request %d: status = %d, want 201\nbody: %s", i, rec.Code, rec.Body.String())
		}
	}
	rec := h.createPost(t, adminID, "Burst")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("request 11: status = %d, want 429\nbody: %s", rec.Code, rec.Body.String())
	}
	if got := errorCode(t, rec); got != "RATE_LIMIT_EXCEEDED" {
		t.Errorf("error code = %q", got)
	}

	// 1 seeded + 10 created: the throttled request stored nothing.
	if got, _ := h.posts.GetByID(context.Background(), 12); got != nil {
		t.Error("the throttled request must not create a post")
	}
	if got, _ := h.posts.GetByID(context.Background(), 11); got == nil {
		t.Error("the tenth create should exist")
	}

	// Reads are not limited by the write limiter.
	read := h.do(t, call{method: http.MethodGet, path: "/api/v1/admin/posts/1", actor: adminID})
	if read.Code != http.StatusOK {
		t.Errorf("GET after the burst: status = %d, want 200", read.Code)
	}
}

// The limiter sits after auth on the route, so rejected callers cannot burn the admin's budget.
func TestAdminPostRoutes_DeniedRequestsDoNotConsumeRateLimit(t *testing.T) {
	h := newHarness(t)

	for i := range 30 {
		if rec := h.createPost(t, anonymous, "Spam"); rec.Code != http.StatusUnauthorized {
			t.Fatalf("anonymous request %d: status = %d, want 401", i, rec.Code)
		}
		if rec := h.createPost(t, editorID, "Spam"); rec.Code != http.StatusForbidden {
			t.Fatalf("editor request %d: status = %d, want 403", i, rec.Code)
		}
	}

	for i := 1; i <= 10; i++ {
		if rec := h.createPost(t, adminID, "Real"); rec.Code != http.StatusCreated {
			t.Fatalf("admin request %d: status = %d, want 201 (budget was consumed by denied callers)", i, rec.Code)
		}
	}
}

// ============================================================
// Audit log through the full stack
// ============================================================

func TestAdminPostRoutes_AuditActorComesFromTheSession(t *testing.T) {
	logs := captureLogs(t)
	h := newHarness(t)

	// Denied writes must leave no audit trail entry.
	h.createPost(t, anonymous, "Denied A")
	h.createPost(t, editorID, "Denied B")
	h.do(t, call{method: http.MethodPost, path: "/api/v1/admin/posts", actor: adminID, body: `{"title":"No CSRF","content":"c"}`})
	if got := len(logs.auditRecords(t)); got != 0 {
		t.Fatalf("denied requests produced %d audit records, want 0", got)
	}

	rec := h.createPost(t, adminID, "Audited Through Router")
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d\nbody: %s", rec.Code, rec.Body.String())
	}

	records := logs.auditRecords(t)
	if len(records) != 1 {
		t.Fatalf("got %d audit records, want 1: %v", len(records), records)
	}
	got := records[0]
	if got["audit"] != true || got["action"] != "post.create" {
		t.Errorf("record = %v", got)
	}
	if got["actor_user_id"] != float64(adminID) {
		t.Errorf("actor_user_id = %v, want %d", got["actor_user_id"], adminID)
	}
	if got["slug"] != "audited-through-router" {
		t.Errorf("slug = %v", got["slug"])
	}
}

// ============================================================
// PUT /admin/posts/:id: CSRF, rate limit, audit, denied writes
// ============================================================

// Denied callers must not be able to edit anything. The seeded post is the witness.
func TestAdminPostRoutes_DeniedUpdatesChangeNothing(t *testing.T) {
	for _, a := range []struct {
		name string
		id   int64
		want int
	}{
		{"anonymous", anonymous, http.StatusUnauthorized},
		{"ghost session", ghostID, http.StatusUnauthorized},
		{"user", userID, http.StatusForbidden},
		{"editor", editorID, http.StatusForbidden},
		{"no role", noRoleID, http.StatusForbidden},
	} {
		t.Run(a.name, func(t *testing.T) {
			h := newHarness(t)

			rec := h.updatePost(t, a.id, `{"title":"Hijacked","content":"hijacked","status":"published"}`)

			if rec.Code != a.want {
				t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, a.want, rec.Body.String())
			}
			if !h.seededUnchanged(t) {
				t.Error("a denied update must not change the post")
			}
		})
	}

	t.Run("admin does change it, proving the route is really mounted", func(t *testing.T) {
		h := newHarness(t)
		rec := h.updatePost(t, adminID, routerUpdateBody)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
		}
		if h.seededUnchanged(t) {
			t.Error("the admin's update was not applied")
		}
	})
}

func TestAdminPostRoutes_UpdateCSRF(t *testing.T) {
	tests := []struct {
		name string
		call call
	}{
		{"admin, header missing", call{actor: adminID, csrf: ""}},
		{"admin, header does not match the cookie", call{actor: adminID, csrf: "another-token"}},
		{"anonymous, header missing (CSRF is checked before auth)", call{actor: anonymous, csrf: ""}},
		{"editor, header missing", call{actor: editorID, csrf: ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			c := tt.call
			c.method, c.path, c.body = http.MethodPut, "/api/v1/admin/posts/1", routerUpdateBody

			rec := h.do(t, c)

			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403\nbody: %s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), "CSRF") {
				t.Errorf("want a CSRF rejection, got: %s", rec.Body.String())
			}
			if !h.seededUnchanged(t) {
				t.Error("nothing may change without a valid CSRF token")
			}
		})
	}

	t.Run("admin with a matching token succeeds", func(t *testing.T) {
		h := newHarness(t)
		if rec := h.updatePost(t, adminID, routerUpdateBody); rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200\nbody: %s", rec.Code, rec.Body.String())
		}
	})
}

func TestAdminPostRoutes_RateLimitOnUpdate(t *testing.T) {
	h := newHarness(t)

	// Burst is 10: the first ten writes pass, the eleventh is rejected.
	for i := 1; i <= 10; i++ {
		rec := h.updatePost(t, adminID, `{"title":"Edit `+strings.Repeat("x", i)+`","content":"c","status":"draft"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want 200\nbody: %s", i, rec.Code, rec.Body.String())
		}
	}
	rec := h.updatePost(t, adminID, `{"title":"Throttled","content":"c","status":"draft"}`)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("request 11: status = %d, want 429\nbody: %s", rec.Code, rec.Body.String())
	}
	if got := errorCode(t, rec); got != "RATE_LIMIT_EXCEEDED" {
		t.Errorf("error code = %q", got)
	}
	stored, _ := h.posts.GetByID(context.Background(), 1)
	if stored.Title == "Throttled" {
		t.Error("the throttled request must not be applied")
	}

	// Reads are not limited by the write limiter.
	if read := h.do(t, call{method: http.MethodGet, path: "/api/v1/admin/posts/1", actor: adminID}); read.Code != http.StatusOK {
		t.Errorf("GET after the burst: status = %d, want 200", read.Code)
	}
}

// Edits and creates draw from the same per-client write budget.
func TestAdminPostRoutes_UpdatesAndCreatesShareTheWriteBudget(t *testing.T) {
	h := newHarness(t)

	for i := 1; i <= 5; i++ {
		if rec := h.createPost(t, adminID, "Shared"); rec.Code != http.StatusCreated {
			t.Fatalf("create %d: status = %d", i, rec.Code)
		}
		if rec := h.updatePost(t, adminID, routerUpdateBody); rec.Code != http.StatusOK {
			t.Fatalf("update %d: status = %d", i, rec.Code)
		}
	}
	if rec := h.updatePost(t, adminID, routerUpdateBody); rec.Code != http.StatusTooManyRequests {
		t.Errorf("11th write: status = %d, want 429", rec.Code)
	}
}

// The limiter sits after auth on the route, so rejected callers cannot burn the admin's budget.
func TestAdminPostRoutes_DeniedUpdatesDoNotConsumeRateLimit(t *testing.T) {
	h := newHarness(t)

	for i := range 30 {
		if rec := h.updatePost(t, anonymous, routerUpdateBody); rec.Code != http.StatusUnauthorized {
			t.Fatalf("anonymous request %d: status = %d, want 401", i, rec.Code)
		}
		if rec := h.updatePost(t, editorID, routerUpdateBody); rec.Code != http.StatusForbidden {
			t.Fatalf("editor request %d: status = %d, want 403", i, rec.Code)
		}
	}
	for i := 1; i <= 10; i++ {
		if rec := h.updatePost(t, adminID, routerUpdateBody); rec.Code != http.StatusOK {
			t.Fatalf("admin request %d: status = %d, want 200 (budget was consumed by denied callers)", i, rec.Code)
		}
	}
}

func TestAdminPostRoutes_UpdateAuditActorComesFromTheSession(t *testing.T) {
	logs := captureLogs(t)
	h := newHarness(t)

	// Denied edits leave no audit trail entry.
	h.updatePost(t, anonymous, routerUpdateBody)
	h.updatePost(t, editorID, routerUpdateBody)
	h.do(t, call{method: http.MethodPut, path: "/api/v1/admin/posts/1", actor: adminID, body: routerUpdateBody}) // no CSRF
	if got := len(logs.auditRecords(t)); got != 0 {
		t.Fatalf("denied requests produced %d audit records, want 0", got)
	}

	rec := h.updatePost(t, adminID, `{"title":"Going Live","content":"c","status":"published"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d\nbody: %s", rec.Code, rec.Body.String())
	}

	records := logs.auditRecords(t)
	if len(records) != 1 {
		t.Fatalf("got %d audit records, want 1: %v", len(records), records)
	}
	got := records[0]
	if got["audit"] != true || got["action"] != "post.publish" {
		t.Errorf("record = %v", got)
	}
	if got["actor_user_id"] != float64(adminID) {
		t.Errorf("actor_user_id = %v, want %d", got["actor_user_id"], adminID)
	}
	if got["post_id"] != float64(1) || got["slug"] != "seeded-post-title" {
		t.Errorf("post_id = %v, slug = %v", got["post_id"], got["slug"])
	}
}

// ============================================================
// DELETE /admin/posts/:id
// ============================================================

func TestAdminPostRoutes_DeleteDeniedCallersChangeNothing(t *testing.T) {
	for _, a := range []struct {
		name string
		id   int64
		want int
	}{
		{"anonymous", anonymous, http.StatusUnauthorized},
		{"ghost session", ghostID, http.StatusUnauthorized},
		{"user", userID, http.StatusForbidden},
		{"editor", editorID, http.StatusForbidden},
		{"no role", noRoleID, http.StatusForbidden},
	} {
		t.Run(a.name, func(t *testing.T) {
			h := newHarness(t)

			rec := h.deletePost(t, a.id)

			if rec.Code != a.want {
				t.Fatalf("status = %d, want %d\nbody: %s", rec.Code, a.want, rec.Body.String())
			}
			if !h.seededPresent(t) || h.posts.IsSoftDeleted(1) {
				t.Error("a denied delete must not delete the post")
			}
		})
	}

	t.Run("admin does delete it, proving the route is really mounted", func(t *testing.T) {
		h := newHarness(t)

		rec := h.deletePost(t, adminID)

		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204\nbody: %s", rec.Code, rec.Body.String())
		}
		if rec.Body.Len() != 0 {
			t.Errorf("body = %q, want empty", rec.Body.String())
		}
		requireSeededSoftDeleted(t, h)
	})
}

func TestAdminPostRoutes_DeleteCSRF(t *testing.T) {
	tests := []struct {
		name string
		call call
	}{
		{"admin, header missing", call{actor: adminID, csrf: ""}},
		{"admin, header does not match the cookie", call{actor: adminID, csrf: "another-token"}},
		{"anonymous, header missing (CSRF is checked before auth)", call{actor: anonymous, csrf: ""}},
		{"editor, header missing", call{actor: editorID, csrf: ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newHarness(t)
			c := tt.call
			c.method, c.path = http.MethodDelete, "/api/v1/admin/posts/1"

			rec := h.do(t, c)

			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403\nbody: %s", rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), "CSRF") {
				t.Errorf("want a CSRF rejection, got: %s", rec.Body.String())
			}
			if !h.seededPresent(t) || h.posts.IsSoftDeleted(1) {
				t.Error("nothing may be deleted without a valid CSRF token")
			}
		})
	}

	t.Run("admin with a matching token succeeds", func(t *testing.T) {
		h := newHarness(t)
		if rec := h.deletePost(t, adminID); rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204\nbody: %s", rec.Code, rec.Body.String())
		}
		requireSeededSoftDeleted(t, h)
	})
}

// seedMore stores n extra posts straight into the mock (ids 2..n+1), bypassing the write limiter.
func (h *harness) seedMore(t *testing.T, n int) {
	t.Helper()
	for i := range n {
		if _, err := h.posts.Create(context.Background(), model.Post{
			Title: "Extra", Slug: "extra-" + strconv.Itoa(i), Content: "c", Status: model.PostStatusDraft,
		}); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
}

func TestAdminPostRoutes_RateLimitOnDelete(t *testing.T) {
	h := newHarness(t)
	h.seedMore(t, 11) // ids 2..12

	// Burst is 10: the first ten deletes pass, the eleventh is rejected and deletes nothing.
	for id := int64(1); id <= 10; id++ {
		if rec := h.deletePostID(t, adminID, id); rec.Code != http.StatusNoContent {
			t.Fatalf("delete %d: status = %d, want 204\nbody: %s", id, rec.Code, rec.Body.String())
		}
	}
	rec := h.deletePostID(t, adminID, 11)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("delete 11: status = %d, want 429\nbody: %s", rec.Code, rec.Body.String())
	}
	if got := errorCode(t, rec); got != "RATE_LIMIT_EXCEEDED" {
		t.Errorf("error code = %q", got)
	}
	if h.posts.IsSoftDeleted(11) {
		t.Error("the throttled request must not delete anything")
	}

	// Reads are not limited by the write limiter.
	if read := h.do(t, call{method: http.MethodGet, path: "/api/v1/admin/posts/11", actor: adminID}); read.Code != http.StatusOK {
		t.Errorf("GET after the burst: status = %d, want 200", read.Code)
	}
}

// Deletes, creates and edits draw from the same per-client write budget (one postWriteLimit).
func TestAdminPostRoutes_DeletesShareTheWriteBudgetWithCreatesAndUpdates(t *testing.T) {
	h := newHarness(t)
	h.seedMore(t, 5) // ids 2..6

	for i := 1; i <= 4; i++ {
		if rec := h.createPost(t, adminID, "Shared"); rec.Code != http.StatusCreated {
			t.Fatalf("create %d: status = %d", i, rec.Code)
		}
	}
	for i := 1; i <= 3; i++ {
		if rec := h.updatePost(t, adminID, routerUpdateBody); rec.Code != http.StatusOK {
			t.Fatalf("update %d: status = %d", i, rec.Code)
		}
	}
	for id := int64(2); id <= 4; id++ { // 3 deletes: 4 + 3 + 3 = 10 writes
		if rec := h.deletePostID(t, adminID, id); rec.Code != http.StatusNoContent {
			t.Fatalf("delete %d: status = %d", id, rec.Code)
		}
	}
	if rec := h.deletePostID(t, adminID, 5); rec.Code != http.StatusTooManyRequests {
		t.Errorf("11th write (a delete): status = %d, want 429", rec.Code)
	}
	if h.posts.IsSoftDeleted(5) {
		t.Error("the throttled delete must not be applied")
	}
}

// The limiter sits after auth on the route, so rejected callers cannot burn the admin's budget.
func TestAdminPostRoutes_DeniedDeletesDoNotConsumeRateLimit(t *testing.T) {
	h := newHarness(t)
	h.seedMore(t, 9) // ids 2..10

	for i := range 30 {
		if rec := h.deletePost(t, anonymous); rec.Code != http.StatusUnauthorized {
			t.Fatalf("anonymous request %d: status = %d, want 401", i, rec.Code)
		}
		if rec := h.deletePost(t, editorID); rec.Code != http.StatusForbidden {
			t.Fatalf("editor request %d: status = %d, want 403", i, rec.Code)
		}
	}
	for id := int64(1); id <= 10; id++ {
		if rec := h.deletePostID(t, adminID, id); rec.Code != http.StatusNoContent {
			t.Fatalf("admin delete %d: status = %d, want 204 (budget was consumed by denied callers)", id, rec.Code)
		}
	}
}

func TestAdminPostRoutes_DeleteAuditActorComesFromTheSession(t *testing.T) {
	logs := captureLogs(t)
	h := newHarness(t)

	// Denied deletes leave no audit trail entry.
	h.deletePost(t, anonymous)
	h.deletePost(t, editorID)
	h.do(t, call{method: http.MethodDelete, path: "/api/v1/admin/posts/1", actor: adminID}) // no CSRF
	if got := len(logs.auditRecords(t)); got != 0 {
		t.Fatalf("denied requests produced %d audit records, want 0", got)
	}

	if rec := h.deletePost(t, adminID); rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d\nbody: %s", rec.Code, rec.Body.String())
	}

	records := logs.auditRecords(t)
	if len(records) != 1 {
		t.Fatalf("got %d audit records, want 1: %v", len(records), records)
	}
	got := records[0]
	if got["audit"] != true || got["action"] != "post.delete" {
		t.Errorf("record = %v", got)
	}
	if got["actor_user_id"] != float64(adminID) {
		t.Errorf("actor_user_id = %v, want %d", got["actor_user_id"], adminID)
	}
	if got["post_id"] != float64(1) || got["slug"] != "seeded-post-title" {
		t.Errorf("post_id = %v, slug = %v", got["post_id"], got["slug"])
	}
}

func TestAdminPostRoutes_DeletedPostDisappearsFromPublicAndAdminReads(t *testing.T) {
	h := newHarness(t)
	// Publish the seeded post so a public read can see it before the delete.
	if rec := h.updatePost(t, adminID, `{"title":"Live","content":"c","status":"published"}`); rec.Code != http.StatusOK {
		t.Fatalf("publish: status = %d", rec.Code)
	}
	if rec := h.do(t, call{method: http.MethodGet, path: "/api/v1/posts/seeded-post-title"}); rec.Code != http.StatusOK {
		t.Fatalf("public read before delete: status = %d, want 200", rec.Code)
	}

	if rec := h.deletePost(t, adminID); rec.Code != http.StatusNoContent {
		t.Fatalf("delete: status = %d", rec.Code)
	}

	if rec := h.do(t, call{method: http.MethodGet, path: "/api/v1/posts/seeded-post-title"}); rec.Code != http.StatusNotFound {
		t.Errorf("public read after delete: status = %d, want 404", rec.Code)
	}
	if rec := h.do(t, call{method: http.MethodGet, path: "/api/v1/admin/posts/1", actor: adminID}); rec.Code != http.StatusNotFound {
		t.Errorf("admin read after delete: status = %d, want 404", rec.Code)
	}
	if rec := h.deletePost(t, adminID); rec.Code != http.StatusNotFound {
		t.Errorf("second delete: status = %d, want 404", rec.Code)
	}
}

// ============================================================
// Public post routes
// ============================================================

func TestPublicPostRoutes_NeedNoAuthAndHideDrafts(t *testing.T) {
	h := newHarness(t) // post 1 is a draft

	list := h.do(t, call{method: http.MethodGet, path: "/api/v1/posts", actor: anonymous})
	if list.Code != http.StatusOK {
		t.Fatalf("anonymous GET /posts = %d, want 200\nbody: %s", list.Code, list.Body.String())
	}
	if !strings.Contains(list.Body.String(), `"data":[]`) {
		t.Errorf("draft leaked into the public list: %s", list.Body.String())
	}

	draft := h.do(t, call{method: http.MethodGet, path: "/api/v1/posts/seeded-post-title", actor: anonymous})
	missing := h.do(t, call{method: http.MethodGet, path: "/api/v1/posts/no-such-post", actor: anonymous})
	if draft.Code != http.StatusNotFound || draft.Body.String() != missing.Body.String() {
		t.Errorf("draft = %d %q, missing = %d %q, want identical 404s", draft.Code, draft.Body.String(), missing.Code, missing.Body.String())
	}

	// A logged-in non-admin sees exactly the same thing.
	asUser := h.do(t, call{method: http.MethodGet, path: "/api/v1/posts/seeded-post-title", actor: userID})
	if asUser.Code != http.StatusNotFound || asUser.Body.String() != missing.Body.String() {
		t.Errorf("as user = %d %q, want the same 404", asUser.Code, asUser.Body.String())
	}
}
