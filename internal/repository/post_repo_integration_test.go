//go:build integration

package repository

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ssr0016/ssr-blog/internal/database"
	"github.com/ssr0016/ssr-blog/internal/model"
	"github.com/ssr0016/ssr-blog/internal/testutil"
)

func setupPostRepo(t *testing.T) (*PostRepo, *testutil.TestDB) {
	t.Helper()

	tdb := testutil.SetupPostgres(t)
	tdb.RunMigrations(t)
	tdb.TruncateAll(t)

	db := &database.DB{
		Pool:    tdb.Pool,
		Builder: database.NewStatementBuilder(),
	}
	return NewPostRepo(db), tdb
}

func newTestPost(slug string) model.Post {
	return model.Post{
		Title:   "Title for " + slug,
		Slug:    slug,
		Content: "# Heading\n\nBody",
		Status:  model.PostStatusDraft,
	}
}

func countPosts(t *testing.T, tdb *testutil.TestDB) int {
	t.Helper()
	var n int
	require.NoError(t, tdb.Pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM posts").Scan(&n))
	return n
}

func TestPostRepo_Create_RoundTrip(t *testing.T) {
	repo, _ := setupPostRepo(t)
	ctx := context.Background()

	publishedAt := time.Now().UTC().Add(-time.Hour)
	in := model.Post{
		Title:         "Café Au Lait",
		Slug:          "cafe-au-lait",
		Content:       "<script>alert(1)</script>\n\n**bold**",
		Excerpt:       "A short excerpt",
		CoverImageURL: "https://example.com/cover.png",
		Status:        model.PostStatusPublished,
		PublishedAt:   &publishedAt,
	}

	created, err := repo.Create(ctx, in)
	require.NoError(t, err)
	require.NotNil(t, created)

	assert.NotZero(t, created.ID)
	assert.Equal(t, in.Title, created.Title)
	assert.Equal(t, in.Slug, created.Slug)
	assert.Equal(t, in.Content, created.Content, "content must be stored exactly as written")
	assert.Equal(t, in.Excerpt, created.Excerpt)
	assert.Equal(t, in.CoverImageURL, created.CoverImageURL)
	assert.Equal(t, model.PostStatusPublished, created.Status)
	require.NotNil(t, created.PublishedAt)
	assert.WithinDuration(t, publishedAt, *created.PublishedAt, time.Millisecond)
	assert.False(t, created.CreatedAt.IsZero())
	assert.False(t, created.UpdatedAt.IsZero())
	assert.Nil(t, created.DeletedAt)

	got, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, in.Title, got.Title)
	assert.Equal(t, in.Slug, got.Slug)
	assert.Equal(t, in.Content, got.Content)
	assert.Equal(t, in.Excerpt, got.Excerpt)
	assert.Equal(t, in.CoverImageURL, got.CoverImageURL)
	assert.Equal(t, model.PostStatusPublished, got.Status)
	require.NotNil(t, got.PublishedAt)
	assert.WithinDuration(t, publishedAt, *got.PublishedAt, time.Millisecond)
	assert.Nil(t, got.DeletedAt)
}

func TestPostRepo_Create_DraftHasNoPublishedAt(t *testing.T) {
	repo, _ := setupPostRepo(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, newTestPost("a-draft"))
	require.NoError(t, err)

	assert.Equal(t, model.PostStatusDraft, created.Status)
	assert.Nil(t, created.PublishedAt)
	assert.Empty(t, created.Excerpt)
	assert.Empty(t, created.CoverImageURL)
}

func TestPostRepo_Create_DuplicateSlug(t *testing.T) {
	repo, tdb := setupPostRepo(t)
	ctx := context.Background()

	_, err := repo.Create(ctx, newTestPost("taken"))
	require.NoError(t, err)

	created, err := repo.Create(ctx, newTestPost("taken"))
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrSlugTaken), "want ErrSlugTaken, got %v", err)
	assert.Nil(t, created)
	assert.Equal(t, 1, countPosts(t, tdb), "the losing insert must not leave a row")
}

func TestPostRepo_Create_OtherConstraintFailuresAreNotSlugTaken(t *testing.T) {
	repo, tdb := setupPostRepo(t)
	ctx := context.Background()

	badStatus := newTestPost("bad-status")
	badStatus.Status = "archived"
	_, err := repo.Create(ctx, badStatus)
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrSlugTaken), "CHECK violation must not look like a slug collision")

	tooLong := newTestPost("too-long")
	tooLong.Title = strings.Repeat("a", 201)
	_, err = repo.Create(ctx, tooLong)
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrSlugTaken), "value-too-long must not look like a slug collision")

	assert.Equal(t, 0, countPosts(t, tdb))
}

func TestPostRepo_Create_ConcurrentSameSlug(t *testing.T) {
	repo, tdb := setupPostRepo(t)
	ctx := context.Background()

	const workers = 20
	results := make([]error, workers)
	start := make(chan struct{})
	var wg sync.WaitGroup

	for i := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, results[i] = repo.Create(ctx, newTestPost("same-slug"))
		}()
	}
	close(start)
	wg.Wait()

	var succeeded, taken, other int
	for _, err := range results {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrSlugTaken):
			taken++
		default:
			other++
			t.Errorf("unexpected error: %v", err)
		}
	}
	assert.Equal(t, 1, succeeded, "exactly one insert may win")
	assert.Equal(t, workers-1, taken, "every loser must get ErrSlugTaken")
	assert.Zero(t, other)
	assert.Equal(t, 1, countPosts(t, tdb))
}

func TestPostRepo_GetByID_NotFound(t *testing.T) {
	repo, _ := setupPostRepo(t)

	got, err := repo.GetByID(context.Background(), 999999)
	require.NoError(t, err, "a missing row is not an error")
	assert.Nil(t, got)
}

func TestPostRepo_SoftDeleted(t *testing.T) {
	repo, tdb := setupPostRepo(t)
	ctx := context.Background()

	created, err := repo.Create(ctx, newTestPost("to-delete"))
	require.NoError(t, err)

	// Soft delete directly: the repository has no delete method until a later ticket.
	_, err = tdb.Pool.Exec(ctx, "UPDATE posts SET deleted_at = NOW() WHERE id = $1", created.ID)
	require.NoError(t, err)

	got, err := repo.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Nil(t, got, "soft-deleted post must not be returned")

	// The row is still there, so its slug stays reserved.
	assert.Equal(t, 1, countPosts(t, tdb))
	_, err = repo.Create(ctx, newTestPost("to-delete"))
	assert.True(t, errors.Is(err, ErrSlugTaken), "deleted post's slug must never be reused, got %v", err)
}

// publishedPost returns a published post fixture with the given slug and published_at.
func publishedPost(slug string, publishedAt time.Time) model.Post {
	p := newTestPost(slug)
	p.Status = model.PostStatusPublished
	p.PublishedAt = &publishedAt
	return p
}

func slugsOf(items []model.PostSummary) []string {
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Slug)
	}
	return out
}

func TestPostRepo_ListPublished_OnlyPublishedRowsAppear(t *testing.T) {
	repo, tdb := setupPostRepo(t)
	ctx := context.Background()
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)

	visible, err := repo.Create(ctx, publishedPost("visible", base))
	require.NoError(t, err)
	_, err = repo.Create(ctx, newTestPost("plain-draft"))
	require.NoError(t, err)

	deleted, err := repo.Create(ctx, publishedPost("soft-deleted", base.Add(time.Minute)))
	require.NoError(t, err)
	_, err = tdb.Pool.Exec(ctx, "UPDATE posts SET deleted_at = NOW() WHERE id = $1", deleted.ID)
	require.NoError(t, err)

	unpublished, err := repo.Create(ctx, publishedPost("unpublished-again", base.Add(2*time.Minute)))
	require.NoError(t, err)
	_, err = tdb.Pool.Exec(ctx, "UPDATE posts SET status = 'draft', published_at = NULL WHERE id = $1", unpublished.ID)
	require.NoError(t, err)

	items, total, err := repo.ListPublished(ctx, 1, 20)
	require.NoError(t, err)

	assert.Equal(t, []string{"visible"}, slugsOf(items))
	assert.Equal(t, int64(1), total, "drafts, soft-deleted and unpublished rows must not be counted")
	assert.Equal(t, 4, countPosts(t, tdb), "the excluded rows still exist")

	got := items[0]
	assert.Equal(t, visible.Title, got.Title)
	assert.Equal(t, visible.Excerpt, got.Excerpt)
	assert.Equal(t, visible.CoverImageURL, got.CoverImageURL)
	require.NotNil(t, got.PublishedAt)
	assert.True(t, base.Equal(*got.PublishedAt))
}

func TestPostRepo_ListPublished_OrdersByPublishedAtThenIDDescending(t *testing.T) {
	repo, _ := setupPostRepo(t)
	ctx := context.Background()
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)

	// Insertion order differs from published_at order; "tie-a" and "tie-b" share a timestamp.
	for _, p := range []model.Post{
		publishedPost("oldest", base),
		publishedPost("newest", base.Add(2*time.Hour)),
		publishedPost("tie-a", base.Add(time.Hour)),
		publishedPost("tie-b", base.Add(time.Hour)),
	} {
		_, err := repo.Create(ctx, p)
		require.NoError(t, err)
	}

	items, total, err := repo.ListPublished(ctx, 1, 20)
	require.NoError(t, err)

	assert.Equal(t, int64(4), total)
	assert.Equal(t, []string{"newest", "tie-b", "tie-a", "oldest"}, slugsOf(items), "published_at DESC, id DESC")
}

func TestPostRepo_ListPublished_Pagination(t *testing.T) {
	repo, _ := setupPostRepo(t)
	ctx := context.Background()
	base := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)

	for i, slug := range []string{"p1", "p2", "p3", "p4", "p5"} {
		_, err := repo.Create(ctx, publishedPost(slug, base.Add(time.Duration(i)*time.Minute)))
		require.NoError(t, err)
	}

	page1, total, err := repo.ListPublished(ctx, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total, "total counts all matching rows, not just the page")
	assert.Equal(t, []string{"p5", "p4"}, slugsOf(page1))

	page3, total, err := repo.ListPublished(ctx, 3, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total)
	assert.Equal(t, []string{"p1"}, slugsOf(page3))

	past, total, err := repo.ListPublished(ctx, 4, 2)
	require.NoError(t, err)
	assert.Equal(t, int64(5), total, "total is still reported past the last page")
	assert.NotNil(t, past, "a page past the end is an empty slice, not nil")
	assert.Empty(t, past)
}

func TestPostRepo_ListPublished_EmptyTable(t *testing.T) {
	repo, _ := setupPostRepo(t)

	items, total, err := repo.ListPublished(context.Background(), 1, 20)
	require.NoError(t, err)
	assert.Equal(t, int64(0), total)
	assert.NotNil(t, items)
	assert.Empty(t, items)
}

func TestPostRepo_GetPublishedBySlug_Published(t *testing.T) {
	repo, _ := setupPostRepo(t)
	ctx := context.Background()

	in := publishedPost("hello-world", time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond))
	in.Content = "full **markdown** body"
	created, err := repo.Create(ctx, in)
	require.NoError(t, err)

	got, err := repo.GetPublishedBySlug(ctx, "hello-world")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "full **markdown** body", got.Content)
	assert.Equal(t, model.PostStatusPublished, got.Status)
}

func TestPostRepo_GetPublishedBySlug_DraftDeletedAndMissingAllReturnNil(t *testing.T) {
	repo, tdb := setupPostRepo(t)
	ctx := context.Background()
	publishedAt := time.Now().UTC().Add(-time.Hour)

	_, err := repo.Create(ctx, newTestPost("a-draft"))
	require.NoError(t, err)

	deleted, err := repo.Create(ctx, publishedPost("a-deleted", publishedAt))
	require.NoError(t, err)
	_, err = tdb.Pool.Exec(ctx, "UPDATE posts SET deleted_at = NOW() WHERE id = $1", deleted.ID)
	require.NoError(t, err)

	unpublished, err := repo.Create(ctx, publishedPost("a-unpublished", publishedAt))
	require.NoError(t, err)
	_, err = tdb.Pool.Exec(ctx, "UPDATE posts SET status = 'draft', published_at = NULL WHERE id = $1", unpublished.ID)
	require.NoError(t, err)

	for _, slug := range []string{"a-draft", "a-deleted", "a-unpublished", "never-existed"} {
		got, err := repo.GetPublishedBySlug(ctx, slug)
		require.NoError(t, err, "%s: not-found is not an error", slug)
		assert.Nil(t, got, "%s must be indistinguishable from a missing slug", slug)
	}
}
