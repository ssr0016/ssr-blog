package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/ssr0016/ssr-blog/internal/apperror"
	"github.com/ssr0016/ssr-blog/internal/model"
	"github.com/ssr0016/ssr-blog/internal/service"
	"github.com/ssr0016/ssr-blog/pkg/pagination"
)

// Swag resolves @Failure types through the file's imports, and this file returns errors from the
// service without naming apperror, so the import is pinned here for the annotations below.
var _ apperror.ErrorResponse

// PostHandler serves the public, read-only blog endpoints. It has no session or auth dependency:
// the routes are mounted without auth middleware and the handler never looks at credentials.
type PostHandler struct {
	postService *service.PostService
}

func NewPostHandler(postService *service.PostService) *PostHandler {
	return &PostHandler{postService: postService}
}

// List godoc
// @Summary      List published blog posts
// @Description  Public. Returns summaries of published posts, newest first. Full content is not included; fetch a single post by slug for that. Out-of-range page and limit values are clamped.
// @Tags         posts
// @Produce      json
// @Param        page query int false "Page number (default 1)"
// @Param        limit query int false "Items per page (default 20, max 100)"
// @Success      200 {object} pagination.Response[model.PostSummary]
// @Failure      500 {object} apperror.ErrorResponse
// @Router       /posts [get]
func (h *PostHandler) List(c echo.Context) error {
	params := pagination.FromContext(c)

	items, total, err := h.postService.ListPublished(c.Request().Context(), params)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, pagination.NewResponse[model.PostSummary](items, params, total))
}

// Get godoc
// @Summary      Get a published blog post by slug
// @Description  Public. A draft, a deleted post and a slug that does not exist all return the same 404 response.
// @Tags         posts
// @Produce      json
// @Param        slug path string true "Post slug"
// @Success      200 {object} model.PostResponse
// @Failure      404 {object} apperror.ErrorResponse
// @Failure      500 {object} apperror.ErrorResponse
// @Router       /posts/{slug} [get]
func (h *PostHandler) Get(c echo.Context) error {
	post, err := h.postService.GetPublishedBySlug(c.Request().Context(), c.Param("slug"))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, post.ToResponse())
}
