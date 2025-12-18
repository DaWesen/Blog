package handler

import (
	userservice "blog/service/UserService"
	"context"
	"io"
	"strings"

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
	ctx := context.WithValue(c.Request.Context(), "user_id", userID)

	resp, err := h.userService.GetUserProfile(ctx, userID)
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
	ctx := context.WithValue(c.Request.Context(), "user_id", userID)

	resp, err := h.userService.UpdateProfile(ctx, userID, &req)
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

	// 特殊处理 "count" 请求
	if username == "count" {
		c.JSON(http.StatusOK, gin.H{
			"count":   0,
			"message": "用户统计接口已迁移到 /api/stats/users/count",
		})
		return
	}

	resp, err := h.userService.GetUserPublicProfile(c.Request.Context(), username)
	if err != nil {
		slog.Error("获取用户公开资料失败", "username", username, "error", err)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "用户不存在"})
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

// UploadAvatar 上传头像
func (h *UserHandler) UploadAvatar(c *gin.Context) {
	// 1. 获取当前用户ID
	userID, err := utils.GetUserIDFromGin(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "用户未认证"})
		return
	}

	// 2. 获取上传的文件
	file, err := c.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "请选择头像文件"})
		return
	}

	// 3. 读取文件内容
	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "读取文件失败"})
		return
	}
	defer src.Close()

	// 读取文件字节
	fileBytes, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "读取文件内容失败"})
		return
	}

	// 4. 调用Service上传头像
	ctx := context.WithValue(c.Request.Context(), "user_id", userID)
	avatarURL, err := h.userService.UploadAvatar(ctx, userID, fileBytes, file.Filename)
	if err != nil {
		status := http.StatusBadRequest
		errorMsg := err.Error()

		// 如果是文件格式错误，返回400
		if strings.Contains(err.Error(), "不支持的图片格式") {
			status = http.StatusBadRequest
		} else {
			// 其他错误返回500
			status = http.StatusInternalServerError
			slog.Error("上传头像失败", "user_id", userID, "error", err)
			errorMsg = "上传头像失败，请稍后重试"
		}

		c.JSON(status, ErrorResponse{Error: errorMsg})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"avatar_url": avatarURL,
		"message":    "头像上传成功",
	})
}

// DeleteAvatar 删除头像
func (h *UserHandler) DeleteAvatar(c *gin.Context) {
	// 1. 获取当前用户ID
	userID, err := utils.GetUserIDFromGin(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "用户未认证"})
		return
	}

	// 2. 调用Service删除头像
	ctx := context.WithValue(c.Request.Context(), "user_id", userID)
	err = h.userService.DeleteAvatar(ctx, userID)
	if err != nil {
		slog.Error("删除头像失败", "user_id", userID, "error", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "删除头像失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "头像删除成功",
	})
}

// GetAvatar 获取用户头像URL
func (h *UserHandler) GetAvatar(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "用户名不能为空"})
		return
	}

	// 获取用户公开资料以获取头像URL
	resp, err := h.userService.GetUserPublicProfile(c.Request.Context(), username)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"avatar_url": resp.AvatarURL,
		"username":   username,
	})
}
