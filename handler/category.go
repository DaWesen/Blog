package handler

import (
	"blog/model"
	categoryservice "blog/service/CategoryService"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CategoryHandler 分类处理器
type CategoryHandler struct {
	categoryService categoryservice.CategoryService
}

// NewCategoryHandler 创建分类处理器
func NewCategoryHandler(categoryService categoryservice.CategoryService) *CategoryHandler {
	return &CategoryHandler{categoryService: categoryService}
}

// ListCategoriesResponse 分类列表响应结构体
type ListCategoriesResponse struct {
	Categories []*model.Category `json:"categories"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	Size       int               `json:"size"`
}

// CreateCategory 创建分类
func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var req categoryservice.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "请求参数错误", Details: err.Error()})
		return
	}

	category, err := h.categoryService.CreateCategory(c.Request.Context(), &req)
	if err != nil {
		status := http.StatusBadRequest
		if err == categoryservice.ErrCategoryExists {
			status = http.StatusConflict
		}
		c.JSON(status, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, category)
}

// GetCategory 获取分类详情
func (h *CategoryHandler) GetCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的分类ID"})
		return
	}

	category, err := h.categoryService.GetCategory(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, category)
}

// GetCategoryBySlug 通过Slug获取分类
func (h *CategoryHandler) GetCategoryBySlug(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "分类别名不能为空"})
		return
	}

	category, err := h.categoryService.GetCategoryBySlug(c.Request.Context(), slug)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, category)
}

// UpdateCategory 更新分类
func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的分类ID"})
		return
	}

	var req categoryservice.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "请求参数错误", Details: err.Error()})
		return
	}

	category, err := h.categoryService.UpdateCategory(c.Request.Context(), uint(id), &req)
	if err != nil {
		status := http.StatusBadRequest
		if err == categoryservice.ErrCategoryNotFound {
			status = http.StatusNotFound
		} else if err == categoryservice.ErrCategoryExists {
			status = http.StatusConflict
		}
		c.JSON(status, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, category)
}

// DeleteCategory 删除分类
func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的分类ID"})
		return
	}

	err = h.categoryService.DeleteCategory(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListCategories 分页列出分类
func (h *CategoryHandler) ListCategories(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	categories, total, err := h.categoryService.ListCategories(c.Request.Context(), page, size)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "获取分类列表失败"})
		return
	}

	c.JSON(http.StatusOK, ListCategoriesResponse{
		Categories: categories,
		Total:      total,
		Page:       page,
		Size:       size,
	})
}

// SearchCategories 搜索分类
func (h *CategoryHandler) SearchCategories(c *gin.Context) {
	keyword := c.Query("keyword")
	if keyword == "" {
		h.ListCategories(c)
		return
	}

	categories, err := h.categoryService.SearchCategories(c.Request.Context(), keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "搜索分类失败"})
		return
	}

	c.JSON(http.StatusOK, categories)
}
