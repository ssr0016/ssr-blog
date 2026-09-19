# ssrBlog

> Personal blog site — content, projects, and portfolio.

A Go backend for a personal blog. Built on the template-go-echo-squirrel starter kit.

---

## Features

### Core
- Echo v4 — fast, minimalist web framework
- PostgreSQL + pgx — high-performance database driver
- Squirrel — type-safe SQL query builder
- Goose — database migrations
- Type-safe config
- Structured logging (log/slog)
- Standardized errors (apperror)

### Auth (from starter kit)
- Session-based auth (alexedwards/scs + Postgres store)
- CSRF protection (double-submit cookie)
- Rate limiting (per-IP)
- Email verification
- Password reset
- Account lockout (5 failed, 15 min)
- Bcrypt hashing (cost 10)

### RBAC (from starter kit)
- Roles (admin, editor, user)
- Permissions (granular)
- Role-permission mapping
- RequireRole / RequirePermission middleware

### API (from starter kit)
- Swagger UI
- Pagination
- Health checks
- Prometheus metrics

### Blog Features (planned)
- Post CRUD
- Tags
- Draft / Published workflow
- Cover image upload
- Public read endpoints
- Full-text search

---

## Quick Start

Clone:

    git clone https://github.com/ssr0016/ssr-blog.git
    cd ssr-blog

Setup:

    cp .env.example .env
    # Edit .env

Start DB:

    make db-start

Migrate:

    make migrate-up

Run:

    make dev

---

## AI Workflow

This project uses a structured AI workflow.

- Global rules: ~/.ai-workflow/global/AGENTS.md
- Project rules: AGENTS.md
- Constitution: .specify/memory/constitution.md
- Slash commands: .claude/commands/speckit-*.md
- Templates: .ai-workflow/templates/
- ADRs: docs/adr/

Workflow: Grill -> Spec -> Plan -> Tickets -> Implement -> Verify -> Ship

---

## Makefile Commands

    make help           Show all
    make dev            Hot reload
    make run            Direct run
    make build          Build binary
    make test           Run tests
    make lint           Run linter
    make swagger        Generate Swagger
    make db-start       Start DB
    make db-stop        Stop DB
    make db-reset       Wipe + restart
    make migrate-up     Apply migrations
    make migrate-down   Rollback
    make migrate-status Check status

---

## Tech Stack

- Language: Go 1.26
- Framework: Echo v4
- Database: PostgreSQL 15
- Driver: pgx
- Query builder: Squirrel
- Migrations: Goose
- Auth: alexedwards/scs
- Logging: log/slog

---

## License

MIT
