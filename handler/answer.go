package handler

import (
	answerservice "blog/service/AnswerService"
	"blog/utils"
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

// AnswerHandler 回答处理器
type AnswerHandler struct {
	answerService answerservice.AnswerService
}

// NewAnswerHandler 创建回答处理器
func NewAnswerHandler(answerService answerservice.AnswerService) *AnswerHandler {
	return &AnswerHandler{answerService: answerService}
}

// CreateAnswer 创建回答
func (h *AnswerHandler) CreateAnswer(c *gin.Context) {
	var req answerservice.CreateAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "请求参数错误"})
		return
	}

	userID, err := utils.GetUserIDFromGin(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "用户未认证"})
		return
	}
	ctx := context.WithValue(c.Request.Context(), "user_id", userID)

	answer, err := h.answerService.CreateAnswer(ctx, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, answer)
}

// GetAnswer 获取回答详情
func (h *AnswerHandler) GetAnswer(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的回答ID"})
		return
	}

	answer, err := h.answerService.GetAnswer(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, answer)
}

// UpdateAnswer 更新回答
func (h *AnswerHandler) UpdateAnswer(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的回答ID"})
		return
	}

	var req answerservice.UpdateAnswerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "请求参数错误"})
		return
	}

	userID, err := utils.GetUserIDFromGin(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "用户未认证"})
		return
	}
	ctx := context.WithValue(c.Request.Context(), "user_id", userID)

	answer, err := h.answerService.UpdateAnswer(ctx, uint(id), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, answer)
}

// DeleteAnswer 删除回答
func (h *AnswerHandler) DeleteAnswer(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的回答ID"})
		return
	}

	userID, err := utils.GetUserIDFromGin(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "用户未认证"})
		return
	}
	ctx := context.WithValue(c.Request.Context(), "user_id", userID)

	err = h.answerService.DeleteAnswer(ctx, uint(id))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListAnswersByQuestion 获取问题回答列表
func (h *AnswerHandler) ListAnswersByQuestion(c *gin.Context) {
	questionIDStr := c.Param("question_id")
	questionID, err := strconv.ParseUint(questionIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的问题ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	order := c.DefaultQuery("order", "")

	answers, total, err := h.answerService.ListAnswersByQuestion(c.Request.Context(), uint(questionID), page, size, order)
	if err != nil {
		slog.Error("获取问题回答列表失败", "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"answers": answers,
		"total":   total,
		"page":    page,
		"size":    size,
	})
}

// ListAnswersByUser 获取用户回答列表
func (h *AnswerHandler) ListAnswersByUser(c *gin.Context) {
	userIDStr := c.Param("user_id")
	var userID uint

	if userIDStr == "" || userIDStr == "me" {
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

	answers, total, err := h.answerService.ListAnswersByUser(c.Request.Context(), userID, page, size)
	if err != nil {
		slog.Error("获取用户回答列表失败", "error", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"answers": answers,
		"total":   total,
		"page":    page,
		"size":    size,
	})
}

// AcceptAnswer 采纳回答
func (h *AnswerHandler) AcceptAnswer(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "无效的回答ID"})
		return
	}

	userID, err := utils.GetUserIDFromGin(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "用户未认证"})
		return
	}
	ctx := context.WithValue(c.Request.Context(), "user_id", userID)

	err = h.answerService.AcceptAnswer(ctx, uint(id))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
