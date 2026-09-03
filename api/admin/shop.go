// api/admin/shop.go
package admin

import (
	"net/http"
	"time"

	"github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/log"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/db"
	"github.com/gin-gonic/gin"
)

// ============================================================
// 店铺设置（管理后台）
// ============================================================

type ShopSettings struct {
	ID          int    `json:"id" gorm:"column:id"`
	ShopName    string `json:"shopName" gorm:"column:shop_name"`
	WelcomeText string `json:"welcomeText" gorm:"column:welcome_text"`
	Address     string `json:"address" gorm:"column:address"`
	Phone       string `json:"phone" gorm:"column:phone"`
	UpdateTime  string `json:"update_time" gorm:"column:update_time"`
}

// GetShopSettingsAdmin 获取店铺设置（管理后台）
func GetShopSettingsAdmin(c *gin.Context) {
	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	var settings ShopSettings
	err := dbConn.Table("shop_settings").First(&settings).Error
	if err != nil {
		// 如果没有记录，返回默认值
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"shopName":    "智小助餐厅",
				"welcomeText": "欢迎光临，用心做好每一餐",
				"address":     "",
				"phone":       "",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": settings})
}

// UpdateShopSettings 更新店铺设置
func UpdateShopSettings(c *gin.Context) {
	var req ShopSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	// 检查是否存在
	var count int64
	dbConn.Table("shop_settings").Count(&count)

	now := time.Now().Format("2006-01-02 15:04:05")
	if count == 0 {
		// 插入
		err := dbConn.Exec(`
			INSERT INTO shop_settings (shop_name, welcome_text, address, phone, update_time)
			VALUES (?, ?, ?, ?, ?)
		`, req.ShopName, req.WelcomeText, req.Address, req.Phone, now).Error
		if err != nil {
			log.Errorf("创建店铺设置失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存失败"})
			return
		}
	} else {
		// 更新
		err := dbConn.Exec(`
			UPDATE shop_settings SET 
				shop_name = ?, welcome_text = ?, address = ?, phone = ?, update_time = ?
		`, req.ShopName, req.WelcomeText, req.Address, req.Phone, now).Error
		if err != nil {
			log.Errorf("更新店铺设置失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "保存成功"})
}
