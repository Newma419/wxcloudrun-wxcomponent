// api/admin/login.go
package admin

import (
	"net/http"

	"github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/log"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/db"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/middleware"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// AdminLogin 管理员登录
func AdminLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "用户名和密码不能为空"})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	var admin struct {
		ID       int    `gorm:"column:id"`
		Username string `gorm:"column:username"`
		Password string `gorm:"column:password"`
		Role     string `gorm:"column:role"`
	}

	// 查询管理员
	err := dbConn.Table("admin").
		Where("username = ?", req.Username).
		First(&admin).Error
	if err != nil {
		log.Errorf("查询管理员失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "用户名或密码错误"})
		return
	}

	// 验证密码（假设密码是 bcrypt 加密的）
	err = bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(req.Password))
	if err != nil {
		log.Errorf("密码验证失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "用户名或密码错误"})
		return
	}

	// 生成 JWT
	token, err := middleware.GenerateAdminToken(admin.ID, admin.Username, admin.Role)
	if err != nil {
		log.Errorf("生成 token 失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"token":    token,
			"admin": gin.H{
				"id":       admin.ID,
				"username": admin.Username,
				"role":     admin.Role,
			},
		},
		"message": "登录成功",
	})
}

// AdminLogout 管理员登出（可选，前端清除 token 即可）
func AdminLogout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "登出成功"})
}
