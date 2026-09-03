// api/custom/dish.go
package custom

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/log"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/db"
)

// ============================================================
// 获取菜品列表
// ============================================================
func GetDishList(c *gin.Context) {
	categoryId := c.Query("categoryId")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "0"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	status := c.DefaultQuery("status", "1")

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	query := dbConn.Table("dish")
	if categoryId != "" {
		query = query.Where("category_id = ?", categoryId)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Errorf("统计菜品总数失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": []interface{}{}, "total": 0}})
		return
	}

	var list []map[string]interface{}
	err := query.Offset(page * size).Limit(size).Order("sort ASC, id DESC").Find(&list).Error
	if err != nil {
		log.Errorf("查询菜品列表失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": []interface{}{}, "total": 0}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"list":  list,
			"total": total,
		},
	})
}

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

	dish["images"] = images
	dish["skus"] = skus
	dish["tags"] = tags

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": dish,
	})
}
