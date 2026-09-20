package testutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type TestDB struct {
	Container *postgres.PostgresContainer
	Pool      *pgxpool.Pool
	DSN       string
}

func SetupPostgres(t *testing.T) *TestDB {
	t.Helper()
	ctx := context.Background()

	container, err := postgres.Run(ctx,
		"postgres:15.3-alpine",
		postgres.WithDatabase("test_db"),
		postgres.WithUsername("test_user"),
		postgres.WithPassword("test_password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres: %v", err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get conn string: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("failed to connect: %v", err)
	}

	t.Cleanup(func() {
		pool.Close()
		_ = container.Terminate(ctx)
	})

	return &TestDB{Container: container, Pool: pool, DSN: dsn}
}

func (tdb *TestDB) RunMigrations(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	migrations := []string{
		`CREATE TABLE IF NOT EXISTS roles (
			id BIGSERIAL PRIMARY KEY,
			name VARCHAR(50) UNIQUE NOT NULL,
			description VARCHAR(255) NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS permissions (
			id BIGSERIAL PRIMARY KEY,
			name VARCHAR(100) UNIQUE NOT NULL,
			resource VARCHAR(50) NOT NULL,
			action VARCHAR(50) NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS role_permissions (
			role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
			permission_id BIGINT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (role_id, permission_id)
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id BIGSERIAL PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			name VARCHAR(100) NOT NULL,
			password_hash VARCHAR(255) NOT NULL,
			role_id BIGINT REFERENCES roles(id) ON DELETE RESTRICT,
			email_verified BOOLEAN NOT NULL DEFAULT FALSE,
			email_verified_at TIMESTAMPTZ,
			failed_login_attempts INT NOT NULL DEFAULT 0,
			locked_until TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			token TEXT PRIMARY KEY,
			data BYTEA NOT NULL,
			expiry TIMESTAMPTZ NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS verification_tokens (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token VARCHAR(255) UNIQUE NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			used_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS password_resets (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			token VARCHAR(255) UNIQUE NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			used_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`INSERT INTO roles (name, description) VALUES ('user', 'Default user role') ON CONFLICT (name) DO NOTHING`,
	}

	for _, m := range migrations {
		if _, err := tdb.Pool.Exec(ctx, m); err != nil {
			t.Fatalf("migration failed: %v\nSQL: %s", err, m)
		}
	}

	// Newer tables run the real goose migration files, so tests exercise the actual schema
	// (index names, constraints) instead of a copy that can drift.
	tdb.applyMigrationFile(t, "*_create_posts.sql")
}

// applyMigrationFile executes the Up section of the single migration file in
// internal/database/migrations that matches glob.
func (tdb *TestDB) applyMigrationFile(t *testing.T, glob string) {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate testutil source directory")
	}
	pattern := filepath.Join(filepath.Dir(thisFile), "..", "database", "migrations", glob)
	files, err := filepath.Glob(pattern)
	if err != nil || len(files) != 1 {
		t.Fatalf("want exactly one migration matching %s, got %v (err %v)", pattern, files, err)
	}

	raw, err := os.ReadFile(files[0]) // #nosec G304 - path is built from a fixed directory and a test-supplied glob
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}

	up := string(raw)
	if i := strings.Index(up, "-- +goose Down"); i >= 0 {
		up = up[:i]
	}
	var sql strings.Builder
	for _, line := range strings.Split(up, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "-- +goose") {
			continue
		}
		sql.WriteString(line)
		sql.WriteByte('\n')
	}

	if _, err := tdb.Pool.Exec(context.Background(), sql.String()); err != nil {
		t.Fatalf("apply %s: %v", filepath.Base(files[0]), err)
	}
}

func (tdb *TestDB) TruncateAll(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	tables := []string{
		"password_resets", "verification_tokens", "role_permissions",
		"permissions", "sessions", "users", "roles", "posts",
	}
	for _, table := range tables {
		_, _ = tdb.Pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE", table))
	}

	// Re-seed default roles (needed for user creation)
	_, _ = tdb.Pool.Exec(ctx, `INSERT INTO roles (name, description) VALUES ('user', 'Default user role') ON CONFLICT (name) DO NOTHING`)
}
