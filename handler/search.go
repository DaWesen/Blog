package handler

import (
	postservice "blog/service/PostService"
	userservice "blog/service/UserService"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

// SearchHandler 搜索处理器
type SearchHandler struct {
	postService postservice.PostService
	userService userservice.UserService
}

func NewSearchHandler(postService postservice.PostService, userService userservice.UserService) *SearchHandler {
	return &SearchHandler{
		postService: postService,
		userService: userService,
	}
}

// AdvancedSearch 高级搜索
func (h *SearchHandler) AdvancedSearch(c *gin.Context) {
	keyword := c.Query("keyword")
	categoryIDStr := c.Query("category_id")
	tagIDStr := c.Query("tag_id")
	order := c.Query("order") // "newest", "hot", "most_commented"

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	// 构建搜索参数
	params := make(map[string]interface{})

	if keyword != "" {
		params["keyword"] = keyword
	}

	if categoryIDStr != "" {
		categoryID, err := strconv.ParseUint(categoryIDStr, 10, 32)
		if err == nil {
			params["category_id"] = uint(categoryID)
		}
	}

	if tagIDStr != "" {
		tagID, err := strconv.ParseUint(tagIDStr, 10, 32)
		if err == nil {
			params["tag_id"] = uint(tagID)
		}
	}

	if order != "" {
		params["order"] = order
	}
	if keyword != "" {
		posts, total, err := h.postService.SearchPosts(c.Request.Context(), keyword, page, size)
		if err != nil {
			slog.Error("搜索文章失败", "error", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "搜索文章失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"posts":  posts,
			"total":  total,
			"page":   page,
			"size":   size,
			"params": params,
		})
	} else {
		// 如果没有关键词，就返回文章列表
		posts, total, err := h.postService.ListPosts(c.Request.Context(), page, size)
		if err != nil {
			slog.Error("获取文章列表失败", "error", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "获取文章列表失败"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"posts":  posts,
			"total":  total,
			"page":   page,
			"size":   size,
			"params": params,
		})
	}
}

// SearchPosts 搜索文章（兼容旧接口）
func (h *SearchHandler) SearchPosts(c *gin.Context) {
	h.AdvancedSearch(c)
}

// SearchUsers 搜索用户
func (h *SearchHandler) SearchUsers(c *gin.Context) {
	keyword := c.Query("keyword")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	// 验证参数
	if keyword == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "搜索关键词不能为空"})
		return
	}

	if page < 1 {
		page = 1
	}

	if size < 1 || size > 100 {
		size = 20
	}
	// 调用搜索方法
	users, total, err := h.userService.SearchUsers(c.Request.Context(), keyword, page, size)
	if err != nil {
		slog.Error("搜索用户失败",
			"keyword", keyword,
			"page", page,
			"size", size,
			"error", err)

		status := http.StatusInternalServerError
		errorMsg := "搜索用户失败"

		if err.Error() == "搜索关键词不能为空" {
			status = http.StatusBadRequest
		} else if err.Error() == "操作过于频繁，请稍后再试" {
			status = http.StatusTooManyRequests
		}

		c.JSON(status, ErrorResponse{Error: errorMsg})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"users":   users,
		"total":   total,
		"page":    page,
		"size":    size,
		"keyword": keyword,
	})
}
