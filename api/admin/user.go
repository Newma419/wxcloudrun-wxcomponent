// api/admin/users.go
package admin

import (
	"net/http"
	"regexp"
	"strconv"

	"github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/errno"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/log"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/utils"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/db"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/db/dao"
	"github.com/gin-gonic/gin"
)

// ============================================================
// 一、管理员账号管理（原有功能）
// ============================================================

type userReq struct {
	Username    string `json:"username"`    // 用户名
	Password    string `json:"password"`    // 密码md5
	OldPassword string `json:"oldPassword"` // 旧密码md5
}

// 更新用户名
func updateUserNameHandler(c *gin.Context) {
	var req userReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error(err.Error())
		c.JSON(http.StatusOK, errno.ErrInvalidParam.WithData(err.Error()))
		return
	}
	if req.Username == "" {
		log.Error("param empty ", req)
		c.JSON(http.StatusOK, errno.ErrInvalidParam)
		return
	}
	jwt, _ := c.Get("jwt")
	Id, _ := strconv.Atoi(jwt.(*utils.Claims).ID)
	if err := dao.UpdateUserRecord(int32(Id), req.Username, "", ""); err != nil {
		log.Error(err.Error())
		c.JSON(http.StatusOK, errno.ErrUserErr.WithData(err.Error()))
		return
	}
	c.JSON(http.StatusOK, errno.OK)
}

// 更新用户密码
func updateUserPwdHandler(c *gin.Context) {
	var req userReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Error(err.Error())
		c.JSON(http.StatusOK, errno.ErrInvalidParam.WithData(err.Error()))
		return
	}
	if req.OldPassword == "" || req.Password == "" {
		log.Error("param empty ", req)
		c.JSON(http.StatusOK, errno.ErrInvalidParam)
		return
	}
	if req.OldPassword == req.Password {
		c.JSON(http.StatusOK, errno.ErrInvalidParam.WithData("the new password is the same as the old password"))
		return
	}
	ok, err := checkPassword(req.Password)
	if err != nil {
		c.JSON(http.StatusOK, errno.ErrSystemError.WithData(err.Error()))
		return
	}
	if !ok {
		c.JSON(http.StatusOK, errno.ErrInvalidParam.WithData("invalid password"))
		return
	}
	jwt, _ := c.Get("jwt")
	Id, _ := strconv.Atoi(jwt.(*utils.Claims).ID)
	if err := dao.UpdateUserRecord(int32(Id), req.Username, req.Password, req.OldPassword); err != nil {
		log.Error(err.Error())
		c.JSON(http.StatusOK, errno.ErrUserErr.WithData(err.Error()))
		return
	}
	c.JSON(http.StatusOK, errno.OK)
}

func checkPassword(pwd string) (bool, error) {
	return regexp.MatchString(`^\w{32}$`, pwd)
}

// ============================================================
// 二、会员管理（新增功能）
// ============================================================

// GetUserListAdmin 获取会员列表（管理后台）
func GetUserListAdmin(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "0"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	keyword := c.Query("keyword")

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	query := dbConn.Table("user")
	if keyword != "" {
		query = query.Where("phoneNumber LIKE ? OR nickName LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Errorf("统计会员总数失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": []interface{}{}, "total": 0}})
		return
	}

	var list []map[string]interface{}
	err := query.Offset(page * size).Limit(size).Order("id DESC").Find(&list).Error
	if err != nil {
		log.Errorf("查询会员列表失败: %v", err)
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

// UpdateUserBalance 调整会员余额
func UpdateUserBalance(c *gin.Context) {
	var req struct {
		UserID  int     `json:"user_id"`
		Balance float64 `json:"balance"`
		Action  string  `json:"action"` // "add" 或 "subtract"
		Remark  string  `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if req.UserID <= 0 || req.Balance <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数不完整"})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	var currentBalance float64
	err := dbConn.Table("user").Where("id = ?", req.UserID).Pluck("balance", &currentBalance).Error
	if err != nil {
		log.Errorf("查询用户余额失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询失败"})
		return
	}

	var newBalance float64
	if req.Action == "add" {
		newBalance = currentBalance + req.Balance
	} else if req.Action == "subtract" {
		if currentBalance < req.Balance {
			c.JSON(http.StatusOK, gin.H{"code": 400, "message": "余额不足"})
			return
		}
		newBalance = currentBalance - req.Balance
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "无效的操作类型"})
		return
	}

	err = dbConn.Exec("UPDATE user SET balance = ? WHERE id = ?", newBalance, req.UserID).Error
	if err != nil {
		log.Errorf("更新余额失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "余额更新成功", "data": gin.H{"new_balance": newBalance}})
}
