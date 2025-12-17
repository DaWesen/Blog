package handler

import (
	"blog/model"
	commentservice "blog/service/CommentService"
	"blog/utils"
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

// CommentHandler 评论处理器
type CommentHandler struct {
	commentService commentservice.CommentService
}

// NewCommentHandler 创建评论处理器
func NewCommentHandler(commentService commentservice.CommentService) *CommentHandler {
	return &CommentHandler{commentService: commentService}
}

// ListCommentsResponse 评论列表响应结构体
type ListCommentsResponse struct {
	Comments []*model.Comment `json:"comments"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	Size     int              `json:"size"`
}

// LikesCountResponse 点赞数响应结构体
type LikesCountResponse struct {
	Count uint `json:"count"`
}

// IsLikedResponse 是否点赞响应结构体
type IsLikedResponse struct {
	Liked bool `json:"liked"`
}

// CreateComment 创建评论
func (h *CommentHandler) CreateComment(c *gin.Context) {
	var req commentservice.CreateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "请求参数错误", Details: err.Error()})
		return
	}
	currentUserID, err := utils.GetUserIDFromGin(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "用户未认证"})
		return
	}
	ctx := context.WithValue(c.Request.Context(), "user_id", currentUserID)

	comment, err := h.commentService.CreateComment(ctx, &req)
	if err != nil {
		status := http.StatusBadRequest
		if err == commentservice.ErrUnauthorized {
			status = http.StatusUnauthorized
		}
		c.JSON(status, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, comment)
}

// GetComment 获取评论详情
func (h *CommentHandler) GetComment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的评论ID"})
		return
	}

	comment, err := h.commentService.GetComment(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, comment)
}

// DeleteComment 删除评论
func (h *CommentHandler) DeleteComment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的评论ID"})
		return
	}
	currentUserID, err := utils.GetUserIDFromGin(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "用户未认证"})
		return
	}
	ctx := context.WithValue(c.Request.Context(), "user_id", currentUserID)

	err = h.commentService.DeleteComment(ctx, uint(id))
	if err != nil {
		status := http.StatusBadRequest
		if err == commentservice.ErrCommentNotFound {
			status = http.StatusNotFound
		} else if err == commentservice.ErrUnauthorized {
			status = http.StatusUnauthorized
		}
		c.JSON(status, ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListCommentsByPost 获取文章评论列表
func (h *CommentHandler) ListCommentsByPost(c *gin.Context) {
	postIDStr := c.Param("id")
	postID, err := strconv.ParseUint(postIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的文章ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	comments, total, err := h.commentService.ListCommentsByPost(c.Request.Context(), uint(postID), page, size)
	if err != nil {
		status := http.StatusInternalServerError
		errorMsg := "获取评论失败"

		// 根据错误类型返回不同的状态码和错误信息
		if err == commentservice.ErrPostIsDeleted {
			status = http.StatusNotFound
			errorMsg = "文章不存在或已被删除"
		} else if err == commentservice.ErrRateLimited {
			status = http.StatusTooManyRequests
			errorMsg = "请求过于频繁，请稍后再试"
		} else {
			// 记录服务器错误日志
			slog.Error("获取评论列表失败",
				"postID", postID,
				"page", page,
				"size", size,
				"error", err)
		}

		c.JSON(status, ErrorResponse{Error: errorMsg})
		return
	}

	c.JSON(http.StatusOK, ListCommentsResponse{
		Comments: comments,
		Total:    total,
		Page:     page,
		Size:     size,
	})
}

// ListCommentsByUser 获取用户评论列表
func (h *CommentHandler) ListCommentsByUser(c *gin.Context) {
	userIDStr := c.Param("user_id")
	var userID uint

	if userIDStr == "" || userIDStr == "me" {
		// 获取当前用户ID
		currentUserID, err := utils.GetUserIDFromGin(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: err.Error()})
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

	comments, total, err := h.commentService.ListCommentsByUser(c.Request.Context(), userID, page, size)
	if err != nil {
		slog.Error("获取用户评论列表失败", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "获取评论列表失败"})
		return
	}

	c.JSON(http.StatusOK, ListCommentsResponse{
		Comments: comments,
		Total:    total,
		Page:     page,
		Size:     size,
	})
}

// LikeComment 点赞评论
func (h *CommentHandler) LikeComment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的评论ID"})
		return
	}
	currentUserID, err := utils.GetUserIDFromGin(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "用户未认证"})
		return
	}
	ctx := context.WithValue(c.Request.Context(), "user_id", currentUserID)

	err = h.commentService.LikeComment(ctx, uint(id))
	if err != nil {
		status := http.StatusBadRequest
		if err == commentservice.ErrCommentNotFound {
			status = http.StatusNotFound
		} else if err == commentservice.ErrUnauthorized {
			status = http.StatusUnauthorized
		} else if err == commentservice.ErrCommentAlreadyLiked {
			status = http.StatusConflict
		}
		c.JSON(status, ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// UnlikeComment 取消点赞评论
func (h *CommentHandler) UnlikeComment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的评论ID"})
		return
	}
	currentUserID, err := utils.GetUserIDFromGin(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "用户未认证"})
		return
	}
	ctx := context.WithValue(c.Request.Context(), "user_id", currentUserID)

	err = h.commentService.UnlikeComment(ctx, uint(id))
	if err != nil {
		status := http.StatusBadRequest
		if err == commentservice.ErrCommentNotFound {
			status = http.StatusNotFound
		} else if err == commentservice.ErrUnauthorized {
			status = http.StatusUnauthorized
		} else if err == commentservice.ErrCommentNotLiked {
			status = http.StatusConflict
		}
		c.JSON(status, ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// CreateReply 创建回复
func (h *CommentHandler) CreateReply(c *gin.Context) {
	var req commentservice.CreateReplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "请求参数错误", Details: err.Error()})
		return
	}
	currentUserID, err := utils.GetUserIDFromGin(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "用户未认证"})
		return
	}
	ctx := context.WithValue(c.Request.Context(), "user_id", currentUserID)

	reply, err := h.commentService.CreateReply(ctx, &req)
	if err != nil {
		status := http.StatusBadRequest
		if err == commentservice.ErrUnauthorized {
			status = http.StatusUnauthorized
		}
		c.JSON(status, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, reply)
}

// ListReplies 获取评论回复列表
func (h *CommentHandler) ListReplies(c *gin.Context) {
	commentIDStr := c.Param("comment_id")
	commentID, err := strconv.ParseUint(commentIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的评论ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	replies, total, err := h.commentService.ListReplies(c.Request.Context(), uint(commentID), page, size)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "评论不存在"})
		return
	}

	c.JSON(http.StatusOK, ListCommentsResponse{
		Comments: replies,
		Total:    total,
		Page:     page,
		Size:     size,
	})
}

// GetCommentLikes 获取评论点赞数
func (h *CommentHandler) GetCommentLikes(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的评论ID"})
		return
	}

	count, err := h.commentService.GetCommentLikes(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, LikesCountResponse{Count: count})
}

// IsCommentLiked 检查是否点赞评论
func (h *CommentHandler) IsCommentLiked(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的评论ID"})
		return
	}

	isLiked, err := h.commentService.IsCommentLiked(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, IsLikedResponse{Liked: isLiked})
}
