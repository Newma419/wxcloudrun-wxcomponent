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

	// 构建查询条件
	query := dbConn.Table("dish")
	if categoryId != "" {
		query = query.Where("category_id = ?", categoryId)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 统计总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Errorf("统计菜品总数失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": []interface{}{}, "total": 0}})
		return
	}

	// 查询列表
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
