package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthHandler 健康检查处理器
type HealthHandler struct{}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// HealthResponse 健康检查响应结构体
type HealthResponse struct {
	Status    string `json:"status"`
	Timestamp int64  `json:"timestamp"`
}

// VersionResponse 版本信息响应结构体
type VersionResponse struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	Commit    string `json:"commit"`
}

// HealthCheck 健康检查
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().Unix(),
	})
}

// Version 获取版本信息
func (h *HealthHandler) Version(c *gin.Context) {
	c.JSON(http.StatusOK, VersionResponse{
		Version:   "1.0.0",
		BuildTime: "2026-01-01 00:00:00",
		Commit:    "abcdefg",
	})
}
