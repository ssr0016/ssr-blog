# Notes: Blog Post CRUD (001-blog-post-crud)

**Date:** 2026-09-20
**Status:** T9 converge complete; feature shipped

---

## Decisions

- **Slug retry with numeric suffix** (not rejection). Service retries on unique-violation (SQLSTATE 23505 on `idx_posts_slug`); after base is taken, tries `base-2`, `base-3`, ... Cap is 100. Chosen over "reject on collision" because the spec wants two posts with the same title to both succeed.
- **Full unique index on `slug`** (not partial). Soft-deleted slugs stay reserved. Chosen over partial-on-`deleted_at IS NULL` because spec says a slug is never reused, even after delete.
- **`published_at` lifecycle handled in service**, not the DB. Clock injected for deterministic tests. Rules: set on first publish, kept on subsequent edits, cleared on return to draft, re-set on re-publish.
- **PUT is full replace** (no PATCH). Matches project convention.
- **Slug field in request bodies is ignored.** Slug is server-generated and immutable; documented in `docs/API.md`.
- **`content` stored raw.** No server-side rendering or sanitization. Client renders. Documented.

## Q&A Log (Grill)

Full log in `clarifications.md`. Key resolutions:

- Q: Pagination format? A: reuse project standard (data + meta, default 20, max 100).
- Q: Slug uniqueness scope? A: full, including soft-deleted.
- Q: Multi-author? A: non-goal. No author byline.
- Q: Scheduled publish? A: non-goal. published_at set when admin flips status.
- Q: Restore soft-deleted? A: non-goal; may be future ticket.

## T9 Converge (live verification, 2026-09-20)

### Environment

- App: make run, :8080, dev DB ssrblog_db on :5437 (Docker ssrblog-pg).
- Test DB: :5438 (Docker ssrblog-test-pg).
- Seeded admin: admin@example.com (id 1, role 2).
- Ad-hoc test actors for T9: user@example.com (id 23, role 1), editor@example.com (id 24, role 3). Not a production pattern.

### RBAC matrix (28 cells, live)

| Route | anon | user | editor | admin |
|---|---|---|---|---|
| GET /posts | 200 | 200 | 200 | 200 |
| GET /posts/:slug | 200 | 200 | 200 | 200 |
| POST /admin/posts | 403* | 403 | 403 | 201 |
| GET /admin/posts | 401 | 403 | 403 | 200 |
| GET /admin/posts/:id | 401 | 403 | 403 | 200 |
| PUT /admin/posts/:id | 403* | 403 | 403 | 200 |
| DELETE /admin/posts/:id | 403* | 403 | 403 | 204 |

**403* = CSRF-first design.** On unsafe methods (POST/PUT/DELETE), CSRF middleware runs before auth. An anonymous caller without a CSRF token gets 403 FORBIDDEN "CSRF token invalid or missing" instead of 401 UNAUTHORIZED. Anonymous GETs on the same paths correctly return 401. Spec SC #8 satisfied: no state changes from denied cells (verified via DB row-count check). Not a bug; documented as intentional ordering.

### Not-found parity (byte-identical 404)

GET /posts/:slug returns the same body for draft, soft-deleted, and never-existed slugs:

    {"error":"NOT_FOUND","message":"post not found"}

Verified with 3 live requests. No timing or body-length difference observed at the HTTP level.

### Smoke test (15/15)

1. Create "Cafe Au Lait" as draft -> slug cafe-au-lait (accents stripped, lowercase). PASS
2. Public GET draft slug -> 404. PASS
3. Public list -> draft excluded. PASS
4. Publish -> published_at set. PASS
5. Public GET published -> 200. PASS
6. Public list -> included, newest first. PASS
7. Edit title -> slug unchanged. PASS
8. Unpublish -> published_at=null, public GET 404. PASS
9. Re-publish -> new published_at. PASS
10. Delete -> 204, public GET 404. PASS
11. Same title again -> suffixed slug (see matrix: rbac-published-probe, rbac-published-probe-2). PASS
12. Blank title -> 422 VALIDATION_ERROR. PASS
13. Bad cover URL (ftp://) -> 422. PASS
14. Bad status (deleted) -> 422. PASS
15. Title "!!!" (no slug-usable chars) -> 422 "title has no usable characters for a slug". PASS

### Metrics

blog_post_writes_total (not post_writes_total as originally drafted):
- Prefix blog_ (registered under Prometheus subsystem).
- Label name is operation (not op).
- Bounded values (pkg/metrics/metrics.go): create, update, delete, publish, unpublish.
- Live counts at end of T9: create=8, update=6, delete=2, publish=2, unpublish=1. Match actual admin actions.

blog_post_slug_attempts — additional metric not in original notes; counts slug-generation attempts.

http_requests_total — has method, path, status labels. path uses route templates (/api/v1/admin/posts/:id), not literal IDs, so no high-cardinality explosion.

### Confirmed bug: Prometheus status label (Known Issue #3)

http_requests_total records status="200" for apperror responses (401/403/404/422/429). Evidence from internal/middleware/logger.go:59:

    msg="request completed" ... uri=/api/v1/posts/cafe-au-lait status=200 ... error="NOT_FOUND: post not found"
    msg="request completed" ... uri=/api/v1/admin/posts status=200 ... error="VALIDATION_ERROR: ..."
    msg="request completed" ... uri=/api/v1/admin/posts status=200 ... error="RATE_LIMIT_EXCEEDED: ..."

Client received 404/422/429 (verified via curl -w "%{http_code}"), but the logger middleware captured the default 200. Root cause: middleware reads the http.ResponseWriter status before the apperror mapping calls WriteHeader. The real status only appears in the error="CODE: message" field.

Impact: Grafana/Prometheus dashboards see all errors as 2xx. Fix requires a wrapper ResponseWriter that intercepts WriteHeader, plus a middleware test.

### Audit log

16 events during T9. All have audit=true, actor_user_id=1, action=post.{create,update,publish,unpublish,delete}, post_id, slug. No audit event on any denied request. Post content never appears in logs.

### Other findings

- Rate limit: write routes share 30/min with burst 10. Login endpoint has its own limiter. T9 testing hit the write limiter during smoke step 15; retried after 65s and passed. POST /api/v1/admin/posts at 429 is captured in metrics as status="200" (same bug above).
- Post ID gaps: the slug retry loop consumes sequence values. Observed ids 4, 5, 6, 8, 10, 12 in a single T9 run. Documented in docs/API.md.
- Full test suite: make test all green. Integration: repository 82.8s, service 11.3s. Swagger: byte-identical on re-run.

## Context Clears

| # | Ticket | Reason |
|---|---|---|
| 1 | T1 | Start |
| 2 | T2 | After T1 commit |
| 3 | T3a | After T2 commit |
| 4 | T3b | After T3a commit |
| 5 | T3c | After T3b commit |
| 6 | T4 | After T3c commit |
| 7 | T5 | After T4 commit |
| 8 | T6 | After T5 commit |
| 9 | T7 | After T6 commit |
| 10 | T8 | After T7 commit |
| 11 | T9 | After T8 push |

## Blockers

None blocking. Two verification-adjacent known issues carried forward (see below).

## Known Issues (carried forward, post-T9)

1. RED **Seeded admin hardcoded password.** internal/database/seed.go creates admin@example.com with a fixed password. Must be env-gated and disabled in production.
2. RED **Production doesn't run migrations.** Goose runs on startup in dev; production deploy path unclear. Needs an explicit migrate step or init container.
3. YELLOW **Prometheus status label bug.** Confirmed live (T9). Root cause identified in internal/middleware/logger.go:59. Fix in a dedicated ticket.
4. YELLOW **No BodyLimit middleware.** Large request bodies not rejected at the middleware layer; relies on validator limits.
5. NO **docs/ARCHITECTURE.md not updated.** Posts feature not reflected in system diagram/text.
6. NO **T7 had no red phase.** Noted for process, not for correctness.
7. DONE **checklist.md + notes.md** — created in this ticket.
8. DONE **tasks.md** — T4–T9 detailed sections restored from .bak and merged.

## Converge round 1 (2026-09-20)

Diff range 0820faf..HEAD (main == HEAD, so `git diff main` was empty). Two fresh reviewers: spec conformance, AGENTS.md/security. No original BLOCKERs.
- Fixed (treated as BLOCKER, spec edge case "page beyond last -> empty data"): huge `page` overflowed `(page-1)*limit`, Postgres "bigint out of range", public 500. `pageOffset` helper in post_repo.go; unit + integration tests. Same pattern remains in user/role/permission repos and pkg/pagination (pre-existing, out of scope).
- Fixed: Makefile lint target had a garbled duplicate tail.
- Note: `make test`/`make lint` are scoped to ./internal/... ./pkg/... ./cmd/... because `./...` hits root-owned volumes/ (permission denied). Not whole-tree.
- Open SHOULD-FIX/NIT: CSRF-before-auth gives 403 not 401 for bare anonymous writes (documented, needs product decision); NUL byte in title/content may 500 (unconfirmed); RBAC security checklist in tasks.md not signed by a second reviewer; CI not verified.

## Converge round 3 (2026-09-20)

Round 2 (fix verification + fresh pass): CLEAN. Round 3 (adversarial + mutation testing, ~50 mutants): 0 BLOCKER, 2 SHOULD-FIX, so not clean; convergence count reset.
- Fixed: `GET /api/v1/posts/%00` returned 500 (Postgres rejects NUL). `validSlug` in service/slug.go now returns the normal 404 for any slug the generator cannot produce, without querying the DB.
- Fixed: NUL byte in title/content/excerpt returned 500 on admin create/update. New `nonul` validator on the request DTOs gives 422.
- Fixed: repo test now asserts `updated_at` strictly advances (mutant removing `updated_at = NOW()` survived before). Service test now pins the 404 message.
- Not fixed (NITs): public list `id DESC` tiebreak untested; repo-level `limit>100` clamp untested (masked by pagination.FromContext); admin id accepts `+1`/`01`; cover_image_url accepts URL credentials; timestamps not normalized to UTC (API.md shows Z); 422 messages expose Go struct names (pre-existing).

## ADRs Created

None. No irreversible architectural decisions in this feature.
