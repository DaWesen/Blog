package handler

import (
	"blog/model"
	postservice "blog/service/PostService"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

// PostHandler 文章处理器
type PostHandler struct {
	postService postservice.PostService
}

// NewPostHandler 创建文章处理器
func NewPostHandler(postService postservice.PostService) *PostHandler {
	return &PostHandler{postService: postService}
}

// ListPostsResponse 文章列表响应结构体
type ListPostsResponse struct {
	Posts []*model.Post `json:"posts"`
	Total int64         `json:"total"`
	Page  int           `json:"page"`
	Size  int           `json:"size"`
}

// CreatePost 创建文章
func (h *PostHandler) CreatePost(c *gin.Context) {
	var req postservice.CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "请求参数错误", Details: err.Error()})
		return
	}

	post, err := h.postService.CreatePost(c.Request.Context(), &req)
	if err != nil {
		status := http.StatusBadRequest
		switch err {
		case postservice.ErrPostSlugExists:
			status = http.StatusConflict
		case postservice.ErrUnauthorized:
			status = http.StatusUnauthorized
		}
		c.JSON(status, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, post)
}

// GetPost 获取文章详情
func (h *PostHandler) GetPost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的文章ID"})
		return
	}

	post, err := h.postService.GetPost(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, post)
}

// GetPostBySlug 通过Slug获取文章
func (h *PostHandler) GetPostBySlug(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "文章别名不能为空"})
		return
	}

	post, err := h.postService.GetPostBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, post)
}

// UpdatePost 更新文章
func (h *PostHandler) UpdatePost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的文章ID"})
		return
	}

	var req postservice.UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "请求参数错误", Details: err.Error()})
		return
	}

	post, err := h.postService.UpdatePost(c.Request.Context(), uint(id), &req)
	if err != nil {
		status := http.StatusBadRequest
		switch err {
		case postservice.ErrPostNotFound:
			status = http.StatusNotFound
		case postservice.ErrUnauthorized:
			status = http.StatusUnauthorized
		default:
			if err.Error() == "没有权限修改此帖子" {
				status = http.StatusForbidden
			}
		}
		c.JSON(status, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, post)
}

// DeletePost 删除文章
func (h *PostHandler) DeletePost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的文章ID"})
		return
	}

	err = h.postService.DeletePost(c.Request.Context(), uint(id))
	if err != nil {
		status := http.StatusBadRequest
		switch err {
		case postservice.ErrPostNotFound:
			status = http.StatusNotFound
		case postservice.ErrUnauthorized:
			status = http.StatusUnauthorized
		default:
			if err.Error() == "没有权限删除此帖子" {
				status = http.StatusForbidden
			}
		}
		c.JSON(status, ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListPosts 分页列出文章
func (h *PostHandler) ListPosts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	posts, total, err := h.postService.ListPosts(c.Request.Context(), page, size)
	if err != nil {
		slog.Error("获取文章列表失败", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "获取文章列表失败"})
		return
	}

	c.JSON(http.StatusOK, ListPostsResponse{
		Posts: posts,
		Total: total,
		Page:  page,
		Size:  size,
	})
}

// ListPostsByCategory 按分类列出文章
func (h *PostHandler) ListPostsByCategory(c *gin.Context) {
	categoryIDStr := c.Param("category_id")
	categoryID, err := strconv.ParseUint(categoryIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的分类ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	posts, total, err := h.postService.ListPostsByCategory(c.Request.Context(), uint(categoryID), page, size)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "分类不存在"})
		return
	}

	c.JSON(http.StatusOK, ListPostsResponse{
		Posts: posts,
		Total: total,
		Page:  page,
		Size:  size,
	})
}

// ListPostsByTag 按标签列出文章
func (h *PostHandler) ListPostsByTag(c *gin.Context) {
	tagIDStr := c.Param("tag_id")
	tagID, err := strconv.ParseUint(tagIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的标签ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	posts, total, err := h.postService.ListPostsByTag(c.Request.Context(), uint(tagID), page, size)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "标签不存在"})
		return
	}

	c.JSON(http.StatusOK, ListPostsResponse{
		Posts: posts,
		Total: total,
		Page:  page,
		Size:  size,
	})
}

// SearchPosts 搜索文章
func (h *PostHandler) SearchPosts(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		h.ListPosts(c)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	posts, total, err := h.postService.SearchPosts(c.Request.Context(), keyword, page, size)
	if err != nil {
		slog.Error("搜索文章失败", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "搜索文章失败"})
		return
	}

	c.JSON(http.StatusOK, ListPostsResponse{
		Posts: posts,
		Total: total,
		Page:  page,
		Size:  size,
	})
}

// LikePost 点赞文章
func (h *PostHandler) LikePost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的文章ID"})
		return
	}

	err = h.postService.LikePost(c.Request.Context(), uint(id))
	if err != nil {
		status := http.StatusBadRequest
		switch err {
		case postservice.ErrPostNotFound:
			status = http.StatusNotFound
		case postservice.ErrUnauthorized:
			status = http.StatusUnauthorized
		case postservice.ErrPostAlreadyLiked:
			status = http.StatusConflict
		}
		c.JSON(status, ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// UnlikePost 取消点赞
func (h *PostHandler) UnlikePost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的文章ID"})
		return
	}

	err = h.postService.UnlikePost(c.Request.Context(), uint(id))
	if err != nil {
		status := http.StatusBadRequest
		switch err {
		case postservice.ErrPostNotFound:
			status = http.StatusNotFound
		case postservice.ErrUnauthorized:
			status = http.StatusUnauthorized
		case postservice.ErrPostNotLiked:
			status = http.StatusConflict
		}
		c.JSON(status, ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// StarPost 收藏文章
func (h *PostHandler) StarPost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的文章ID"})
		return
	}

	err = h.postService.StarPost(c.Request.Context(), uint(id))
	if err != nil {
		status := http.StatusBadRequest
		switch err {
		case postservice.ErrPostNotFound:
			status = http.StatusNotFound
		case postservice.ErrUnauthorized:
			status = http.StatusUnauthorized
		case postservice.ErrPostAlreadyStarred:
			status = http.StatusConflict
		}
		c.JSON(status, ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// UnstarPost 取消收藏
func (h *PostHandler) UnstarPost(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的文章ID"})
		return
	}

	err = h.postService.UnstarPost(c.Request.Context(), uint(id))
	if err != nil {
		status := http.StatusBadRequest
		switch err {
		case postservice.ErrPostNotFound:
			status = http.StatusNotFound
		case postservice.ErrUnauthorized:
			status = http.StatusUnauthorized
		case postservice.ErrPostNotStarred:
			status = http.StatusConflict
		}
		c.JSON(status, ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetPostStats 获取文章统计信息
func (h *PostHandler) GetPostStats(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的文章ID"})
		return
	}

	stats, err := h.postService.GetPostStats(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
