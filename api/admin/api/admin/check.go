package admin

import (
	"net/http"

	"github.com/WeixinCloud/wxcloudrun-wxcomponent/db"
	"github.com/gin-gonic/gin"
)

// CheckAdminFirstTime 检查管理员是否首次创建（即是否存在管理员账号）
func CheckAdminFirstTime(c *gin.Context) {
	dbConn := db.Get()
	if dbConn == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "数据库连接失败",
		})
		return
	}

	var count int64
	err := dbConn.Table("admin").Count(&count).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询失败",
		})
		return
	}

	exists := count > 0

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"exists": exists,
		},
	})
}
