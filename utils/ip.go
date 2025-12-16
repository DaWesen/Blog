package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// 用于解析太平洋API返回的JSON结构
type IPLocation struct {
	IP     string `json:"ip"`
	Pro    string `json:"pro"`  // 省份
	City   string `json:"city"` // 城市
	Addr   string `json:"addr"` // 完整地址描述
	Region string `json:"region"`
	ISP    string `json:"isp"` // 运营商
}

func queryIPLocation(ip string) (*IPLocation, error) {
	// 构造太平洋网络IP查询API的URL[citation:5][citation:8]
	url := fmt.Sprintf("http://whois.pconline.com.cn/ipJson.jsp?ip=%s&json=true", ip)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("请求API失败: %w", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	var location IPLocation
	if err := json.Unmarshal(body, &location); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	return &location, nil
}

// GetIPFromContext 从上下文中获取客户端IP
func GetIPFromContext(ctx context.Context) string {
	// 尝试从Gin上下文中获取
	if ginCtx, ok := ctx.Value("ginContext").(*gin.Context); ok {
		return GetClientIP(ginCtx.Request)
	}

	// 尝试从标准context中获取
	if req, ok := ctx.Value("httpRequest").(*http.Request); ok {
		return GetClientIP(req)
	}

	return "unknown"
}

// GetClientIP 获取客户端真实IP
func GetClientIP(r *http.Request) string {
	// 尝试从X-Forwarded-For获取
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			clientIP := strings.TrimSpace(ips[0])
			if net.ParseIP(clientIP) != nil {
				return clientIP
			}
		}
	}

	// 尝试从X-Real-IP获取
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		if net.ParseIP(xri) != nil {
			return xri
		}
	}

	// 从RemoteAddr获取
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return ip
}
