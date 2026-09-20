//go:build integration

package service

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ssr0016/ssr-blog/internal/database"
	"github.com/ssr0016/ssr-blog/internal/model"
	"github.com/ssr0016/ssr-blog/internal/repository"
	"github.com/ssr0016/ssr-blog/internal/testutil"
)

func setupPostService(t *testing.T) (*PostService, *testutil.TestDB) {
	t.Helper()

	tdb := testutil.SetupPostgres(t)
	tdb.RunMigrations(t)
	tdb.TruncateAll(t)

	db := &database.DB{
		Pool:    tdb.Pool,
		Builder: database.NewStatementBuilder(),
	}
	return NewPostService(repository.NewPostRepo(db)), tdb
}

func TestPostService_Create_ConcurrentSameTitle(t *testing.T) {
	svc, tdb := setupPostService(t)
	ctx := context.Background()

	const workers = 20
	posts := make([]*model.Post, workers)
	errs := make([]error, workers)
	start := make(chan struct{})
	var wg sync.WaitGroup

	countBefore, sumBefore := slugAttemptTotals(t)

	for i := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			posts[i], errs[i] = svc.Create(ctx, model.CreatePostRequest{
				Title:   "Hello World",
				Content: "Same title, same moment",
			})
		}()
	}
	close(start)
	wg.Wait()

	slugs := make([]string, 0, workers)
	seen := make(map[string]bool, workers)
	for i := range workers {
		require.NoError(t, errs[i], "worker %d must not fail on a slug conflict", i)
		require.NotNil(t, posts[i])
		assert.False(t, seen[posts[i].Slug], "duplicate slug %q", posts[i].Slug)
		seen[posts[i].Slug] = true
		slugs = append(slugs, posts[i].Slug)
	}
	assert.Len(t, seen, workers, "want 20 distinct slugs")

	// A worker only moves to candidate n+1 after n was taken, so the winners must be
	// exactly hello-world, hello-world-2, ..., hello-world-20 with no gaps.
	want := []string{"hello-world"}
	for n := 2; n <= workers; n++ {
		want = append(want, fmt.Sprintf("hello-world-%d", n))
	}
	sort.Strings(slugs)
	sort.Strings(want)
	assert.Equal(t, want, slugs)

	var rows int
	require.NoError(t, tdb.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM posts").Scan(&rows))
	assert.Equal(t, workers, rows)

	// Every successful create is observed once; collisions push the sum above one attempt each.
	countAfter, sumAfter := slugAttemptTotals(t)
	assert.Equal(t, uint64(workers), countAfter-countBefore)
	assert.GreaterOrEqual(t, sumAfter-sumBefore, float64(workers))
}

func TestPostService_Create_DeletedPostSlugIsNeverReused(t *testing.T) {
	svc, tdb := setupPostService(t)
	ctx := context.Background()

	first, err := svc.Create(ctx, model.CreatePostRequest{Title: "Reserved", Content: "C"})
	require.NoError(t, err)
	assert.Equal(t, "reserved", first.Slug)

	// Soft delete directly: the service has no delete until a later ticket.
	_, err = tdb.Pool.Exec(ctx, "UPDATE posts SET deleted_at = NOW() WHERE id = $1", first.ID)
	require.NoError(t, err)

	_, err = svc.GetByID(ctx, first.ID)
	require.Error(t, err, "soft-deleted post must read as not found")

	second, err := svc.Create(ctx, model.CreatePostRequest{Title: "Reserved", Content: "C"})
	require.NoError(t, err)
	assert.Equal(t, "reserved-2", second.Slug)
}

func TestPostService_Create_PublishedRoundTrip(t *testing.T) {
	svc, _ := setupPostService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, model.CreatePostRequest{
		Title: "Go Live", Content: "C", Status: model.PostStatusPublished,
	})
	require.NoError(t, err)
	require.NotNil(t, created.PublishedAt)

	got, err := svc.GetByID(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, model.PostStatusPublished, got.Status)
	require.NotNil(t, got.PublishedAt)
	assert.WithinDuration(t, *created.PublishedAt, *got.PublishedAt, time.Millisecond)
}
