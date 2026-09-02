// api/custom/user.go
package custom

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/log"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/db"
	"gorm.io/gorm"
)

// ============================================================
// 工具函数
// ============================================================

// 密码加密（SHA256）
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// 生成登录令牌
func generateToken(userID string) string {
	return fmt.Sprintf("token_%s_%d", userID, time.Now().UnixNano())
}

// 获取微信手机号（通过 code）
// TODO: 正式环境需要替换为真实的微信API调用
func getPhoneNumberByCode(code string) (string, error) {
	log.Infof("获取手机号，code: %s", code)
	// 临时返回模拟手机号（开发测试用）
	return "13800138000", nil
}

// ============================================================
// 1. 一键登录（微信授权获取手机号）
// ============================================================
func LoginByPhone(c *gin.Context) {
	var req struct {
		Code string `json:"code"`
	}

	if err := c.ShouldBindJSON(&req); err != nil || req.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	// 获取底层 *sql.DB 用于原生查询
	sqlDB, err := dbConn.DB()
	if err != nil {
		log.Errorf("获取数据库连接失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	// 1. 调用微信接口获取手机号
	phoneNumber, err := getPhoneNumberByCode(req.Code)
	if err != nil {
		log.Errorf("获取手机号失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "获取手机号失败"})
		return
	}

	// 2. 查询用户是否存在
	var userInfo struct {
		ID            string  `json:"_id"`
		Openid        string  `json:"_openid"`
		PhoneNumber   string  `json:"phoneNumber"`
		NickName      string  `json:"nickName"`
		IsSetPassword int     `json:"is_set_password"`
		Balance       float64 `json:"balance"`
	}

	err = sqlDB.QueryRow(`
		SELECT id, _openid, phoneNumber, nickName, is_set_password, balance 
		FROM user WHERE phoneNumber = ? OR username = ?
	`, phoneNumber, phoneNumber).Scan(
		&userInfo.ID, &userInfo.Openid, &userInfo.PhoneNumber,
		&userInfo.NickName, &userInfo.IsSetPassword, &userInfo.Balance,
	)

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 3. 新用户，创建记录
			openid := fmt.Sprintf("user_%d", time.Now().UnixNano())
			result, err := sqlDB.Exec(`
				INSERT INTO user (_openid, phoneNumber, username, nickName, is_set_password, balance, createTime)
				VALUES (?, ?, ?, ?, 0, 0, NOW())
			`, openid, phoneNumber, phoneNumber, fmt.Sprintf("用户%s", phoneNumber[7:11]))
			if err != nil {
				log.Errorf("创建用户失败: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
				return
			}
			userID, _ := result.LastInsertId()
			userInfo.ID = fmt.Sprintf("%d", userID)
			userInfo.Openid = openid
			userInfo.PhoneNumber = phoneNumber
			userInfo.IsSetPassword = 0
			userInfo.NickName = fmt.Sprintf("用户%s", phoneNumber[7:11])
			userInfo.Balance = 0

			c.JSON(http.StatusOK, gin.H{
				"code": 0,
				"data": gin.H{
					"userInfo":      userInfo,
					"is_first_time": true,
					"token":         "",
				},
			})
			return
		}
		log.Errorf("查询用户失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	// 4. 老用户，生成token
	token := generateToken(userInfo.ID)
	expireTime := time.Now().Add(7 * 24 * time.Hour)

	// ★★★ 修复点：将原来的 _, err = dbConn.Exec(...) 改为 result := dbConn.Exec(...) ★★★
	result := dbConn.Exec(`
		UPDATE user SET token = ?, token_expire = ? WHERE id = ?
	`, token, expireTime, userInfo.ID)

	if result.Error != nil {
		log.Errorf("更新token失败: %v", result.Error)
	} else {
		log.Infof("token更新成功，影响行数: %d", result.RowsAffected)
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"userInfo":      userInfo,
			"is_first_time": userInfo.IsSetPassword == 0,
			"token":         token,
		},
	})
}

// ============================================================
// 2. 设置密码（首次登录）
// ============================================================
func SetPassword(c *gin.Context) {
	var req struct {
		Phone    string `json:"phone"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if len(req.Password) < 6 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "密码至少6位"})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	// 加密密码
	hashed := hashPassword(req.Password)

	// 更新用户密码
	result := dbConn.Exec(`
		UPDATE user SET password = ?, is_set_password = 1 
		WHERE phoneNumber = ? OR username = ?
	`, hashed, req.Phone, req.Phone)

	if result.Error != nil {
		log.Errorf("更新密码失败: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "用户不存在"})
		return
	}

	// 查询用户信息
	var userInfo struct {
		ID          string  `json:"_id"`
		Openid      string  `json:"_openid"`
		PhoneNumber string  `json:"phoneNumber"`
		NickName    string  `json:"nickName"`
		Balance     float64 `json:"balance"`
	}
	err := dbConn.Raw(`
		SELECT id, _openid, phoneNumber, nickName, balance 
		FROM user WHERE phoneNumber = ? OR username = ?
	`, req.Phone, req.Phone).Scan(&userInfo).Error
	if err != nil {
		log.Errorf("查询用户失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	// 生成token
	token := generateToken(userInfo.ID)
	expireTime := time.Now().Add(7 * 24 * time.Hour)
	dbConn.Exec(`
		UPDATE user SET token = ?, token_expire = ? WHERE id = ?
	`, token, expireTime, userInfo.ID)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "密码设置成功",
		"data": gin.H{
			"userInfo": userInfo,
			"token":    token,
		},
	})
}

// ============================================================
// 3. 校验Token（自动登录）
// ============================================================
func CheckToken(c *gin.Context) {
	var req struct {
		Token string `json:"token"`
	}

	if err := c.ShouldBindJSON(&req); err != nil || req.Token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	sqlDB, err := dbConn.DB()
	if err != nil {
		log.Errorf("获取数据库连接失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	var userInfo struct {
		ID          string  `json:"_id"`
		Openid      string  `json:"_openid"`
		PhoneNumber string  `json:"phoneNumber"`
		NickName    string  `json:"nickName"`
		Balance     float64 `json:"balance"`
	}
	var expireTime time.Time
	err = sqlDB.QueryRow(`
		SELECT id, _openid, phoneNumber, nickName, balance, token_expire 
		FROM user WHERE token = ?
	`, req.Token).Scan(
		&userInfo.ID, &userInfo.Openid, &userInfo.PhoneNumber,
		&userInfo.NickName, &userInfo.Balance, &expireTime,
	)

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"valid": false}})
			return
		}
		log.Errorf("查询token失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	if time.Now().After(expireTime) {
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"valid": false}})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{
			"valid":    true,
			"userInfo": userInfo,
		},
	})
}

// ============================================================
// 4. 验证手机号（用于重置密码）
// ============================================================
func VerifyPhoneForReset(c *gin.Context) {
	var req struct {
		Code string `json:"code"`
	}

	if err := c.ShouldBindJSON(&req); err != nil || req.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	sqlDB, err := dbConn.DB()
	if err != nil {
		log.Errorf("获取数据库连接失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	// 获取手机号
	phoneNumber, err := getPhoneNumberByCode(req.Code)
	if err != nil {
		log.Errorf("获取手机号失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "获取手机号失败"})
		return
	}

	// 检查该手机号是否已注册
	var exists int
	err = sqlDB.QueryRow(`
		SELECT COUNT(*) FROM user WHERE phoneNumber = ? OR username = ?
	`, phoneNumber, phoneNumber).Scan(&exists)
	if err != nil {
		log.Errorf("查询用户失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	if exists == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code":    404,
			"message": "该手机号未注册",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "验证通过",
		"data": gin.H{
			"phoneNumber": phoneNumber,
		},
	})
}

// ============================================================
// 5. 重置密码
// ============================================================
func ResetPassword(c *gin.Context) {
	var req struct {
		Phone    string `json:"phone"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if len(req.Password) < 6 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "密码至少6位"})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	// 加密密码
	hashed := hashPassword(req.Password)

	// 更新密码
	result := dbConn.Exec(`
		UPDATE user SET password = ?, is_set_password = 1 
		WHERE phoneNumber = ? OR username = ?
	`, hashed, req.Phone, req.Phone)

	if result.Error != nil {
		log.Errorf("重置密码失败: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "密码重置成功",
	})
}
