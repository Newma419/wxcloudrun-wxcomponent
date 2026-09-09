package custom

import (
	"net/http"
	"time"

	"github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/log"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/db"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SaveProfileRequest 保存用户信息请求
type SaveProfileRequest struct {
	AvatarUrl   string `json:"avatarUrl"`
	NickName    string `json:"nickName"`
	PhoneNumber string `json:"phoneNumber"`
	Openid      string `json:"openid"`
}

// User 用户表结构（根据实际表结构调整）
type User struct {
	ID          int        `gorm:"column:id;primaryKey;autoIncrement"`
	Openid      string     `gorm:"column:openid;uniqueIndex"`
	NickName    string     `gorm:"column:nick_name"`
	AvatarUrl   string     `gorm:"column:avatar_url"`
	PhoneNumber string     `gorm:"column:phone_number"`
	Balance     float64    `gorm:"column:balance;default:0"`
	CreateTime  *time.Time `gorm:"column:create_time;autoCreateTime"`
	UpdateTime  *time.Time `gorm:"column:update_time;autoUpdateTime"`
}

func (User) TableName() string {
	return "user"
}

// SaveProfile 保存用户信息（存在则更新，不存在则创建）
func SaveProfile(c *gin.Context) {
	var req SaveProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误",
		})
		return
	}

	if req.Openid == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "openid 不能为空",
		})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "数据库连接失败",
		})
		return
	}

	// 更新或插入用户信息
	var user User
	result := dbConn.Where("openid = ?", req.Openid).First(&user)

	if result.Error != nil {
		// 如果用户不存在，创建新用户
		if result.Error == gorm.ErrRecordNotFound {
			newUser := User{
				Openid:      req.Openid,
				NickName:    req.NickName,
				AvatarUrl:   req.AvatarUrl,
				PhoneNumber: req.PhoneNumber,
				Balance:     0,
			}
			if err := dbConn.Create(&newUser).Error; err != nil {
				log.Errorf("创建用户失败: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":    500,
					"message": "创建用户失败",
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"data": gin.H{
					"openid":      newUser.Openid,
					"avatarUrl":   newUser.AvatarUrl,
					"nickName":    newUser.NickName,
					"phoneNumber": newUser.PhoneNumber,
					"balance":     newUser.Balance,
				},
			})
			return
		}
		log.Errorf("查询用户失败: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "查询用户失败",
		})
		return
	}

	// 更新用户信息
	updates := map[string]interface{}{
		"nick_name":    req.NickName,
		"avatar_url":   req.AvatarUrl,
		"phone_number": req.PhoneNumber,
		"update_time":  time.Now(),
	}
	if err := dbConn.Model(&user).Updates(updates).Error; err != nil {
		log.Errorf("更新用户失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "更新用户失败",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"openid":      user.Openid,
			"avatarUrl":   req.AvatarUrl,
			"nickName":    req.NickName,
			"phoneNumber": req.PhoneNumber,
			"balance":     user.Balance,
		},
	})
}
