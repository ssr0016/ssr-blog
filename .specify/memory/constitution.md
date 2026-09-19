# Constitution: ssrBlog

> Project principles. Non-negotiable. All decisions must align.

---

## Article 1: Architecture

### 1.1 Layer Discipline
- Handlers: thin, HTTP only. No SQL. No business logic.
- Services: business logic. No SQL. No HTTP.
- Repositories: data access. All SQL here. Squirrel only.
- Models: domain types. No behavior.

**Rationale:** Testability, separation of concerns, deep modules.

### 1.2 Deep Modules > Shallow Files
- One heavy interface beats ten thin files.
- Design interfaces by hand. Delegate implementations to AI.
- Package cohesion: related code together.

### 1.3 Dependency Direction
handler -> service -> repository -> database
   |          |            |
   +----------+------------+-> model
   |
   +-> apperror, middleware

No reverse dependencies. No cycles.

---

## Article 2: Data

### 2.1 Migrations
- Always up + down. No exceptions.
- Always indexed if querying by new column.
- No editing applied migrations.
- File: internal/database/migrations/NNNN_name.sql

### 2.2 Queries
- Squirrel only. No string concatenation.
- Always sq.Dollar placeholder (Postgres).
- Handle pgx.ErrNoRows explicitly.
- Context everywhere.

### 2.3 Sessions
- Postgres store (not in-memory).
- Store only user_id.
- RenewToken on login.
- CSRF double-submit cookie.

---

## Article 3: Security

### 3.1 Auth
- Session-based (not JWT). See ADR 0001.
- Bcrypt cost 10. See ADR 0004.
- 5 failed logins -> 15 min lockout. See ADR 0005.

### 3.2 Input
- Validate all input via c.Validate().
- No raw SQL. Parameterized only.
- No secrets in code. Env vars only.
- No sensitive data in logs.

### 3.3 CSRF
- Double-submit cookie pattern. See ADR 0003.
- SameSite=Lax on session cookie.

---

## Article 4: Testing

### 4.1 Unit Tests
- Business logic only.
- Mock repositories.
- Fast. No DB.

### 4.2 Integration Tests
- Repositories with testcontainers.
- Real Postgres.
- Tag: -tags=integration.

### 4.3 Before Claiming Done
- make lint -- zero warnings.
- make test -- all pass.
- Integration tests -- pass (if repo changed).
- make swagger -- no diff (if API changed).

---

## Article 5: API

### 5.1 Endpoints
- RESTful. Versioned: /api/v1/.
- Pagination: data + meta format.
- Default limit 20, max 100.

### 5.2 Errors
- Use apperror package only.
- HTTP errors via echo.NewHTTPError.
- No raw error returns to client.

### 5.3 Swagger
- Regenerate on any API change.
- make swagger before commit.

---

## Article 6: Code Quality

### 6.1 Dependencies
- No new deps without ADR.
- Prefer stdlib.
- Justify in commit message.

### 6.2 Errors
- Wrap with context.
- Use apperror types.
- Log at handler level only.

### 6.3 Context
- context.Context everywhere.
- No context.Background() in request path.
- Respect cancellation.

---

## Article 7: Process

### 7.1 Before Coding
- Grill first. No assumptions.
- Spec written. Non-goals explicit.
- Plan as DAG. Tickets = vertical slices.

### 7.2 While Coding
- One ticket = one fresh context.
- Test first, then code.
- No batching.

### 7.3 Before Shipping
- Multi-agent review.
- Converged (2 consecutive clean).
- ADR if decision changed.

---

## Amendments

Any change to this constitution requires:
1. New ADR in docs/adr/
2. Update this file
3. Commit message: docs(constitution): ...

---

Version: 1.0
Last updated: 2026-09-20
