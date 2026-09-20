# Tasks: Blog Post CRUD

**Plan:** ./plan.md
**Date:** 2026-09-20

---

## DAG

```
T1 ──┐
     ├─> T3 ──┬─> T4 ──┐
T2 ──┘        ├─> T5 ──┤
              ├─> T6 ──┼─> T8 ─> T9
              └─> T7 ──┘
```

| Ticket | Type | blocked_by | Parallel with | RBAC-sensitive |
|---|---|---|---|---|
| T1 | Migration | none | T2 | no |
| T2 | Service (pure logic) + model | none | T1 | no |
| T3 | Repository + Service + Handler + RBAC | T1, T2 | none | **yes** |
| T4 | Repository + Service + Handler + RBAC | T3 | T5, T6, T7 | **yes** |
| T5 | Repository + Service + Handler + RBAC | T3 | T4, T6, T7 | **yes** |
| T6 | Repository + Service + Handler | T3 | T4, T5, T7 | no (admin-only reads, covered by matrix) |
| T7 | Repository + Service + Handler | T3 | T4, T5, T6 | no (public) |
| T8 | API Change | T4, T5, T6, T7 | none | no |
| T9 | RBAC (gate) | T8 | none | **yes** |

T4-T7 can run in parallel in principle but share `post_repo.go`, `post_service.go`, `router.go`
and `mock_repo.go`. Use separate worktrees and rebase per merge, or run serially (T7, T6, T4, T5).

**Every ticket:** fresh context; write failing tests first; before claiming done run `make lint`
(zero warnings) and `make test`; run `go test -tags=integration ./internal/repository/ ./internal/service/`
if either package changed; run `make swagger` if any handler changed (no uncommitted diff after
commit); update checklist.md and notes.md in this spec directory (create from `.ai-workflow/templates/`
on first use).

**RBAC-sensitive tickets (T3, T4, T5, T9)** need an extra reviewer and this checklist resolved in the
PR description:

- [ ] Session renewed on privilege change: N/A, no role or privilege changes here. Reviewer confirms.
- [ ] CSRF token validated: global `CSRFProtection` applies; test proves POST/PUT/DELETE without token are rejected.
- [ ] Rate limit applied: `RateLimit` on the write route; test proves 429 past burst.
- [ ] Audit log entry: `slog` event with `audit=true`, `actor_user_id`, `action`, `post_id`, `slug` on each successful write; test asserts it.

---

## T1: Create posts table

**Type:** Migration
**blocked_by:** none
**Parallel:** T2

**Pre-flight (do first):** verify ports per plan.md "Pre-flight: Database Ports" (`.env` says 5437/5438, not AGENTS.md's 5435/5436). Confirm `make migrate-status` connects to `ssrblog_db` before running `up`.

**Files:**
- `internal/database/migrations/20260920100000_create_posts.sql`

**Test:**
- `make migrate-up && make migrate-down && make migrate-up` succeeds against the dev DB.
- `\d posts` shows: `idx_posts_slug` UNIQUE (not partial), `idx_posts_public_list` and `idx_posts_admin_list` (both partial on `deleted_at IS NULL`), CHECK on `status`, `deleted_at` nullable.
- Inserting a duplicate slug fails; inserting `status='archived'` fails.
- Down drops table and indexes cleanly.

**Commit:** `feat(db): create posts table with slug unique index and soft delete`
**Done when:** up/down/up clean; schema matches plan.md; no other migration edited.

## T2: Post model, slug generator, `notblank` validator, metrics

**Type:** Service (pure business logic) + model
**blocked_by:** none
**Parallel:** T1

**Files:**
- `internal/model/post.go` (Post, PostSummary, AdminPostSummary, PostResponse, Create/Update request structs with validation tags, status constants)
- `internal/service/slug.go`, `internal/service/slug_test.go`
- `internal/validator/validator.go` (+ test): register `notblank`; confirm `http_url` is available in the pinned validator version, otherwise add a custom `httpurl` tag
- `pkg/metrics/metrics.go`: `blog_post_writes_total{operation}`, `blog_post_slug_attempts` histogram

**Test (table-driven, written first):**
- `"Café Au Lait"` -> `cafe-au-lait`; `"Hello,   World!!"` -> `hello-world`; `"C++ & Go"` -> `c-go`; `"2026 Roadmap"` -> `2026-roadmap`; `"  --A--B "` -> `a-b`.
- `"!!!"`, Japanese, Arabic -> empty.
- 300-char title -> base <= 190 chars, no trailing `-`; `WithSuffix(base, 2)` <= 200.
- `notblank` rejects `""`, `"   "`, `"\n\t"`; accepts `" a "`.
- URL rules: `https://x.io/a.png` ok; `javascript:alert(1)`, `ftp://x`, `not a url` rejected; empty allowed.
- Length limits count characters (a 200-rune title of multibyte characters passes, 201 fails).
- Model has no behavior (Constitution 1.1); metrics labels are bounded enums.

**Commit:** `feat(post): add post model, slug generator, notblank validator, metrics`
**Done when:** `make lint` and `make test` pass; no new go.mod entries (`git diff go.mod` empty).

## T3: Admin creates a post and views it by ID

**Type:** Repository + Service + Handler + RBAC (sensitive; extra reviewer)
**blocked_by:** T1, T2
**Parallel:** none (walking skeleton for T4-T7)

**Files:**
- `internal/repository/interfaces.go`: `PostRepository` (Create, GetByID, and the rest as later tickets add them)
- `internal/repository/post_repo.go`: `Create`, `GetByID`, `notDeleted()` helper, `ErrSlugTaken` (SQLSTATE 23505 on `idx_posts_slug` via `pgconn.PgError`, part of pgx v5, no new dependency), `ErrPostNotFound`; `sq.Dollar`; explicit `pgx.ErrNoRows` -> `nil, nil`
- `internal/repository/mock_repo.go`: `MockPostRepo` with override funcs
- `internal/repository/post_repo_integration_test.go`
- `internal/service/post_service.go`: `Create` (slugify -> empty => `apperror.Validation`; retry loop with cap 100; `published_at = now` when created as `published`; metrics), `GetByID`
- `internal/service/post_service_test.go`, `internal/service/post_service_integration_test.go` (tagged)
- `internal/handler/admin_post_handler.go` (+ test): `POST /admin/posts` -> 201 `PostResponse`; `GET /admin/posts/:id` -> 200; swagger annotations; audit `slog` event on create
- `internal/router/router.go` (+ `router_post_test.go`): mount on the existing `admin` group; `RateLimit` on POST; builds the RBAC matrix harness that T4-T6 append rows to
- `cmd/api/main.go`: wire `postRepo` -> `postService` -> handlers

**Test:**
- Unit: create defaults to draft; published-at set only when created published; collision at `base`, `base`+`-2` -> gets `-3`; cap reached -> internal error; empty slug -> 422 and repo never called; repo error -> `apperror.Internal`, no raw error leaks.
- Handler: each field limit (200/100000/500/2048 pass, +1 fails), blank title/content, bad URL scheme, bad status, malformed JSON -> 400, non-numeric id -> 400, unknown id -> 404, deleted id -> 404.
- Integration: create/get round trip; duplicate slug -> `ErrSlugTaken`; 20 concurrent inserts of one slug -> exactly one success; 20 concurrent service creates of one title -> 20 successes, 20 distinct slugs, zero errors; deleted row (set directly in SQL) not returned by `GetByID`.
- RBAC matrix rows for `POST /admin/posts` and `GET /admin/posts/:id`: anonymous 401, `user` 403, `editor` 403, `admin` OK; missing CSRF rejected; 429 after burst; DB row count unchanged after every denial.

**Commit:** `feat(post): admin can create and view posts with unique slugs`
**Done when:** security checklist above resolved; all gates green; `make swagger` regenerated and committed; manual: admin POST then GET returns the post, editor POST returns 403.

## T4: Admin edits a post (publish transitions, immutable slug)

**Type:** Repository + Service + Handler + RBAC (sensitive; extra reviewer)
**blocked_by:** T3
**Parallel:** T5, T6, T7

**Files:**
- `internal/repository/post_repo.go`: `Update(ctx, id, fields)` sets title, content, excerpt, cover_image_url, status, published_at, updated_at; never `slug`; uses `notDeleted()`; 0 rows -> `ErrPostNotFound`
- `internal/repository/mock_repo.go`, `interfaces.go`
- `internal/service/post_service.go`: `Update` loads current post, applies `published_at` rules, calls repo; metrics `update`/`publish`/`unpublish`
- `internal/handler/admin_post_handler.go`: `PUT /admin/posts/:id` -> 200; audit event; swagger
- `internal/router/router.go`: route + `RateLimit`; add matrix row

**Test:**
- Unit `published_at` matrix: draft->published = clock now; published->published unchanged (even when other fields edited); published->draft = nil; draft->published again = new now (not the old value).
- Unit: unknown/deleted id -> 404; invalid payload -> 422 and repo not called; body containing `"slug":"hacked"` leaves slug unchanged.
- Integration: update persists all editable fields; `slug` column identical before and after a title change; updating a soft-deleted row -> `ErrPostNotFound`; `updated_at` advances.
- Handler validation + error-path tests as in T3.
- RBAC matrix row for `PUT /admin/posts/:id` incl. CSRF and rate limit.

**Commit:** `feat(post): admin can edit posts with stable slug and published date rules`
**Done when:** security checklist resolved; gates green; swagger regenerated.

## T5: Admin soft-deletes a post

**Type:** Repository + Service + Handler + RBAC (sensitive; extra reviewer)
**blocked_by:** T3
**Parallel:** T4, T6, T7

**Files:**
- `internal/repository/post_repo.go`: `SoftDelete(ctx, id)` = `UPDATE posts SET deleted_at = NOW() WHERE id = ? AND deleted_at IS NULL`; 0 rows -> `ErrPostNotFound`. No hard delete; do not use the generic `deleteByID` helper.
- `internal/repository/mock_repo.go`, `interfaces.go`
- `internal/service/post_service.go`: `Delete`; metrics `delete`
- `internal/handler/admin_post_handler.go`: `DELETE /admin/posts/:id` -> 204; audit event; swagger
- `internal/router/router.go`: route + `RateLimit`; add matrix row

**Test:**
- Integration: row still present with `deleted_at` set (never removed); second delete -> `ErrPostNotFound`; deleted post gone from `GetByID`; slug stays reserved: create same title afterward -> `-2` (through the real service); deleted published post absent from the public list count.
- Unit: unknown id -> 404; repo error -> 500 via apperror.
- Handler: bad id -> 400; not found -> 404; success -> 204 with empty body.
- RBAC matrix row for `DELETE /admin/posts/:id`; DB row unchanged after every denial.

**Commit:** `feat(post): admin can soft-delete posts and keep slugs reserved`
**Done when:** security checklist resolved; gates green; swagger regenerated.

## T6: Admin lists posts with status filter

**Type:** Repository + Service + Handler (route is behind the admin RBAC group; matrix row required)
**blocked_by:** T3
**Parallel:** T4, T5, T7

**Files:**
- `internal/repository/post_repo.go`: `ListAdmin(ctx, status, page, limit)` returns `[]AdminPostSummary` plus total, ordered `created_at DESC, id DESC`, `notDeleted()`, optional status filter
- `internal/repository/mock_repo.go`, `interfaces.go`
- `internal/service/post_service.go`: `ListAdmin`; reject unknown status filter with `apperror.Validation`
- `internal/handler/admin_post_handler.go`: `GET /admin/posts?status=&page=&limit=` using `pagination.FromContext` and `pagination.NewResponse`; swagger
- `internal/router/router.go`: route; add matrix row

**Test:**
- Integration: drafts and published both listed; `status=draft` and `status=published` filter correctly; soft-deleted excluded from data and from `meta.total`; page beyond last -> empty `data: []` with valid meta; ordering stable with equal `created_at`.
- Handler: `status=bogus` -> 422; `limit=0` and `limit=1000` clamp like other endpoints (assert same behavior as `/admin/roles`); response has no `content` field.
- RBAC matrix row for `GET /admin/posts`.

**Commit:** `feat(post): admin can list posts filtered by status`
**Done when:** gates green; swagger regenerated; pagination format matches `pkg/pagination` exactly.

## T7: Public list and read by slug

**Type:** Repository + Service + Handler
**blocked_by:** T3
**Parallel:** T4, T5, T6

**Files:**
- `internal/repository/post_repo.go`: `ListPublished(ctx, page, limit)` returns `[]PostSummary` plus total, `status='published' AND deleted_at IS NULL`, `published_at DESC, id DESC`; `GetPublishedBySlug(ctx, slug)` with the same predicate and explicit `pgx.ErrNoRows` -> `nil, nil`
- `internal/repository/mock_repo.go`, `interfaces.go`
- `internal/service/post_service.go`: `ListPublished`, `GetPublishedBySlug` (nil -> `apperror.NotFound("post not found")`, one message for every miss)
- `internal/handler/post_handler.go`: `GET /posts` and `GET /posts/:slug`; unauthenticated; swagger
- `internal/router/router.go`: mount under `api` with no auth middleware

**Test:**
- Integration: only published rows appear; drafts, soft-deleted and unpublished-then-draft rows excluded from list and count; newest `published_at` first with `id` tie-break; slug lookup of a draft, a deleted post and a missing slug all return `nil`.
- Handler: draft slug, deleted slug and nonexistent slug return byte-identical 404 bodies; list omits `content`; single read includes `content`, excludes `deleted_at`; empty list -> `{"data":[],"meta":{...}}` with 200; page past the end -> empty data; works with no session cookie; ignores any auth header.
- Response of `GET /posts/:slug` for a published post contains every spec field.

**Commit:** `feat(post): public list and read of published posts by slug`
**Done when:** gates green; swagger regenerated; no auth middleware on these routes (asserted by a router test that sends no cookie).

## T8: Swagger and API docs

**Type:** API Change
**blocked_by:** T4, T5, T6, T7
**Parallel:** none

**Files:**
- `docs/swagger.json`, `docs/swagger.yaml`, `docs/docs.go` (via `make swagger`)
- `docs/API.md`: new "Posts" (public) and "Admin: Posts" sections with request/response examples for all 7 endpoints, using the real error shape `{"error": "...", "message": "..."}` from `internal/apperror/response.go`. State: slug is server-generated and immutable; `PUT` needs title, content and status; `content` is raw markdown and may contain HTML, so clients must render it safely; pagination is `data` + `meta`.

**Test:**
- `make swagger` produces no diff after commit; every new route appears in `swagger.yaml` with tags, security (`CookieAuth` on admin), request bodies and 401/403/404/422 responses.
- Every example in API.md is copy-run against a local server and the output matches.
- Pagination section matches the existing endpoints' format.

**Commit:** `docs(api): document blog post endpoints and regenerate swagger`
**Done when:** swagger diff empty on re-run; API.md examples verified.

## T9: RBAC gate and end-to-end smoke

**Type:** RBAC (sensitive; extra reviewer)
**blocked_by:** T8
**Parallel:** none

**Files:**
- `internal/router/router_post_test.go` (complete the matrix if any row is missing)
- `.specify/specs/001-blog-post-crud/checklist.md`, `notes.md`

**Test:**
- Full matrix: 5 admin routes x {anonymous 401, `user` 403, `editor` 403, `admin` success}; 2 public routes reachable anonymously; CSRF rejection on every write route; 429 on every write route; DB state unchanged after each denial.
- Manual smoke against the running app (ports checked per plan pre-flight): register -> verify -> login as admin -> create draft -> public read 404 -> publish -> public list and read 200 -> edit title, slug unchanged -> unpublish, public 404 and `published_at` cleared -> republish, new `published_at` -> delete, public 404 -> create same title, gets `-2` -> logout.
- `/metrics` shows `blog_post_writes_total{operation=...}`, `blog_post_slug_attempts` and route-templated `http_requests_total` for the new paths, with no slug, id or user labels.
- Logs show an `audit=true` event for each admin write and none for denied requests.
- Walk the spec's Success Criteria and Edge Cases lists; tick each in checklist.md with the test that proves it.

**Commit:** `test(post): complete RBAC matrix and record verification`
**Done when:** every Success Criteria box in spec.md is checked with evidence; `make lint`, `make test`, integration tests, `make swagger` all clean; second reviewer signs the security checklist. Then run `/speckit-converge`.
