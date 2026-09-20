# Plan: Blog Post CRUD

**Spec:** ./spec.md (+ ./clarifications.md)
**Date:** 2026-09-20

---

## Approach

Add one table (`posts`) and one vertical feature under the existing layers. Nothing existing is
modified except wiring (`router.go`, `main.go`), seed-free. All admin routes mount on the existing
`/api/v1/admin` group, which already applies `RequireAuth` + `RequireRole("admin")`. That gives
401 for anonymous and 403 for non-admin (including `editor`) with no new auth code.

Public routes: `GET /api/v1/posts`, `GET /api/v1/posts/:slug`.
Admin routes: `GET|POST /api/v1/admin/posts`, `GET|PUT|DELETE /api/v1/admin/posts/:id`.

Update is `PUT` (full replace of the editable fields), matching `PUT /admin/roles/:id`.

## Design Decisions

| Topic | Decision |
|---|---|
| Slug uniqueness | Full `UNIQUE` index on `slug` (not partial), so soft-deleted slugs stay reserved. Repo maps SQLSTATE `23505` on that index to `repository.ErrSlugTaken`; service never sees pgx types. |
| Slug retry | Service loop: `base`, `base-2`, `base-3`, ... on `ErrSlugTaken`. Cap 100 attempts, then `apperror.Internal`. Base is truncated to 190 chars (trailing `-` trimmed) so a suffix always fits in 200. |
| Slug generation | Stdlib only. Lowercase, fold Latin diacritics via a small map (Latin-1 + Latin Extended-A), non-`[a-z0-9]` runs become one `-`, trim `-`. Empty result -> `apperror.Validation`. Lives in `internal/service/slug.go` (models have no behavior). |
| Slug immutability | `PostRepository.Update` never writes `slug`. Update request has no slug field; a `slug` key in the body is ignored. |
| Soft delete | `deleted_at TIMESTAMPTZ NULL`. Every SELECT/UPDATE/DELETE in `post_repo.go` goes through one `notDeleted()` builder helper. Integration test per method proves deleted rows are excluded. |
| `published_at` | Computed in the service (injectable clock): draft->published = now; published->published = keep; ->draft = NULL. Create with `status=published` sets now. |
| Validation | Handler `c.Validate()`: title `required,notblank,max=200`; content `required,notblank,max=100000`; excerpt `max=500`; cover URL `omitempty,max=2048,http_url`; status `required,oneof=draft published`. New `notblank` tag in `internal/validator`. `url` alone is rejected because it accepts `javascript:`. Empty-slug title is rejected in the service (422). |
| Response shapes | `PostResponse` (all fields except `deleted_at`) for single reads. `PostSummary` (title, slug, excerpt, cover_image_url, published_at) for the public list. Admin list returns `AdminPostSummary` (summary + id, status, created_at, updated_at; no content). |
| Ordering | Public: `published_at DESC, id DESC`. Admin: `created_at DESC, id DESC`. |
| Pagination | `pkg/pagination` as-is (`data` + `meta`, default 20, max 100, out-of-range params clamped). |
| Not-found parity | Public single read is one query: `slug = ? AND status = 'published' AND deleted_at IS NULL`. Draft, deleted and missing are the same code path and the same response body. |
| Metrics | Existing `PrometheusMiddleware` already records `http_requests_total` and `http_request_duration_seconds` per route template for every new endpoint. Add two domain metrics: `blog_post_writes_total{operation}` (create, update, delete, publish, unpublish) and `blog_post_slug_attempts` histogram. Labels are bounded enums; no slug/id/user labels. |
| Audit log | No audit facility exists in the repo. Emit a structured `slog` event (`audit=true`, `actor_user_id`, `action`, `post_id`, `slug`) from the admin handler after each successful write. No new table. Log at handler level per Constitution 6.2. |
| Rate limit | Existing `RateLimit` middleware on admin write routes: 30/min, burst 10 (proposal, tunable). |
| Migration filename | Follow existing files: `YYYYMMDDHHMMSS_name.sql` (AGENTS.md says `NNNN_name.sql`, but no existing migration uses it). |

## Schema

```sql
CREATE TABLE posts (
    id              BIGSERIAL PRIMARY KEY,
    title           VARCHAR(200)  NOT NULL,
    slug            VARCHAR(200)  NOT NULL,
    content         TEXT          NOT NULL,
    excerpt         VARCHAR(500)  NOT NULL DEFAULT '',
    cover_image_url VARCHAR(2048) NOT NULL DEFAULT '',
    status          VARCHAR(16)   NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','published')),
    published_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);
CREATE UNIQUE INDEX idx_posts_slug ON posts(slug);
CREATE INDEX idx_posts_public_list ON posts(published_at DESC, id DESC)
    WHERE status = 'published' AND deleted_at IS NULL;
CREATE INDEX idx_posts_admin_list ON posts(created_at DESC, id DESC)
    WHERE deleted_at IS NULL;
```

Three indexes, none redundant: uniqueness/slug lookup, public list, admin list. The admin `status`
filter is applied on the `idx_posts_admin_list` scan. Postgres `VARCHAR(n)` counts characters, which
matches the spec's "characters" limits.

## Files to Touch

| File | Change | Risk |
|---|---|---|
| internal/database/migrations/20260920100000_create_posts.sql | New table + 3 indexes | High |
| internal/model/post.go | Post, PostSummary, AdminPostSummary, requests, responses, status consts | Low |
| internal/service/slug.go (+ test) | Slugify, suffixing | Medium |
| internal/validator/validator.go (+ test) | `notblank` tag; confirm `http_url` exists in the pinned validator version | Low |
| pkg/metrics/metrics.go | 2 domain metrics | Low |
| internal/repository/interfaces.go | `PostRepository` interface | Low |
| internal/repository/post_repo.go (+ integration test) | All post SQL, `ErrSlugTaken`, `ErrPostNotFound`, `notDeleted()` | High |
| internal/repository/mock_repo.go | `MockPostRepo` (existing override-func pattern) | Low |
| internal/service/post_service.go (+ test, + integration test) | Create/Update/Delete/List/Get, retry loop, `published_at` rules | High |
| internal/handler/admin_post_handler.go (+ test) | Admin CRUD + list, audit log, swagger annotations | Medium |
| internal/handler/post_handler.go (+ test) | Public list + read, swagger annotations | Low |
| internal/router/router.go (+ test) | Mount routes, rate limit on writes | High (RBAC) |
| cmd/api/main.go | Construct and inject repo/service/handlers | Low |
| docs/swagger.{json,yaml}, docs/docs.go | Regenerated | Low |
| docs/API.md | Post endpoints with examples; note that content is unsanitized markdown | Low |

Not touched: `AGENTS.md`, `docs/ARCHITECTURE.md` (no auth flow changes), seeds, existing repos/services.

## Order (DAG)

```
T1 migration ──┐
               ├──> T3 admin create+get ──┬──> T4 admin update ──┐
T2 foundation ─┘        (RBAC)            ├──> T5 admin delete ──┤
                                          ├──> T6 admin list ────┼──> T8 API docs ──> T9 RBAC gate
                                          └──> T7 public reads ──┘
```

- T1 and T2 run in parallel.
- T4, T5, T6, T7 are independent of each other once T3 lands, but all edit `post_repo.go`,
  `post_service.go`, `router.go` and `mock_repo.go`. Parallel work needs separate worktrees and a
  rebase per merge; serial order T7, T6, T4, T5 is the low-conflict default.
- Each ticket is a vertical slice for one behavior and carries every layer it needs. Its `Type`
  lists the AGENTS.md ticket types whose rules apply (union of rules).

## Pre-flight: Database Ports

AGENTS.md lists Postgres on 5435/5436 and PgAdmin on 5051. Actual `.env` (gitignored, machine-local)
has `DB_PORT=5437`, `TEST_DB_PORT=5438`, `PGADMIN_PORT=5052`. `docker-compose.yaml` defaults are
5432/5433/5050. `make migrate-*` uses `DATABASE_URL` from `.env` (currently `...:5437/ssrblog_db`).

Before T1 runs any migration:

1. `grep -E '^(DB_PORT|DATABASE_URL)' .env`
2. `docker compose ps` and confirm the `pg` container publishes that same host port.
3. `make migrate-status` must connect to `ssrblog_db` before `make migrate-up`.

Integration tests use testcontainers and ignore these ports. AGENTS.md is not edited in this
feature; correcting it is a separate `docs` chore.

## Risks

- Slug retry never converges under heavy same-title creation -> 100-attempt cap, attempts histogram exposes it, unit test at the cap.
- Suffix pushes slug past 200 chars -> base truncated to 190; unit test with a 300-char title plus suffix.
- A query forgets `deleted_at IS NULL` -> single `notDeleted()` helper, integration test per repo method, reviewer greps `post_repo.go` for builders that do not use it.
- Partial unique index would free deleted slugs -> index is full; integration test: soft-delete, then create the same title, expect `-2`.
- Accent folding without `golang.org/x/text` mishandles rare diacritics (unmapped letters are dropped) -> accepted for MVP. Promoting `x/text` (currently indirect) to a direct dependency needs an ADR and is rejected here.
- `published_at` read-modify-write race between two admins -> worst case the timestamp differs by milliseconds. Accepted for single-admin site.
- No audit facility exists -> `slog` audit event only; a durable audit table needs its own ticket and ADR.
- Public endpoints have no rate limit (same as existing `/users` reads) -> accepted; revisit if abused.
- Stored content may contain script tags -> stored as written per spec; `docs/API.md` states clients must render safely.
- `router.Setup` may be awkward to call in a test -> fall back to extracting `registerPostRoutes(...)`, called by `Setup` and by the test.
- `PUT` needs `title`, `content`, `status` on every call -> documented in API.md; a `PATCH` is a possible follow-up, not in scope.

## Test Strategy

- **Unit (mock repo, no DB):** slugify table; service create/retry/cap/empty-slug; `published_at` transition matrix (draft->pub, pub->pub, pub->draft, draft->pub again); not-found paths; delete; list. Handler tests via `httptest` + Echo + real validator for each field limit, `http_url`, status enum, blank title, bad JSON (400), bad id (400).
- **Integration (`-tags=integration`, testcontainers):** repo CRUD; unique violation -> `ErrSlugTaken`; N concurrent inserts of one slug yield exactly one winner; soft-deleted rows excluded from every read/update/delete and public list count; slug column unchanged by `Update`; pagination totals and ordering; concurrent same-title creation through the real service yields distinct slugs and zero errors (`internal/service`, tagged).
- **RBAC matrix (router-level):** every admin post route x {anonymous -> 401, `user` -> 403, `editor` -> 403, `admin` -> success}; missing CSRF token on POST/PUT/DELETE rejected; rate limit returns 429 past burst; nothing written on denial.
- **Manual smoke (auth-adjacent):** register -> verify -> login as admin -> create draft -> public 404 -> publish -> public list/read -> edit title (slug stable) -> unpublish -> public 404 -> delete -> public 404 -> recreate same title (gets `-2`) -> logout. Check `/metrics` for the new series.
- **Gates per ticket:** `make lint` (zero warnings), `make test` (race), `go test -tags=integration ./internal/repository/ ./internal/service/` when either changed, `make swagger` no diff when handlers changed.

## Rollback

Routes and table are purely additive. Roll back by reverting the deploy: the app then has no post
routes and the unused `posts` table is harmless. Run `make migrate-down` only when the table holds
no data worth keeping, because the down migration drops it. Migrations are never edited once
applied; a schema fix ships as a new migration.
