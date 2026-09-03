// api/admin/order.go
package admin

import (
	"net/http"
	"strconv"

	"github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/log"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/db"
	"github.com/gin-gonic/gin"
)

// ============================================================
// 订单管理（管理后台）
// ============================================================

// GetOrderListAdmin 获取订单列表（管理后台）
func GetOrderListAdmin(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "0"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	status := c.Query("status")
	keyword := c.Query("keyword") // 按订单号或用户手机号搜索
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	query := dbConn.Table("order")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if keyword != "" {
		query = query.Where("order_no LIKE ? OR openid LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}
	if startDate != "" {
		query = query.Where("create_time >= ?", startDate)
	}
	if endDate != "" {
		query = query.Where("create_time <= ?", endDate)
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

// UpdateOrderStatus 更新订单状态
func UpdateOrderStatus(c *gin.Context) {
	var req struct {
		OrderID int    `json:"order_id"`
		Status  string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if req.OrderID <= 0 || req.Status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数不完整"})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	err := dbConn.Exec("UPDATE `order` SET status = ? WHERE id = ?", req.Status, req.OrderID).Error
	if err != nil {
		log.Errorf("更新订单状态失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// GetOrderDetailAdmin 获取订单详情（管理后台）
func GetOrderDetailAdmin(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少订单ID"})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	var order map[string]interface{}
	err := dbConn.Table("order").Where("id = ?", id).First(&order).Error
	if err != nil {
		log.Errorf("查询订单详情失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "订单不存在"})
		return
	}

	// 查询订单商品详情（假设有 order_items 表）
	var items []map[string]interface{}
	dbConn.Table("order_items").Where("order_id = ?", id).Find(&items)
	order["items"] = items

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": order})
}

// DeleteOrder 删除订单（管理后台）
func DeleteOrder(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少订单ID"})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	err := dbConn.Exec("DELETE FROM `order` WHERE id = ?", id).Error
	if err != nil {
		log.Errorf("删除订单失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}
