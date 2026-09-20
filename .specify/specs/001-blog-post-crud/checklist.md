# Feature Checklist

## Feature: Blog Post CRUD (001-blog-post-crud)
**Started:** 2026-09-20
**Branch:** main
**ADR:** none (no irreversible decisions in this feature)

---

## Phase 1: Understand

- [x] Grilled -- Q&A log in `clarifications.md`
- [x] Spec written (`spec.md`)
- [x] Plan written (`plan.md`)
- [x] Tickets generated (`tasks.md`, T1-T9)

## Phase 2: Build

Per ticket: clear context -> load AGENTS + ticket -> test-first -> commit

- [x] T1: Migration -- commit 645309f
- [x] T2: Foundation -- commit 1f1185b
- [x] T3a: Repository -- commit f08dae8
- [x] T3b: Service -- commit 7a34180
- [x] T3c: Handler + RBAC -- commit 3e961d4
- [x] T4: Admin edit -- commit eccd8c3
- [x] T5: Admin soft-delete -- commit d91ee43
- [x] T6: Admin list -- commit 70f8bbe, leftover 15bb479
- [x] T7: Public list + read -- commit 6db8e94
- [x] T8: Swagger + API docs -- commit 079ab07
- [x] T9: RBAC gate + smoke (this ticket)

Note: T7 had no red phase. Noted as follow-up.

## Phase 3: Verify (T9 converge, live against dev app + dev DB)

### RBAC matrix (28 cells, all verified)

| Route | anon | user | editor | admin | Expected |
|---|---|---|---|---|---|
| GET /posts | 200 | 200 | 200 | 200 | OK |
| GET /posts/:slug | 200 | 200 | 200 | 200 | OK |
| POST /admin/posts | 403* | 403 | 403 | 201 | WARN anon 403 not 401 |
| GET /admin/posts | 401 | 403 | 403 | 200 | OK |
| GET /admin/posts/:id | 401 | 403 | 403 | 200 | OK |
| PUT /admin/posts/:id | 403* | 403 | 403 | 200 | WARN anon 403 not 401 |
| DELETE /admin/posts/:id | 403* | 403 | 403 | 204 | WARN anon 403 not 401 |

*CSRF middleware runs before auth on unsafe methods. Anonymous writes get CSRF rejection (403) instead of auth rejection (401). Spec SC #8 satisfied: no state changes from denied cells (verified via DB row counts). Documented as design in notes.md.

### Smoke test (15/15 pass, live)

- [x] Create draft, slug generated from title (accents stripped)
- [x] Public GET draft -> 404
- [x] Public list excludes draft
- [x] Publish -> published_at set
- [x] Public GET published -> 200
- [x] Public list includes it, newest first
- [x] Title edit -> slug unchanged
- [x] Unpublish -> published_at=null, public GET 404
- [x] Re-publish -> new published_at
- [x] Delete -> 204, public GET 404
- [x] Same title re-created -> suffixed slug
- [x] Blank title -> 422
- [x] Bad URL scheme -> 422
- [x] Bad status -> 422
- [x] Title with no slug-usable chars -> 422

### Not-found parity (byte-identical 404)

- [x] Draft slug -> {"error":"NOT_FOUND","message":"post not found"}
- [x] Missing slug -> identical body
- [x] Soft-deleted slug -> identical body

### Metrics

- [x] blog_post_writes_total{operation=...} present, labeled by operation
- [x] 5 bounded values: create, update, delete, publish, unpublish
- [x] blog_post_slug_attempts present
- [x] http_requests_total{method,path,status} present, path uses route templates
- [x] No high-cardinality labels (no post_id, slug, or user_id in any metric)
- [ ] BUG: status label is always 200 for apperror responses (Known Issue #3, root cause in internal/middleware/logger.go:59)

### Audit log

- [x] 16 audit events captured during T9 (7 create, 3 update, 3 publish, 1 unpublish, 2 delete)
- [x] Every event has audit=true, actor_user_id, action, post_id, slug
- [x] No audit event on any denied request
- [x] Post content never appears in logs

### Gates

- [x] make test -- all packages pass
- [x] go test -tags=integration ./internal/repository/ ./internal/service/ -- pass
- [x] make swagger -- byte-identical on re-run (no diff)
- [x] make lint -- clean
- [x] Working tree clean except for the T9 spec artifacts (as of T9; converge round 1-4 fixes are uncommitted until the loop ends)

---

## Spec Success Criteria (from spec.md)

- [x] An admin can create, edit, list, view, and soft-delete posts, including drafts. -> smoke + matrix
- [x] A new post gets a slug from its title. Slugs are unique, including under simultaneous creation. -> smoke #1, integration
- [x] A slug does not change when the title is edited. -> smoke #7
- [x] Published date is set on first publish, kept on later edits, and cleared on return to draft. -> smoke #4, #8, #9
- [x] Deleted posts are hidden from all reads and their slugs are never reused. -> smoke #10, matrix DB check
- [x] Anonymous visitors can list and read published posts. -> matrix rows 1-2, smoke #5, #6
- [x] Anonymous visitors cannot tell that a draft exists. -> not-found parity
- [x] Non-admins and anonymous callers cannot create, edit, or delete. Nothing changes when they try. -> matrix + DB row count
- [x] Every validation edge case above returns a clear error and saves nothing. -> smoke #12-#15, handler tests
- [x] Public list uses the standard pagination format and limits. -> pkg/pagination
- [x] Tests cover admin-only access and each validation and not-found path. -> handler tests, RBAC matrix in router_post_test.go
- [x] Repository behavior, including slug uniqueness under concurrent creation and soft-delete exclusion, is covered by integration tests. -> repo integration suite passes (~90s)
- [x] Swagger regenerated and docs/API.md updated. -> T8 commit 079ab07
- [x] make lint and make test pass with no warnings or failures.

### Spec Edge Cases (from spec.md)

- [x] Two posts with same title -> unique slugs. -> matrix
- [x] Two admins same title simultaneous -> DB unique index + retry. -> repo integration
- [x] Title edited after publishing -> slug unchanged. -> smoke #7
- [x] Title has no letters or numbers -> validation error. -> smoke #15
- [x] Title has accents or mixed case -> lowercase URL-safe. -> smoke #1
- [x] Title is non-Latin -> empty slug -> validation error. -> unit tests
- [x] Title or content missing/blank -> validation error. -> smoke #12, handler tests
- [x] Field over max length -> validation error. -> handler tests
- [x] Cover image URL not http/https -> validation error. -> smoke #13
- [x] Status other than draft/published -> validation error. -> smoke #14
- [x] Publish an already-published post -> succeeds, date unchanged. -> smoke #9
- [x] Move published back to draft -> hidden, date cleared. -> smoke #8
- [x] Re-publish -> new date. -> smoke #9
- [x] Public request for a draft slug -> not found, identical. -> not-found parity
- [x] Public request for a missing slug -> not found. -> not-found parity
- [x] Public request for a soft-deleted slug -> not found. -> not-found parity
- [x] Admin edits/deletes a nonexistent id -> not found. -> handler tests
- [x] Admin views/lists a soft-deleted post -> not found / excluded. -> repo integration
- [x] Admin deletes a published post -> gone from public list, link returns not found. -> smoke #10
- [x] Admin deletes, then re-creates same title -> suffixed slug, deleted slug never reused. -> integration tests
- [x] Public list has no published posts -> empty data, valid meta. -> repo integration
- [x] Public list page beyond the last page -> empty data, valid meta. -> repo integration (incl. huge page values, converge round 1)
- [x] Pagination limit above 100 or below 1 -> clamped. -> pkg/pagination
- [x] Logged-in non-admin create/edit/delete -> forbidden, no changes. -> matrix rows 3, 6, 7
- [~] Unauthenticated caller create/edit/delete -> no changes. 401 with a valid CSRF token; bare request gets 403 (CSRF runs before auth). Accepted deviation, documented in API.md; needs product sign-off. -> matrix rows 3, 6, 7
- [x] Malicious markdown/HTML in content -> stored as written, not rendered. -> handler stores raw

---

## Phase 4: Ship

- [x] make lint && make test pass
- [x] Integration tests pass
- [x] make swagger no diff
- [x] Commit message follows convention
- [ ] PR opened -- single-branch workflow, no PR
- [ ] CI passing (not yet verified; local make lint/test/integration pass)
- [ ] Rollback plan documented
- [x] Pushed to origin/main

---

## Notes

- Context clears: ~11 across T1-T9
- Blockers: none blocking
- Decisions: see notes.md
- ADRs created: none
- Known issues carried forward: see notes.md post-T9 list
