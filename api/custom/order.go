// api/custom/order.go
package custom

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/log"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/db"
)

// ============================================================
// 获取订单列表
// ============================================================
func GetOrderList(c *gin.Context) {
	openid := c.Query("openid")
	if openid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少 openid 参数"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "0"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	status := c.DefaultQuery("status", "all") // all, pending, paid, completed, cancelled
	orderType := c.DefaultQuery("type", "all") // all, dine_in, takeaway

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	query := dbConn.Table("order").Where("openid = ?", openid)
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}
	if orderType != "" && orderType != "all" {
		query = query.Where("type = ?", orderType)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Errorf("统计订单总数失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": []interface{}{}, "total": 0}})
		return
	}

	var list []map[string]interface{}
	err := query.Offset(page * size).Limit(size).Order("create_time DESC").Find(&list).Error
	if err != nil {
		log.Errorf("查询订单列表失败: %v", err)
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
// 获取预定列表（额外功能）
// ============================================================
func GetReservationList(c *gin.Context) {
	openid := c.Query("openid")
	if openid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少 openid 参数"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "0"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	query := dbConn.Table("reservation").Where("openid = ?", openid)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Errorf("统计预定总数失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": []interface{}{}, "total": 0}})
		return
	}

	var list []map[string]interface{}
	err := query.Offset(page * size).Limit(size).Order("create_time DESC").Find(&list).Error
	if err != nil {
		log.Errorf("查询预定列表失败: %v", err)
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
