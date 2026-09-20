# Tasks: Blog Post CRUD

**Plan:** ./plan.md
**Date:** 2026-09-20

---

## Ticket Graph

- T1 (Migration) and T2 (Foundation) run in parallel.
- T1 + T2 -> T3a (Repository) -> T3 b (Service) -> T3c (Handler + RBAC).
- T3 c -> T4, T5, T6, T7 (can run in parallel; share files, so serial is safer).
- T4 + T5 + T6 + T7 -> T8 (Swagger + API docs).
- T8 -> T9 (RBAC gate + smoke).

| Ticket | Type | blocked_by | Parallel with | RBAC-sensitive |
|---|---|---|---|---|
| T1 | Migration | none | T2 | no |
| T2 | Foundation (mixed) | none | T1 | no |
| T3a | Repository | T1, T2 | none | no |
| T3b | Service | T3a | none | no |
| T3c | Handler + RBAC | T3b | none | yes |
| T4 | Repository + Service + Handler + RBAC | T3c | T5, T6, T7 | yes |
| T5 | Repository + Service + Handler + RBAC | T3c | T4, T6, T7 | yes |
| T6 | Repository + Service + Handler | T3c | T4, T5, T7 | no |
| T7 | Repository + Service + Handler | T3c | T4, T5, T6 | no |
| T8 | API Change | T3c, T4, T5, T6, T7 | none | no |
| T9 | RBAC (verification gate) | T8 | none | yes |

T4-T7 can run in parallel in principle but share `post_repo.go`, `post_service.go`, `router.go`, and `mock_repo.go`. Use separate worktrees and rebase per merge, or run serially (T7, T6, T4, T5).

**Every ticket:** fresh context; write failing tests first; before claiming done run `make lint` (zero warnings) and `make test`; run `go test -tags=integration ./internal/repository/ ./internal/service/` if either package changed; run `make swagger` if any handler changed; update checklist.md and notes.md in this spec directory.

**RBAC-sensitive tickets (T3c, T4, T5, T9)** need an extra reviewer and this checklist resolved in the PR description:

- [ ] Session renewed on privilege change: N/A, no role or privilege changes here. Reviewer confirms.
- [ ] CSRF token validated: global CSRFProtection applies; test proves POST/PUT/DELETE without token are rejected.
- [ ] Rate limit applied: RateLimit on the write route; test proves 429 past burst.
- [ ] Audit log entry: slog event with audit=true, actor_user_id, action, post_id, slug on each successful write; test asserts it.

---


## T1: Create posts table

**Type:** Migration
**blocked_by:** none
**Parallel:** T2

**Pre-flight (do first):** verify ports per plan.md "Pre-flight: Database Ports" (`.env` says 5437/5438, not AGENTS.md's 5435/5436). Confirm `make migrate-status` connects to `ssrblog_db` before running `up`.

**Files:**
- -internal/database/migrations/20260920100000_create_posts.sql

**Test:**
- .make migrate-up && make migrate-down && make migrate-up` succeeds against the dev DB.
- `\d posts` shows: `idx_posts_slug` UNIQUE, `idx_posts_public_list` and `idx_posts_admin_list` (both partial on `deleted_at IS NULL@), CHECK on `status`, `deleted_at` nullable.
- .Inserting a duplicate slug fails; inserting `status='archived'` fails.
- .Down drops table and indexes cleanly.

**Commit:** `feat(db): create posts table with slug unique index and soft delete`
**Done when:** up/down/up clean; schema matches plan.md; no other migration edited.

## T2: Post model, slug generator, notblank validator, metrics

**Type:** Foundation (mixed: model + service + validator + metrics).
No single AGENTS.md ticket type applies. PR description must note this.

**blocked_by:** none
**Parallel:** T1

**Files:*
- `internal/model/post.go` (Post, PostSummary, AdminPostSummary, PostResponse, Create/Update request structs with validation tags, status constants)
- -internal/service/slug.go`, `internal/service/slug_test.go`
- `internal/validator/validator.go` (+ test): register `notblank`; confirm `http_url` is available in the pinned validator version, otherwise add a custom `httpurl` tag
- .pkg/metrics/metrics.go`: `blog_post_writes_total{operation}`, `blog_post_slug_attempts` histogram

**Test (table-driven, written first):**
- `Cafe Au Lait` -> `cafe-au-lait`; `Hello,   World!!` -> `hello-world`; `C++ & Go` -> `c-go`; `2026 Roadmap` -> `2026-roadmap`; `  --A--B  ` -> `a-b`.
- .!!!`, Japanese, Arabic -> empty.
- 300-char title -> base <= 190 chars, no trailing `-`; `WithSuffix(base, 2)` <= 200.
- `notblank` rejects `""`, `"   "`, `"\n\t`; accepts `" a "`.
- URL rules: `https://x.io/a.png` ok; `javascript:alert(1)`, `ftp://x`, `not a url` rejected; empty allowed.
- Length limits count characters (a 200-rune title of multibyte characters passes, 201 fails).
- Model has no behavior (Constitution 1.1); metrics labels are bounded enums.

**Commit:** `feat(post): add post model, slug generator, notblank validator, metrics`
**Done when:** `make lint` and `make test` pass; no new go.mod entries (`git diff go.mod` empty).

## T3a: Post repository foundation

**Type:** Repository
**blocked_by:** T1, T2
**Parallel:** none

**Files:*
- -internal/repository/interfaces.go`: `PostRepository` interface (Create, GetByID, stubs for later)
- `internal/repository/post_repo.go`: `Create`, `GetByID`, `notDeleted()` helper, `ErrSlugTaken` (SQLSTATE 23505 on `idx_posts_slug` via `pgconn.PgError`), `ErrPostNotFound`; `sq.Dollar`; `pgx.ErrNoRows` -> `nil, nil`
- `internal/repository/mock_repo.go`: `MockPostRepo` with override funcs
- `internal/repository/post_repo_integration_test.go`

**Test:**
- Integration: create/get round trip; duplicate slug -> `ErrSlugTaken`; 20 concurrent inserts of one slug -> exactly one success; deleted row not returned by `GetByID`.

**Commit:** `feat(repo): add post repository create and get-by-id`
**Done when:** `go test -tags=integration ./internal/repository/` passes; no handler/service changes.

## T3b: Post service create and get

**Type:** Service
**blocked_by:** T3a
**Parallel:** none

**Files:**
- -internal/service/post_service.go`: `Create` (slugify -> empty => `apperror.Validation`; retry loop cap 100; `published_at = now` when created as `published`; metrics), `GetByID`
- `internal/service/post_service_test.go`
- `internal/service/post_service_integration_test.go` (tagged)

**Test:**
- .Unit: create defaults to draft; published-at set only when created published; collision at `base`, `base`+`-2` -> gets `-3`; cap reached -> internal error; empty slug -> 422 and repo never called; repo error -> `apperror.Internal`, no raw error leaks.
- .Integration: 20 concurrent service creates of one title -> 20 successes, 20 distinct slugs, zero errors.

**Commit:** `feat(service): add post create and get with slug retry`
**Done when:** `make lint` and `make test` pass; integration passes.

## T3c: Admin create + get endpoints, RBAC harness

**Type:** Handler + RBAC (sensitive; extra reviewer)
**blocked_by:** T3b
**Parallel:** none (walking skeleton for T4-T7)

**Files:*
- -internal/handler/admin_post_handler.go` (+ test): `POST /admin/posts` -> 201 `PostResponse`; `GET /admin/posts/:id` -> 200; swagger annotations; audit `slog` event on create
- -internal/router/router.go` (+ `router_post_test.go`): mount on the existing `admin` group; `RateLimit` on POST; builds the RBAC matrix harness
- `cmd/api/main.go`: wire `postRepo` -> `postService` -> handlers

**Test:**
- Handler: each field limit (200/100000/500/2048 pass, +1 fails), blank title/content, bad URL scheme, bad status, malformed JSON -> 400, non-numeric id -> 400, unknown id -> 404, deleted id -> 404.
- RBAC matrix rows for `POST /admin/posts` and `GET /admin/posts/:id`: anonymous 401, `user` 403, `editor` 403, `admin` OK; missing CSRF rejected; 429 after burst; DB row count unchanged after every denial.

**Commit:** `feat(handler): admin can create and view posts with RBAC`
**Done when:** security checklist above resolved; all gates green; `make swagger` regenerated and committed; manual: admin POST then GET returns the post, editor POST returns 403.
