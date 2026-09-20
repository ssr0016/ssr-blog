package repository

import (
	"context"
	"time"

	"github.com/ssr0016/ssr-blog/internal/model"
)

// UserRepository defines the interface for user data access.
type UserRepository interface {
	GetByID(ctx context.Context, id int64) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	CreateWithPassword(ctx context.Context, email, name, hash string) (*model.User, error)
	List(ctx context.Context, emailFilter string, limit int) ([]model.User, error)
	ListWithPagination(ctx context.Context, emailFilter string, page, limit int) ([]model.User, int64, error)
	MarkEmailVerified(ctx context.Context, userID int64) error
	UpdatePassword(ctx context.Context, userID int64, hash string) error
	RecordFailedLogin(ctx context.Context, userID int64, maxAttempts int, lockDuration time.Duration) error
	ResetLoginAttempts(ctx context.Context, userID int64) error
	UpdateRole(ctx context.Context, userID, roleID int64) error
	GetWithRole(ctx context.Context, id int64) (*model.User, error)
}

// RoleRepository defines the interface for role data access.
type RoleRepository interface {
	Create(ctx context.Context, req model.CreateRoleRequest) (*model.Role, error)
	GetByID(ctx context.Context, id int64) (*model.Role, error)
	GetByName(ctx context.Context, name string) (*model.Role, error)
	List(ctx context.Context) ([]model.Role, error)
	ListWithPagination(ctx context.Context, page, limit int) ([]model.Role, int64, error)
	Update(ctx context.Context, id int64, req model.UpdateRoleRequest) (*model.Role, error)
	Delete(ctx context.Context, id int64) error
	GetPermissions(ctx context.Context, roleID int64) ([]model.Permission, error)
	AssignPermission(ctx context.Context, roleID, permissionID int64) error
	RevokePermission(ctx context.Context, roleID, permissionID int64) error
}

// PermissionRepository defines the interface for permission data access.
type PermissionRepository interface {
	Create(ctx context.Context, name, resource, action string) (*model.Permission, error)
	GetByID(ctx context.Context, id int64) (*model.Permission, error)
	GetByName(ctx context.Context, name string) (*model.Permission, error)
	List(ctx context.Context) ([]model.Permission, error)
	ListWithPagination(ctx context.Context, resource string, page, limit int) ([]model.Permission, int64, error)
	ListByResource(ctx context.Context, resource string) ([]model.Permission, error)
	Delete(ctx context.Context, id int64) error
}

// VerificationRepository defines the interface for verification token access.
type VerificationRepository interface {
	Create(ctx context.Context, userID int64) (*model.VerificationToken, error)
	GetByToken(ctx context.Context, token string) (*model.VerificationToken, error)
	MarkUsed(ctx context.Context, id int64) error
	DeleteExpired(ctx context.Context) (int64, error)
}

// PasswordResetRepository defines the interface for password reset token access.
type PasswordResetRepository interface {
	Create(ctx context.Context, userID int64) (*model.PasswordReset, error)
	GetByToken(ctx context.Context, token string) (*model.PasswordReset, error)
	MarkUsed(ctx context.Context, id int64) error
	DeleteExpired(ctx context.Context) (int64, error)
}

// PostRepository defines the interface for blog post data access.
// Every method excludes soft-deleted posts.
type PostRepository interface {
	// Create inserts a post. A slug collision returns ErrSlugTaken.
	Create(ctx context.Context, p model.Post) (*model.Post, error)
	// GetByID returns a non-deleted post of any status, or nil, nil if there is none.
	GetByID(ctx context.Context, id int64) (*model.Post, error)
	// ListPublished returns one page of public summaries (published, not deleted), ordered by
	// published_at DESC, id DESC, plus the total count of public posts.
	ListPublished(ctx context.Context, page, limit int) ([]model.PostSummary, int64, error)
	// GetPublishedBySlug returns a published, non-deleted post, or nil, nil if there is none.
	// Drafts and deleted posts are indistinguishable from a slug that never existed.
	GetPublishedBySlug(ctx context.Context, slug string) (*model.Post, error)
	// ListAdmin returns one page of non-deleted posts of any status, ordered by created_at DESC,
	// id DESC, plus the total for the same filter. An empty status means all statuses.
	ListAdmin(ctx context.Context, status string, page, limit int) ([]model.AdminPostSummary, int64, error)
}

// Compile-time checks: implementations must satisfy interfaces.
var (
	_ UserRepository          = (*UserRepo)(nil)
	_ RoleRepository          = (*RoleRepo)(nil)
	_ PermissionRepository    = (*PermissionRepo)(nil)
	_ VerificationRepository  = (*VerificationRepo)(nil)
	_ PasswordResetRepository = (*PasswordResetRepo)(nil)
	_ PostRepository          = (*PostRepo)(nil)
	_ PostRepository          = (*MockPostRepo)(nil)
)
