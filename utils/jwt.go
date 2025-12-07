package utils

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

var jwtSecret = []byte("misono mika")

// Claims 自定义 JWT 声明
type Claims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken 生成 JWT Token
func GenerateToken(userID uint, username, role string) (string, error) {
	nowTime := time.Now()
	expireTime := nowTime.Add(24 * time.Hour) // Token 24小时有效

	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(nowTime),
			Issuer:    "blog-system",
			Subject:   username,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ParseToken 解析 Token
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// JWTAuthMiddleware JWT 认证中间件
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求头获取 token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "请求未携带 token",
			})
			c.Abort()
			return
		}

		// 检查 token 格式
		parts := strings.SplitN(authHeader, " ", 2)
		if !(len(parts) == 2 && parts[0] == "Bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "token 格式错误",
			})
			c.Abort()
			return
		}

		// 解析 token
		claims, err := ParseToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "无效的 token",
			})
			c.Abort()
			return
		}

		// 检查 token 是否过期
		if time.Now().Unix() > claims.ExpiresAt.Unix() {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "token 已过期",
			})
			c.Abort()
			return
		}

		// 将用户信息存入 Gin 上下文
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// RefreshToken 刷新 Token
func RefreshToken(oldToken string) (string, error) {
	claims, err := ParseToken(oldToken)
	if err != nil {
		return "", err
	}

	// 允许在过期前 30 分钟内刷新
	if time.Until(claims.ExpiresAt.Time) > 30*time.Minute {
		return "", errors.New("token 尚未到刷新时间")
	}

	return GenerateToken(claims.UserID, claims.Username, claims.Role)
}

// GetUserIDFromGin 从 Gin 上下文获取用户 ID（给 Handler 层使用）
func GetUserIDFromGin(c *gin.Context) (uint, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return 0, errors.New("用户未认证")
	}

	// 类型断言
	switch v := userID.(type) {
	case uint:
		return v, nil
	case float64:
		return uint(v), nil
	case int:
		return uint(v), nil
	default:
		return 0, errors.New("无效的用户 ID 类型")
	}
}
