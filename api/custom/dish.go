// api/custom/dish.go
package custom

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/log"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/db"
)

// ============================================================
// 获取菜品详情
// ============================================================
func GetDishDetail(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少 id 参数"})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	var dish map[string]interface{}
	err := dbConn.Table("dish").Where("id = ?", id).First(&dish).Error
	if err != nil {
		log.Errorf("查询菜品详情失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "菜品不存在"})
		return
	}

	// 查询菜品图片（如果有单独的表）
	var images []string
	dbConn.Table("dish_images").Where("dish_id = ?", id).Pluck("image_url", &images)

	// 查询规格（如果有单独的表）
	var skus []map[string]interface{}
	dbConn.Table("dish_sku").Where("dish_id = ?", id).Order("sort ASC").Find(&skus)

	// 查询标签（如果有单独的表）
	var tags []map[string]interface{}
	dbConn.Table("dish_tag").Where("dish_id = ?", id).Find(&tags)

	// 组装返回数据
	dish["images"] = images
	dish["skus"] = skus
	dish["tags"] = tags

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": dish,
	})
}
