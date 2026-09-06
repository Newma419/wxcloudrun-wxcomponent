package custom

import (
	"net/http"

	"github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/log"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/db"
	"github.com/gin-gonic/gin"
)

// SaveProfileRequest 保存用户信息请求
type SaveProfileRequest struct {
	AvatarUrl   string `json:"avatarUrl"`
	NickName    string `json:"nickName"`
	PhoneNumber string `json:"phoneNumber"`
	Openid      string `json:"openid"`
}

// SaveProfile 保存用户信息
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
	// 实际逻辑根据您的表结构调整
	err
