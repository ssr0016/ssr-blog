package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/ssr0016/ssr-blog/internal/model"
)

// MockUserRepo is an in-memory implementation of UserRepository for testing.
type MockUserRepo struct {
	mu     sync.Mutex
	users  map[int64]*model.User
	nextID int64

	// Override functions for testing specific behaviors
	GetByIDFunc            func(ctx context.Context, id int64) (*model.User, error)
	GetByEmailFunc         func(ctx context.Context, email string) (*model.User, error)
	CreateWithPasswordFunc func(ctx context.Context, email, name, hash string) (*model.User, error)
	ListFunc               func(ctx context.Context, emailFilter string, limit int) ([]model.User, error)
	UpdateRoleFunc         func(ctx context.Context, userID, roleID int64) error
	GetWithRoleFunc        func(ctx context.Context, id int64) (*model.User, error)
	ListWithPaginationFunc func(ctx context.Context, emailFilter string, page, limit int) ([]model.User, int64, error)
	MarkEmailVerifiedFunc  func(ctx context.Context, userID int64) error
	UpdatePasswordFunc     func(ctx context.Context, userID int64, hash string) error
	RecordFailedLoginFunc  func(ctx context.Context, userID int64, maxAttempts int, lockDuration time.Duration) error
	ResetLoginAttemptsFunc func(ctx context.Context, userID int64) error
}

// NewMockUserRepo creates a new mock repository.
func NewMockUserRepo() *MockUserRepo {
	return &MockUserRepo{
		users:  make(map[int64]*model.User),
		nextID: 1,
	}
}

func (m *MockUserRepo) GetByID(ctx context.Context, id int64) (*model.User, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	user, ok := m.users[id]
	if !ok {
		return nil, nil
	}
	return user, nil
}

func (m *MockUserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	if m.GetByEmailFunc != nil {
		return m.GetByEmailFunc(ctx, email)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, nil
}

func (m *MockUserRepo) CreateWithPassword(ctx context.Context, email, name, hash string) (*model.User, error) {
	if m.CreateWithPasswordFunc != nil {
		return m.CreateWithPasswordFunc(ctx, email, name, hash)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	user := &model.User{
		ID:           m.nextID,
		Email:        email,
		Name:         name,
		PasswordHash: hash,
	}
	m.users[m.nextID] = user
	m.nextID++
	return user, nil
}

func (m *MockUserRepo) List(ctx context.Context, emailFilter string, limit int) ([]model.User, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, emailFilter, limit)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]model.User, 0, len(m.users))
	for _, u := range m.users {
		result = append(result, *u)
	}
	return result, nil
}

// UpdateRole changes a user's role (mock).
func (m *MockUserRepo) UpdateRole(ctx context.Context, userID, roleID int64) error {
	if m.UpdateRoleFunc != nil {
		return m.UpdateRoleFunc(ctx, userID, roleID)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	user, ok := m.users[userID]
	if !ok {
		return fmt.Errorf("user not found")
	}
	user.RoleID = roleID
	return nil
}

// GetWithRole returns a user with role (mock).
func (m *MockUserRepo) GetWithRole(ctx context.Context, id int64) (*model.User, error) {
	if m.GetWithRoleFunc != nil {
		return m.GetWithRoleFunc(ctx, id)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	user, ok := m.users[id]
	if !ok {
		return nil, nil
	}
	return user, nil
}

// ListWithPagination returns users with pagination (mock).
func (m *MockUserRepo) ListWithPagination(ctx context.Context, emailFilter string, page, limit int) ([]model.User, int64, error) {
	if m.ListWithPaginationFunc != nil {
		return m.ListWithPaginationFunc(ctx, emailFilter, page, limit)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]model.User, 0, len(m.users))
	for _, u := range m.users {
		result = append(result, *u)
	}
	return result, int64(len(result)), nil
}

// MarkEmailVerified marks email as verified (mock).
func (m *MockUserRepo) MarkEmailVerified(ctx context.Context, userID int64) error {
	if m.MarkEmailVerifiedFunc != nil {
		return m.MarkEmailVerifiedFunc(ctx, userID)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	user, ok := m.users[userID]
	if !ok {
		return fmt.Errorf("user not found")
	}
	user.EmailVerified = true
	now := time.Now()
	user.EmailVerifiedAt = &now
	return nil
}

// UpdatePassword updates a user's password (mock).
func (m *MockUserRepo) UpdatePassword(ctx context.Context, userID int64, hash string) error {
	if m.UpdatePasswordFunc != nil {
		return m.UpdatePasswordFunc(ctx, userID, hash)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	user, ok := m.users[userID]
	if !ok {
		return fmt.Errorf("user not found")
	}
	user.PasswordHash = hash
	return nil
}

// RecordFailedLogin increments failed attempts (mock).
func (m *MockUserRepo) RecordFailedLogin(ctx context.Context, userID int64, maxAttempts int, lockDuration time.Duration) error {
	if m.RecordFailedLoginFunc != nil {
		return m.RecordFailedLoginFunc(ctx, userID, maxAttempts, lockDuration)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	user, ok := m.users[userID]
	if !ok {
		return fmt.Errorf("user not found")
	}
	user.FailedLoginAttempts++
	if user.FailedLoginAttempts >= maxAttempts {
		lockedUntil := time.Now().Add(lockDuration)
		user.LockedUntil = &lockedUntil
	}
	return nil
}

// ResetLoginAttempts clears failed attempts (mock).
func (m *MockUserRepo) ResetLoginAttempts(ctx context.Context, userID int64) error {
	if m.ResetLoginAttemptsFunc != nil {
		return m.ResetLoginAttemptsFunc(ctx, userID)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	user, ok := m.users[userID]
	if !ok {
		return fmt.Errorf("user not found")
	}
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil
	return nil
}

// MockPostRepo is an in-memory implementation of PostRepository for testing.
type MockPostRepo struct {
	mu     sync.Mutex
	posts  map[int64]*model.Post
	nextID int64

	// Override functions for testing specific behaviors
	CreateFunc             func(ctx context.Context, p model.Post) (*model.Post, error)
	GetByIDFunc            func(ctx context.Context, id int64) (*model.Post, error)
	ListPublishedFunc      func(ctx context.Context, page, limit int) ([]model.PostSummary, int64, error)
	GetPublishedBySlugFunc func(ctx context.Context, slug string) (*model.Post, error)
	ListAdminFunc          func(ctx context.Context, status string, page, limit int) ([]model.AdminPostSummary, int64, error)
	UpdateFunc             func(ctx context.Context, p model.Post) (*model.Post, error)
	SoftDeleteFunc         func(ctx context.Context, id int64) error
}

// NewMockPostRepo creates a new mock post repository.
func NewMockPostRepo() *MockPostRepo {
	return &MockPostRepo{
		posts:  make(map[int64]*model.Post),
		nextID: 1,
	}
}

// Create stores a post (mock). Like the real repository, a duplicate slug returns ErrSlugTaken.
func (m *MockPostRepo) Create(ctx context.Context, p model.Post) (*model.Post, error) {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, p)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, existing := range m.posts {
		if existing.Slug == p.Slug {
			return nil, fmt.Errorf("create post: %w", ErrSlugTaken)
		}
	}

	now := time.Now()
	p.ID = m.nextID
	p.CreatedAt = now
	p.UpdatedAt = now
	p.DeletedAt = nil
	stored := p
	m.posts[p.ID] = &stored
	m.nextID++

	created := stored
	return &created, nil
}

// GetByID returns a post (mock), or nil, nil if there is none.
func (m *MockPostRepo) GetByID(ctx context.Context, id int64) (*model.Post, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	post, ok := m.posts[id]
	if !ok || post.DeletedAt != nil {
		return nil, nil
	}
	found := *post
	return &found, nil
}

// ListPublished returns one page of public summaries (mock): published, not deleted, ordered by
// published_at DESC, id DESC.
func (m *MockPostRepo) ListPublished(ctx context.Context, page, limit int) ([]model.PostSummary, int64, error) {
	if m.ListPublishedFunc != nil {
		return m.ListPublishedFunc(ctx, page, limit)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var public []*model.Post
	for _, p := range m.posts {
		if isPublic(p) {
			public = append(public, p)
		}
	}
	sort.Slice(public, func(i, j int) bool {
		a, b := public[i], public[j]
		if !a.PublishedAt.Equal(*b.PublishedAt) {
			return a.PublishedAt.After(*b.PublishedAt)
		}
		return a.ID > b.ID
	})

	total := int64(len(public))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	items := []model.PostSummary{}
	for i := (page - 1) * limit; i < len(public) && i < page*limit; i++ {
		p := public[i]
		items = append(items, model.PostSummary{
			Title: p.Title, Slug: p.Slug, Excerpt: p.Excerpt, CoverImageURL: p.CoverImageURL, PublishedAt: p.PublishedAt,
		})
	}
	return items, total, nil
}

// GetPublishedBySlug returns a public post (mock), or nil, nil if there is none.
func (m *MockPostRepo) GetPublishedBySlug(ctx context.Context, slug string) (*model.Post, error) {
	if m.GetPublishedBySlugFunc != nil {
		return m.GetPublishedBySlugFunc(ctx, slug)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, p := range m.posts {
		if p.Slug == slug && isPublic(p) {
			found := *p
			return &found, nil
		}
	}
	return nil, nil
}

// ListAdmin returns one page of admin summaries (mock): non-deleted posts of any status, or of one
// status when given, ordered by created_at DESC, id DESC.
func (m *MockPostRepo) ListAdmin(ctx context.Context, status string, page, limit int) ([]model.AdminPostSummary, int64, error) {
	if m.ListAdminFunc != nil {
		return m.ListAdminFunc(ctx, status, page, limit)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var matched []*model.Post
	for _, p := range m.posts {
		if p.DeletedAt == nil && (status == "" || p.Status == status) {
			matched = append(matched, p)
		}
	}
	sort.Slice(matched, func(i, j int) bool {
		a, b := matched[i], matched[j]
		if !a.CreatedAt.Equal(b.CreatedAt) {
			return a.CreatedAt.After(b.CreatedAt)
		}
		return a.ID > b.ID
	})

	total := int64(len(matched))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}
	items := []model.AdminPostSummary{}
	for i := (page - 1) * limit; i < len(matched) && i < page*limit; i++ {
		p := matched[i]
		items = append(items, model.AdminPostSummary{
			ID: p.ID, Title: p.Title, Slug: p.Slug, Excerpt: p.Excerpt, CoverImageURL: p.CoverImageURL,
			Status: p.Status, PublishedAt: p.PublishedAt, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
		})
	}
	return items, total, nil
}

// Update replaces the editable fields of a stored post (mock). Like the real repository it never
// changes the slug, and a missing or soft-deleted id returns ErrPostNotFound.
func (m *MockPostRepo) Update(ctx context.Context, p model.Post) (*model.Post, error) {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, p)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	stored, ok := m.posts[p.ID]
	if !ok || stored.DeletedAt != nil {
		return nil, fmt.Errorf("update post: %w", ErrPostNotFound)
	}
	stored.Title = p.Title
	stored.Content = p.Content
	stored.Excerpt = p.Excerpt
	stored.CoverImageURL = p.CoverImageURL
	stored.Status = p.Status
	stored.PublishedAt = p.PublishedAt
	stored.UpdatedAt = time.Now()

	updated := *stored
	return &updated, nil
}

// SoftDelete marks a stored post as deleted (mock). Like the real repository it keeps the row, so the
// slug stays taken, and a missing or already soft-deleted id returns ErrPostNotFound.
func (m *MockPostRepo) SoftDelete(ctx context.Context, id int64) error {
	if m.SoftDeleteFunc != nil {
		return m.SoftDeleteFunc(ctx, id)
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	stored, ok := m.posts[id]
	if !ok || stored.DeletedAt != nil {
		return fmt.Errorf("soft delete post: %w", ErrPostNotFound)
	}
	now := time.Now()
	stored.DeletedAt = &now
	return nil
}

// IsSoftDeleted reports whether the post is still stored with deleted_at set. A hard delete (row
// removed) or a post that was never deleted both return false.
func (m *MockPostRepo) IsSoftDeleted(id int64) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.posts[id]
	return ok && p.DeletedAt != nil
}

// MarkDeleted soft-deletes a stored post (mock) without going through SoftDelete, for seeding tests.
func (m *MockPostRepo) MarkDeleted(id int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.posts[id]; ok {
		now := time.Now()
		p.DeletedAt = &now
	}
}

func isPublic(p *model.Post) bool {
	return p.Status == model.PostStatusPublished && p.DeletedAt == nil && p.PublishedAt != nil
}
