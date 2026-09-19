# ssrBlog -- Agent Instructions

> Project-specific rules. Inherits from ~/.ai-workflow/global/AGENTS.md

---

## Project Context

Production-ready Go backend (Echo v4 + Squirrel + pgx + Goose + scs).
Complete auth flow + RBAC. Deployed on Railway.
14 phases complete. Treat this as stable -- changes need justification.

---

## Constitution

### Code Quality
- Deep modules over shallow files
- Follow existing patterns in internal/
- No new dependencies without ADR
- Handlers thin, services heavy, repositories own all SQL
- Errors via apperror package only

### Testing
- Unit tests for business logic (mocks)
- Integration tests with testcontainers for repositories
- Race detector in CI -- no data races
- Test before claiming done

### Maintainability
- Migrations: always up + down, always indexed
- Swagger: regenerate on any API change
- No breaking changes to public endpoints without version bump

---

## Ticket-Type Rules

### Migration Ticket
- File: internal/database/migrations/NNNN_name.sql
- Must have up + down
- Must add index if querying by new column
- Run: make migrate-up && make migrate-down && make migrate-up

### Handler Ticket
- File: internal/handler/<name>.go
- Must have: validation test + error-path test
- Return echo.NewHTTPError for HTTP errors
- Thin only -- delegate to service

### Service Ticket
- File: internal/service/<name>.go
- Unit tests with mock repository
- Business logic only -- no SQL, no HTTP

### Repository Ticket
- File: internal/repository/<name>.go
- Squirrel with sq.Dollar placeholder
- Integration test with testcontainers
- Handle pgx.ErrNoRows explicitly

### Auth/RBAC Ticket (sensitive)
- Extra reviewer required
- Security checklist:
  - [ ] Session renewed on privilege change
  - [ ] CSRF token validated
  - [ ] Rate limit applied
  - [ ] Audit log entry
- Update docs/ARCHITECTURE.md if flow changes

### API Change Ticket
- Run make swagger before commit
- Update docs/API.md with example
- Check pagination format consistency

---

## Key Files

| Path | Purpose |
|---|---|
| cmd/api/main.go | Entry point |
| internal/handler/ | HTTP handlers (thin) |
| internal/service/ | Business logic |
| internal/repository/ | Data access (Squirrel) |
| internal/model/ | Domain models |
| internal/middleware/ | Auth, CSRF, RBAC, rate limit, metrics |
| internal/database/migrations/ | Goose migrations |
| internal/apperror/ | Error types |
| pkg/metrics/ | Prometheus metrics |
| pkg/pagination/ | Pagination helpers |
| docs/ARCHITECTURE.md | Architecture rules |
| docs/adr/ | Decision records |
| .ai-workflow/ | AI workflow (templates, prompts, features) |

---

## Database

- Postgres main: port 5435
- Postgres test: port 5436
- PgAdmin: port 5051
- Migrations: make migrate-up
- Indexes: already optimized -- no redundant
- pg_stat_statements enabled (dev only)

---

## Commands

| Command | Purpose |
|---|---|
| make dev | Hot reload |
| make test | Unit tests |
| make lint | Linter (golangci-lint v2.13.2) |
| make swagger | Regenerate Swagger |
| make migrate-up / down / status | Migrations |
| make db-start / stop / reset | DB lifecycle |
| go test -tags=integration ./internal/repository/ | Integration |

---

## Stack-Specific Rules

### Squirrel
- Always sq.Dollar placeholder
- Never string concatenation
- Repository pattern only

### Echo
- echo.NewHTTPError for HTTP errors
- Validate with c.Validate()
- Thin handlers

### pgx
- pgxpool, not single connection
- context.Context everywhere
- Handle pgx.ErrNoRows explicitly

### Sessions (scs)
- scs.SessionManager
- Always RenewToken on login
- Store only user_id in session

### Metrics
- Add Prometheus counter/histogram for new endpoints
- Don't add high-cardinality labels

---

## Known Decisions

- Session-based auth (not JWT) -> ADR 0001
- Postgres session store (not in-memory) -> ADR 0002
- CSRF double-submit cookie -> ADR 0003
- Bcrypt cost 10 -> ADR 0004
- 5 failed logins = 15 min lockout -> ADR 0005
- Pagination: data + meta format, default limit 20, max 100

> Any change to these requires a new ADR in docs/adr/.

---

## Do NOT

- Put SQL in handlers (repository only)
- Use context.Background() in request path
- Return raw errors (use apperror)
- Add deps without ADR
- Commit without make lint && make test
- Change public API without make swagger
- Skip integration test for repository changes

---

## Before Claiming Done

1. make lint -- zero warnings
2. make test -- all pass
3. go test -tags=integration ./internal/repository/ -- pass (if repo changed)
4. make swagger -- no diff (if API changed)
5. Manual smoke (if auth changed):
   - register -> verify -> login -> protected route -> logout
6. Update checklist.md
7. Update notes.md (decisions, blockers, context clears)

---

Last updated: 2026-09-20
