// middleware/auth.go
package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/log"
)

var jwtSecret = []byte("your-secret-key-change-in-production") // 建议从环境变量读取

type Claims struct {
	AdminID   int    `json:"admin_id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateAdminToken 生成管理员 JWT
func GenerateAdminToken(adminID int, username, role string) (string, error) {
	expireTime := time.Now().Add(24 * time.Hour)
	claims := Claims{
		AdminID:  adminID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expireTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "wxcloudrun-wxcomponent",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// AdminAuth 管理员鉴权中间件
func AdminAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "未提供认证令牌"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "认证格式错误"})
			c.Abort()
			return
		}

		tokenStr := parts[1]
		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			log.Errorf("JWT验证失败: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "message": "无效的认证令牌"})
			c.Abort()
			return
		}

		// 将管理员信息存入上下文
		c.Set("admin_id", claims.AdminID)
		c.Set("admin_username", claims.Username)
		c.Set("admin_role", claims.Role)
		c.Next()
	}
}
