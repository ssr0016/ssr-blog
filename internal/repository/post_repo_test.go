package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/ssr0016/ssr-blog/internal/model"
)

func TestIsSlugTaken(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"unique violation on slug index", &pgconn.PgError{Code: "23505", ConstraintName: "idx_posts_slug"}, true},
		{"wrapped unique violation on slug index", fmt.Errorf("scan post: %w", &pgconn.PgError{Code: "23505", ConstraintName: "idx_posts_slug"}), true},
		{"unique violation on another constraint", &pgconn.PgError{Code: "23505", ConstraintName: "posts_pkey"}, false},
		{"check violation", &pgconn.PgError{Code: "23514", ConstraintName: "posts_status_check"}, false},
		{"value too long", &pgconn.PgError{Code: "22001"}, false},
		{"plain error", errors.New("boom"), false},
		{"nil", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isSlugTaken(tt.err); got != tt.want {
				t.Errorf("isSlugTaken(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestMockPostRepo_CreateAndGet(t *testing.T) {
	ctx := context.Background()
	repo := NewMockPostRepo()

	first, err := repo.Create(ctx, model.Post{Title: "A", Slug: "a", Content: "x", Status: model.PostStatusDraft})
	if err != nil || first == nil {
		t.Fatalf("Create() = %v, %v", first, err)
	}
	second, err := repo.Create(ctx, model.Post{Title: "B", Slug: "b", Content: "x", Status: model.PostStatusDraft})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if first.ID == 0 || second.ID == first.ID {
		t.Errorf("ids = %d, %d, want distinct and non-zero", first.ID, second.ID)
	}
	if first.CreatedAt.IsZero() || first.UpdatedAt.IsZero() {
		t.Error("timestamps must be set")
	}

	got, err := repo.GetByID(ctx, first.ID)
	if err != nil || got == nil || got.Slug != "a" {
		t.Errorf("GetByID() = %+v, %v", got, err)
	}

	got, err = repo.GetByID(ctx, 999)
	if err != nil || got != nil {
		t.Errorf("GetByID(missing) = %+v, %v, want nil, nil", got, err)
	}

	if _, err := repo.Create(ctx, model.Post{Title: "A2", Slug: "a", Content: "x", Status: model.PostStatusDraft}); !errors.Is(err, ErrSlugTaken) {
		t.Errorf("duplicate slug error = %v, want ErrSlugTaken", err)
	}
}

func TestMockPostRepo_OverrideFuncs(t *testing.T) {
	ctx := context.Background()
	repo := NewMockPostRepo()
	boom := errors.New("boom")

	repo.CreateFunc = func(context.Context, model.Post) (*model.Post, error) { return nil, boom }
	repo.GetByIDFunc = func(context.Context, int64) (*model.Post, error) { return nil, boom }

	if _, err := repo.Create(ctx, model.Post{}); !errors.Is(err, boom) {
		t.Errorf("Create() error = %v, want override", err)
	}
	if _, err := repo.GetByID(ctx, 1); !errors.Is(err, boom) {
		t.Errorf("GetByID() error = %v, want override", err)
	}
}
