package handler

import (
	feedservice "blog/service/FeedService"
	"blog/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

// FeedHandler 动态流处理器
type FeedHandler struct {
	feedService feedservice.FeedService
}

// NewFeedHandler 创建动态流处理器
func NewFeedHandler(feedService feedservice.FeedService) *FeedHandler {
	return &FeedHandler{feedService: feedService}
}

// GetUserFeed 获取用户动态流
func (h *FeedHandler) GetUserFeed(c *gin.Context) {
	userIDStr := c.Param("user_id")
	var userID uint

	if userIDStr == "" || userIDStr == "me" {
		// 获取当前用户ID
		currentUserID, err := utils.GetUserIDFromGin(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "用户未认证"})
			return
		}
		userID = currentUserID
	} else {
		parsedID, err := strconv.ParseUint(userIDStr, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的用户ID"})
			return
		}
		userID = uint(parsedID)
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	posts, total, err := h.feedService.GetUserFeed(c.Request.Context(), userID, page, size)
	if err != nil {
		slog.Error("获取用户动态流失败", "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"posts": posts,
		"total": total,
		"page":  page,
		"size":  size,
	})
}

// GetHotFeed 获取热门动态
func (h *FeedHandler) GetHotFeed(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	posts, total, err := h.feedService.GetHotPosts(c.Request.Context(), page, size)
	if err != nil {
		slog.Error("获取热门动态失败", "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"posts": posts,
		"total": total,
		"page":  page,
		"size":  size,
	})
}
