// api/custom/shop.go
package custom

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/log"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/db"
)

// ============================================================
// 获取店铺设置
// ============================================================
func GetShopSettings(c *gin.Context) {
	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	var settings struct {
		ShopName    string `json:"shopName"`
		WelcomeText string `json:"welcomeText"`
	}

	// 从数据库读取店铺设置
	err := dbConn.Raw(`
		SELECT shop_name, welcome_text FROM shop_settings LIMIT 1
	`).Scan(&settings).Error

	if err != nil {
		// 如果表不存在或没有数据，返回默认值（不报错）
		log.Infof("读取店铺设置失败，使用默认值: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"code": 0,
			"data": gin.H{
				"shopName":    "智小助餐厅",
				"welcomeText": "欢迎光临，用心做好每一餐",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": settings,
	})
}
