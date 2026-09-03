// api/admin/recharge.go
package admin

import (
	"net/http"
	"strconv"

	"github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/log"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/db"
	"github.com/gin-gonic/gin"
)

// ============================================================
// 充值选项管理（管理后台）
// ============================================================

type RechargeOption struct {
	ID          int     `json:"id" gorm:"column:id"`
	Amount      float64 `json:"amount" gorm:"column:amount"`
	GiveAmount  float64 `json:"giveAmount" gorm:"column:give_amount"`
	IsRecommend int     `json:"isRecommend" gorm:"column:is_recommend"`
	Status      int     `json:"status" gorm:"column:status"`
	Description string  `json:"description" gorm:"column:description"`
}

// GetRechargeOptions 获取充值选项列表
func GetRechargeOptions(c *gin.Context) {
	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	var list []RechargeOption
	err := dbConn.Table("recharge_options").Order("amount ASC").Find(&list).Error
	if err != nil {
		log.Errorf("查询充值选项失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": []interface{}{}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": list})
}

// SaveRechargeOption 创建或更新充值选项
func SaveRechargeOption(c *gin.Context) {
	var req RechargeOption
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "充值金额必须大于0"})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	if req.ID > 0 {
		// 更新
		err := dbConn.Exec(`
			UPDATE recharge_options SET 
				amount = ?, give_amount = ?, is_recommend = ?, status = ?, description = ?
			WHERE id = ?
		`, req.Amount, req.GiveAmount, req.IsRecommend, req.Status, req.Description, req.ID).Error
		if err != nil {
			log.Errorf("更新充值选项失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存失败"})
			return
		}
	} else {
		// 插入
		err := dbConn.Exec(`
			INSERT INTO recharge_options (amount, give_amount, is_recommend, status, description)
			VALUES (?, ?, ?, ?, ?)
		`, req.Amount, req.GiveAmount, req.IsRecommend, req.Status, req.Description).Error
		if err != nil {
			log.Errorf("创建充值选项失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存失败"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "保存成功"})
}

// DeleteRechargeOption 删除充值选项
func DeleteRechargeOption(c *gin.Context) {
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

	err := dbConn.Exec("DELETE FROM recharge_options WHERE id = ?", id).Error
	if err != nil {
		log.Errorf("删除充值选项失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}
