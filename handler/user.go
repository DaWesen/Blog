package handler

import (
	userservice "blog/service/UserService"

	"blog/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

// ErrorResponse 错误响应结构体
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

// LoginResponse 登录响应结构体
type LoginResponse struct {
	Token string                    `json:"token"`
	User  *userservice.UserResponse `json:"user"`
}

// CheckExistsResponse 检查是否存在响应结构体
type CheckExistsResponse struct {
	Exists bool `json:"exists"`
}

// UserHandler 用户处理器
type UserHandler struct {
	userService userservice.UserService
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userService userservice.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

// Register 用户注册
func (h *UserHandler) Register(c *gin.Context) {
	var req userservice.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "请求参数错误", Details: err.Error()})
		return
	}

	resp, err := h.userService.Register(c.Request.Context(), &req)
	if err != nil {
		status := http.StatusBadRequest
		if err == userservice.ErrEmailExists || err == userservice.ErrUsernameExists {
			status = http.StatusConflict
		}
		c.JSON(status, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusCreated, resp)
}

// Login 用户登录
func (h *UserHandler) Login(c *gin.Context) {
	var req userservice.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "请求参数错误", Details: err.Error()})
		return
	}

	resp, err := h.userService.Login(c.Request.Context(), &req)
	if err != nil {
		status := http.StatusUnauthorized
		if err == userservice.ErrInvalidCredentials {
			status = http.StatusUnauthorized
		}
		c.JSON(status, ErrorResponse{Error: err.Error()})
		return
	}

	// 生成JWT Token
	token, err := utils.GenerateToken(resp.ID, resp.Name, string(resp.Relation))
	if err != nil {
		slog.Error("生成token失败", "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "系统错误"})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token: token,
		User:  resp,
	})
}

// GetProfile 获取当前用户资料
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, err := utils.GetUserIDFromGin(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: err.Error()})
		return
	}

	resp, err := h.userService.GetUserProfile(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// UpdateProfile 更新个人资料
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, err := utils.GetUserIDFromGin(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: err.Error()})
		return
	}

	var req userservice.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "请求参数错误", Details: err.Error()})
		return
	}

	resp, err := h.userService.UpdateProfile(c.Request.Context(), userID, &req)
	if err != nil {
		status := http.StatusBadRequest
		if err == userservice.ErrUsernameExists {
			status = http.StatusConflict
		}
		c.JSON(status, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetUserPublicProfile 获取用户公开资料
func (h *UserHandler) GetUserPublicProfile(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "用户名不能为空"})
		return
	}

	resp, err := h.userService.GetUserPublicProfile(c.Request.Context(), username)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// CheckUsernameExists 检查用户名是否存在
func (h *UserHandler) CheckUsernameExists(c *gin.Context) {
	username := c.Query("username")
	if username == "" {
		c.JSON(http.StatusOK, CheckExistsResponse{Exists: false})
		return
	}

	exists, err := h.userService.CheckUsernameExists(c.Request.Context(), username)
	if err != nil {
		slog.Error("检查用户名失败", "error", err)
		c.JSON(http.StatusOK, CheckExistsResponse{Exists: false})
		return
	}

	c.JSON(http.StatusOK, CheckExistsResponse{Exists: exists})
}

// CheckEmailExists 检查邮箱是否存在
func (h *UserHandler) CheckEmailExists(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		c.JSON(http.StatusOK, CheckExistsResponse{Exists: false})
		return
	}

	exists, err := h.userService.CheckEmailExists(c.Request.Context(), email)
	if err != nil {
		slog.Error("检查邮箱失败", "error", err)
		c.JSON(http.StatusOK, CheckExistsResponse{Exists: false})
		return
	}

	c.JSON(http.StatusOK, CheckExistsResponse{Exists: exists})
}
