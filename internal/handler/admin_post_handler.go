package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/alexedwards/scs/v2"
	"github.com/labstack/echo/v4"

	"github.com/ssr0016/ssr-blog/internal/apperror"
	"github.com/ssr0016/ssr-blog/internal/middleware"
	"github.com/ssr0016/ssr-blog/internal/model"
	"github.com/ssr0016/ssr-blog/internal/service"
)

// AdminPostHandler serves the admin-only blog post endpoints. Every route is mounted behind
// RequireAuth and RequireRole("admin") in the router.
type AdminPostHandler struct {
	postService *service.PostService
	sm          *scs.SessionManager
}

func NewAdminPostHandler(postService *service.PostService, sm *scs.SessionManager) *AdminPostHandler {
	return &AdminPostHandler{postService: postService, sm: sm}
}

// Create godoc
// @Summary      Create a blog post
// @Description  Admin only. The slug is generated from the title and never changes afterwards. Status defaults to draft. Content is stored exactly as written; it is not rendered or sanitized.
// @Tags         admin/posts
// @Accept       json
// @Produce      json
// @Security     CookieAuth
// @Param        body body model.CreatePostRequest true "Post payload"
// @Success      201 {object} model.PostResponse
// @Failure      400 {object} apperror.ErrorResponse
// @Failure      401 {object} apperror.ErrorResponse
// @Failure      403 {object} apperror.ErrorResponse
// @Failure      422 {object} apperror.ErrorResponse
// @Failure      429 {object} apperror.ErrorResponse
// @Router       /admin/posts [post]
func (h *AdminPostHandler) Create(c echo.Context) error {
	var req model.CreatePostRequest
	if err := c.Bind(&req); err != nil {
		return apperror.BadRequest("invalid request body").WithError(err)
	}
	if err := c.Validate(&req); err != nil {
		return apperror.Validation(err.Error())
	}

	ctx := c.Request().Context()
	created, err := h.postService.Create(ctx, req)
	if err != nil {
		return err
	}

	// Audit trail: who did what. Deliberately carries no post content.
	slog.InfoContext(ctx, "audit",
		"audit", true,
		"actor_user_id", h.sm.GetInt64(ctx, middleware.UserIDKey),
		"action", "post.create",
		"post_id", created.ID,
		"slug", created.Slug,
	)

	return c.JSON(http.StatusCreated, created.ToResponse())
}

// Get godoc
// @Summary      Get a blog post by ID
// @Description  Admin only. Returns a post of any status, including drafts. Soft-deleted posts are not found.
// @Tags         admin/posts
// @Produce      json
// @Security     CookieAuth
// @Param        id path int true "Post ID"
// @Success      200 {object} model.PostResponse
// @Failure      400 {object} apperror.ErrorResponse
// @Failure      401 {object} apperror.ErrorResponse
// @Failure      403 {object} apperror.ErrorResponse
// @Failure      404 {object} apperror.ErrorResponse
// @Router       /admin/posts/{id} [get]
func (h *AdminPostHandler) Get(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id < 1 {
		return apperror.BadRequest("invalid post id")
	}

	post, err := h.postService.GetByID(c.Request().Context(), id)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, post.ToResponse())
}
