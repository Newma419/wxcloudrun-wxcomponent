// api/admin/tablecode.go
package admin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/log"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/db"
	"github.com/gin-gonic/gin"
)

// ============================================================
// 桌码管理（管理后台）
// ============================================================

type TableCode struct {
	ID          int    `json:"id" gorm:"column:id"`
	TableNumber string `json:"tableNumber" gorm:"column:table_number"`
	QRCodeUrl   string `json:"qrCodeUrl" gorm:"column:qr_code_url"`
	CreateTime  string `json:"create_time" gorm:"column:create_time"`
}

// GetTableCodeList 获取桌码列表
func GetTableCodeList(c *gin.Context) {
	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	var list []TableCode
	err := dbConn.Table("table_code").Order("id DESC").Find(&list).Error
	if err != nil {
		log.Errorf("查询桌码列表失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": []interface{}{}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": list})
}

// CreateTableCode 生成桌码
func CreateTableCode(c *gin.Context) {
	var req struct {
		TableNumber string `json:"tableNumber"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if req.TableNumber == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "桌号不能为空"})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	// 生成二维码图片 URL（这里简化，实际需要调用微信小程序码接口）
	qrCodeUrl := "https://api.qrserver.com/v1/create-qr-code/?size=200x200&data=table_" + req.TableNumber

	err := dbConn.Exec(`
		INSERT INTO table_code (table_number, qr_code_url, create_time)
		VALUES (?, ?, ?)
	`, req.TableNumber, qrCodeUrl, time.Now().Format("2006-01-02 15:04:05")).Error

	if err != nil {
		log.Errorf("创建桌码失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "生成成功", "data": gin.H{"qrCodeUrl": qrCodeUrl}})
}

// DeleteTableCode 删除桌码
func DeleteTableCode(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少ID"})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	err := dbConn.Exec("DELETE FROM table_code WHERE id = ?", id).Error
	if err != nil {
		log.Errorf("删除桌码失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}
