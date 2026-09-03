// api/custom/category.go
package custom

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/log"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/db"
)

// ============================================================
// 获取菜品分类列表
// ============================================================
func GetCategoryList(c *gin.Context) {
	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	type Category struct {
		ID   string `json:"_id"`
		Name string `json:"name"`
		Sort int    `json:"sort"`
		Icon string `json:"icon"`
	}

	var categories []Category

	err := dbConn.Raw(`
		SELECT id, name, sort, icon FROM dish_category 
		ORDER BY sort ASC, id ASC
	`).Scan(&categories).Error

	if err != nil {
		// 如果表不存在或没有数据，返回空数组（不报错）
		log.Infof("查询分类失败，返回空列表: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": []Category{},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": categories,
	})
}
