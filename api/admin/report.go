// api/admin/report.go
package admin

import (
	"net/http"
	"time"

	"github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/log"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/db"
	"github.com/gin-gonic/gin"
)

// ============================================================
// 数据报表（管理后台）
// ============================================================

// GetDashboardStats 获取仪表盘统计数据
func GetDashboardStats(c *gin.Context) {
	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	// 今日订单数
	var todayOrders int64
	dbConn.Table("order").Where("DATE(create_time) = CURDATE()").Count(&todayOrders)

	// 今日营收
	var todayRevenue float64
	dbConn.Table("order").Where("DATE(create_time) = CURDATE() AND status = 'paid'").
		Select("COALESCE(SUM(total_amount), 0)").Scan(&todayRevenue)

	// 总会员数
	var totalUsers int64
	dbConn.Table("user").Count(&totalUsers)

	// 总订单数
	var totalOrders int64
	dbConn.Table("order").Count(&totalOrders)

	// 近7天每日订单数
	var dailyOrders []map[string]interface{}
	dbConn.Table("order").
		Select("DATE(create_time) as date, COUNT(*) as count").
		Where("create_time >= DATE_SUB(CURDATE(), INTERVAL 7 DAY)").
		Group("DATE(create_time)").
		Order("date ASC").
		Find(&dailyOrders)

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"todayOrders":    todayOrders,
			"todayRevenue":   todayRevenue,
			"totalUsers":     totalUsers,
			"totalOrders":    totalOrders,
			"dailyOrders":    dailyOrders,
			"updateTime":     time.Now().Format("2006-01-02 15:04:05"),
		},
	})
}
