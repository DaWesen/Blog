package handler

import (
	followservice "blog/service/FollowService"
	"blog/utils"
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

// FollowHandler 关注处理器
type FollowHandler struct {
	followService followservice.FollowService
}

// NewFollowHandler 创建关注处理器
func NewFollowHandler(followService followservice.FollowService) *FollowHandler {
	return &FollowHandler{followService: followService}
}

// FollowUser 关注用户
func (h *FollowHandler) FollowUser(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的用户ID"})
		return
	}

	currentUserID, err := utils.GetUserIDFromGin(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "用户未认证"})
		return
	}
	ctx := context.WithValue(c.Request.Context(), "user_id", currentUserID)

	err = h.followService.FollowUser(ctx, uint(userID))
	if err != nil {
		status := http.StatusBadRequest
		switch err {
		case followservice.ErrFollowYourself:
			status = http.StatusConflict
		case followservice.ErrAlreadyFollowing:
			status = http.StatusConflict
		case followservice.ErrUserNotFound:
			status = http.StatusNotFound
		case followservice.ErrRateLimited:
			status = http.StatusTooManyRequests
		}
		c.JSON(status, ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusCreated)
}

// UnfollowUser 取消关注
func (h *FollowHandler) UnfollowUser(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的用户ID"})
		return
	}

	currentUserID, err := utils.GetUserIDFromGin(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "用户未认证"})
		return
	}
	ctx := context.WithValue(c.Request.Context(), "user_id", currentUserID)

	err = h.followService.UnfollowUser(ctx, uint(userID))
	if err != nil {
		status := http.StatusBadRequest
		switch err {
		case followservice.ErrFollowYourself:
			status = http.StatusConflict
		case followservice.ErrNotFollowing:
			status = http.StatusConflict
		case followservice.ErrUserNotFound:
			status = http.StatusNotFound
		}
		c.JSON(status, ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// GetFollowingList 获取关注列表
func (h *FollowHandler) GetFollowingList(c *gin.Context) {
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

	users, total, err := h.followService.GetFollowingList(c.Request.Context(), userID, page, size)
	if err != nil {
		slog.Error("获取关注列表失败", "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"following": users,
		"total":     total,
		"page":      page,
		"size":      size,
	})
}

// GetFollowerList 获取粉丝列表
func (h *FollowHandler) GetFollowerList(c *gin.Context) {
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

	users, total, err := h.followService.GetFollowerList(c.Request.Context(), userID, page, size)
	if err != nil {
		slog.Error("获取粉丝列表失败", "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"followers": users,
		"total":     total,
		"page":      page,
		"size":      size,
	})
}

// GetFollowStats 获取关注统计
func (h *FollowHandler) GetFollowStats(c *gin.Context) {
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

	stats, err := h.followService.GetFollowStats(c.Request.Context(), userID)
	if err != nil {
		slog.Error("获取关注统计失败", "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// IsFollowing 检查是否关注
func (h *FollowHandler) IsFollowing(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的用户ID"})
		return
	}

	currentUserID, err := utils.GetUserIDFromGin(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "用户未认证"})
		return
	}
	ctx := context.WithValue(c.Request.Context(), "user_id", currentUserID)

	isFollowing, err := h.followService.IsFollowing(ctx, uint(userID))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"following": isFollowing,
		"user_id":   userID,
	})
}
