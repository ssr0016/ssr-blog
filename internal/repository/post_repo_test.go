package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

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
	repo.ListPublishedFunc = func(context.Context, int, int) ([]model.PostSummary, int64, error) { return nil, 0, boom }
	repo.GetPublishedBySlugFunc = func(context.Context, string) (*model.Post, error) { return nil, boom }

	if _, err := repo.Create(ctx, model.Post{}); !errors.Is(err, boom) {
		t.Errorf("Create() error = %v, want override", err)
	}
	if _, err := repo.GetByID(ctx, 1); !errors.Is(err, boom) {
		t.Errorf("GetByID() error = %v, want override", err)
	}
	if _, _, err := repo.ListPublished(ctx, 1, 20); !errors.Is(err, boom) {
		t.Errorf("ListPublished() error = %v, want override", err)
	}
	if _, err := repo.GetPublishedBySlug(ctx, "x"); !errors.Is(err, boom) {
		t.Errorf("GetPublishedBySlug() error = %v, want override", err)
	}
}

func TestMockPostRepo_ListPublished(t *testing.T) {
	ctx := context.Background()
	repo := NewMockPostRepo()
	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	at := func(d time.Duration) *time.Time { v := base.Add(d); return &v }

	for _, p := range []model.Post{
		{Title: "Old", Slug: "old", Content: "x", Status: model.PostStatusPublished, PublishedAt: at(0)},
		{Title: "Draft", Slug: "draft", Content: "x", Status: model.PostStatusDraft},
		{Title: "New", Slug: "new", Content: "x", Status: model.PostStatusPublished, PublishedAt: at(time.Hour)},
		{Title: "Tie", Slug: "tie", Content: "x", Status: model.PostStatusPublished, PublishedAt: at(time.Hour)},
	} {
		if _, err := repo.Create(ctx, p); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	items, total, err := repo.ListPublished(ctx, 1, 2)
	if err != nil || total != 3 {
		t.Fatalf("ListPublished() total = %d, err = %v, want 3, nil", total, err)
	}
	if len(items) != 2 || items[0].Slug != "tie" || items[1].Slug != "new" {
		t.Errorf("page 1 = %+v, want [tie new] (published_at DESC, id DESC)", items)
	}

	items, _, _ = repo.ListPublished(ctx, 2, 2)
	if len(items) != 1 || items[0].Slug != "old" {
		t.Errorf("page 2 = %+v, want [old]", items)
	}

	items, total, _ = repo.ListPublished(ctx, 9, 2)
	if items == nil || len(items) != 0 || total != 3 {
		t.Errorf("past the end = %#v, total %d, want empty non-nil slice and total 3", items, total)
	}
}

func TestMockPostRepo_GetPublishedBySlug(t *testing.T) {
	ctx := context.Background()
	repo := NewMockPostRepo()
	now := time.Now()
	_, _ = repo.Create(ctx, model.Post{Title: "Live", Slug: "live", Content: "x", Status: model.PostStatusPublished, PublishedAt: &now})
	_, _ = repo.Create(ctx, model.Post{Title: "Wip", Slug: "wip", Content: "x", Status: model.PostStatusDraft})

	if got, err := repo.GetPublishedBySlug(ctx, "live"); err != nil || got == nil || got.Slug != "live" {
		t.Errorf("GetPublishedBySlug(live) = %+v, %v", got, err)
	}
	for _, slug := range []string{"wip", "missing"} {
		if got, err := repo.GetPublishedBySlug(ctx, slug); err != nil || got != nil {
			t.Errorf("GetPublishedBySlug(%s) = %+v, %v, want nil, nil", slug, got, err)
		}
	}
}
