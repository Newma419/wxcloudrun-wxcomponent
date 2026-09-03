// api/custom/user.go
package custom

import (
	"bytes"
	"crypto/sha256"
	"database/sql" // ★★★ 新增 import ★★★
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/log"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/wx/base"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/db"
	"gorm.io/gorm"
)

// ============================================================
// Token 类型常量
// ============================================================
const (
	TokenTypeComponentAccess  = 1 // component_access_token
	TokenTypeAuthorizerAccess = 2 // authorizer_access_token
)

// ============================================================
// 工具函数
// ============================================================

func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

func generateToken(userID string) string {
	return fmt.Sprintf("token_%s_%d", userID, time.Now().UnixNano())
}

// ============================================================
// 获取第三方平台配置
// ============================================================

func getComponentAppid() string {
	return "wx0c8a63459db5b5c8"
}

func getComponentAppsecret() string {
	return "e07a8212bf494fffbda69dc8d34b4039"
}

// ============================================================
// 获取 component_verify_ticket（从 wxbase 统一读取 + 数据库降级）
// ============================================================
func getComponentVerifyTicket() string {
	ticket := wxbase.GetTicket()
	if ticket != "" {
		log.Infof("从 comm 表获取 component_verify_ticket 成功")
		return ticket
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空，无法获取 component_verify_ticket")
		return ""
	}

	var postbody string
	err := dbConn.Raw(`
		SELECT postbody 
		FROM wxcallback_component 
		WHERE infotype = 'component_verify_ticket'
		ORDER BY id DESC LIMIT 1
	`).Scan(&postbody).Error

	if err != nil || postbody == "" {
		log.Infof("[WARN] 未找到 component_verify_ticket 记录: %v", err)
		return ""
	}

	var data struct {
		ComponentVerifyTicket string `json:"component_verify_ticket"`
	}
	if err := json.Unmarshal([]byte(postbody), &data); err != nil {
		log.Infof("[WARN] 解析 component_verify_ticket 失败: %v", err)
		return ""
	}

	if data.ComponentVerifyTicket == "" {
		log.Infof("[WARN] component_verify_ticket 字段为空")
		return ""
	}

	log.Infof("从 wxcallback_component 表获取 component_verify_ticket 成功（降级）")
	return data.ComponentVerifyTicket
}

// ============================================================
// 获取 component_access_token
// ============================================================
func getComponentAccessToken() (string, error) {
	dbConn := db.Get()
	if dbConn == nil {
		return "", fmt.Errorf("数据库连接为空")
	}

	var tokenInfo struct {
		Token      string    `gorm:"column:token"`
		ExpireTime time.Time `gorm:"column:expiretime"`
	}

	err := dbConn.Raw(`
		SELECT token, expiretime 
		FROM wxtoken 
		WHERE type = ? AND appid = ?
		ORDER BY id DESC LIMIT 1
	`, TokenTypeComponentAccess, getComponentAppid()).Scan(&tokenInfo).Error

	if err != nil || tokenInfo.Token == "" {
		log.Infof("[WARN] component_access_token 不存在，尝试刷新获取")
		return refreshComponentAccessToken()
	}

	if time.Now().Add(5 * time.Minute).After(tokenInfo.ExpireTime) {
		log.Infof("[WARN] component_access_token 即将过期，尝试刷新")
		return refreshComponentAccessToken()
	}

	log.Infof("从缓存获取 component_access_token 成功")
	return tokenInfo.Token, nil
}

// ============================================================
// 刷新 component_access_token
// ============================================================
func refreshComponentAccessToken() (string, error) {
	url := "https://api.weixin.qq.com/cgi-bin/component/api_component_token"

	reqBody := map[string]interface{}{
		"component_appid":         getComponentAppid(),
		"component_appsecret":     getComponentAppsecret(),
		"component_verify_ticket": getComponentVerifyTicket(),
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %v", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("调用微信接口失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	var result struct {
		ErrCode              int    `json:"errcode"`
		ErrMsg               string `json:"errmsg"`
		ComponentAccessToken string `json:"component_access_token"`
		ExpiresIn            int    `json:"expires_in"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析响应失败: %v, body: %s", err, string(body))
	}

	if result.ErrCode != 0 {
		log.Errorf("刷新 component_access_token 失败: %d, %s", result.ErrCode, result.ErrMsg)
		return "", fmt.Errorf("微信接口错误(%d): %s", result.ErrCode, result.ErrMsg)
	}

	dbConn := db.Get()
	if dbConn != nil {
		expireTime := time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
		err = dbConn.Exec(`
			INSERT INTO wxtoken (type, appid, token, expiretime, createtime, updateTime)
			VALUES (?, ?, ?, ?, NOW(), NOW())
		`, TokenTypeComponentAccess, getComponentAppid(), result.ComponentAccessToken, expireTime).Error
		if err != nil {
			log.Infof("[WARN] 缓存 component_access_token 失败: %v", err)
		}
	}

	log.Infof("成功刷新 component_access_token")
	return result.ComponentAccessToken, nil
}

// ============================================================
// 获取 authorizer_access_token
// ============================================================
func getAuthorizerAccessToken(authorizerAppid string) (string, error) {
	dbConn := db.Get()
	if dbConn == nil {
		return "", fmt.Errorf("数据库连接为空")
	}

	var cachedToken struct {
		Token      string    `gorm:"column:token"`
		ExpireTime time.Time `gorm:"column:expiretime"`
	}

	err := dbConn.Raw(`
		SELECT token, expiretime 
		FROM wxtoken 
		WHERE type = ? AND appid = ?
		ORDER BY id DESC LIMIT 1
	`, TokenTypeAuthorizerAccess, authorizerAppid).Scan(&cachedToken).Error

	if err == nil && cachedToken.Token != "" && time.Now().Add(5*time.Minute).Before(cachedToken.ExpireTime) {
		log.Infof("从缓存获取 authorizer_access_token 成功, appid: %s", authorizerAppid)
		return cachedToken.Token, nil
	}

	var refreshTokenInfo struct {
		RefreshToken string `gorm:"column:refreshtoken"`
	}
	err = dbConn.Raw(`
		SELECT refreshtoken 
		FROM authorizers 
		WHERE appid = ?
	`, authorizerAppid).Scan(&refreshTokenInfo).Error

	if err != nil {
		return "", fmt.Errorf("获取 authorizer_refresh_token 失败: %v", err)
	}
	if refreshTokenInfo.RefreshToken == "" {
		return "", fmt.Errorf("商户 %s 未授权，请先完成授权流程", authorizerAppid)
	}

	componentAccessToken, err := getComponentAccessToken()
	if err != nil {
		return "", fmt.Errorf("获取 component_access_token 失败: %v", err)
	}

	url := "https://api.weixin.qq.com/cgi-bin/component/api_authorizer_token"

	reqBody := map[string]interface{}{
		"component_appid":          getComponentAppid(),
		"authorizer_appid":         authorizerAppid,
		"authorizer_refresh_token": refreshTokenInfo.RefreshToken,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %v", err)
	}

	requestURL := fmt.Sprintf("%s?component_access_token=%s", url, componentAccessToken)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(requestURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("调用微信刷新接口失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	var result struct {
		ErrCode                int    `json:"errcode"`
		ErrMsg                 string `json:"errmsg"`
		AuthorizerAccessToken  string `json:"authorizer_access_token"`
		ExpiresIn              int    `json:"expires_in"`
		AuthorizerRefreshToken string `json:"authorizer_refresh_token"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析响应失败: %v, body: %s", err, string(body))
	}

	if result.ErrCode != 0 {
		log.Errorf("微信刷新 token 失败: %d, %s", result.ErrCode, result.ErrMsg)
		return "", fmt.Errorf("微信接口错误(%d): %s", result.ErrCode, result.ErrMsg)
	}

	expireTime := time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	err = dbConn.Exec(`
		INSERT INTO wxtoken (type, appid, token, expiretime, createtime, updateTime)
		VALUES (?, ?, ?, ?, NOW(), NOW())
	`, TokenTypeAuthorizerAccess, authorizerAppid, result.AuthorizerAccessToken, expireTime).Error

	if err != nil {
		log.Infof("[WARN] 缓存 authorizer_access_token 失败: %v", err)
	}

	if result.AuthorizerRefreshToken != "" && result.AuthorizerRefreshToken != refreshTokenInfo.RefreshToken {
		err = dbConn.Exec(`
			UPDATE authorizers 
			SET refreshtoken = ?, updatetime = NOW()
			WHERE appid = ?
		`, result.AuthorizerRefreshToken, authorizerAppid).Error
		if err != nil {
			log.Infof("[WARN] 更新 refreshtoken 失败: %v", err)
		}
	}

	log.Infof("成功刷新 authorizer_access_token, appid: %s", authorizerAppid)
	return result.AuthorizerAccessToken, nil
}

// ============================================================
// ★★★ 配置商户小程序隐私协议 ★★★
// ============================================================
func setupPrivacySetting(authorizerAccessToken string, authorizerAppid string) error {
	log.Infof("开始配置隐私协议, appid: %s", authorizerAppid)

	url := "https://api.weixin.qq.com/cgi-bin/component/setprivacysetting"

	reqBody := map[string]interface{}{
		"privacy_ver": 2,
		"owner_setting": map[string]string{
			"contact_name":  "智小助科技",
			"contact_email": "19974995457@163.com",
			"contact_phone": "19974995457",
			"notice_method": "公告",
		},
		"setting_list": []map[string]string{
			{
				"privacy_key":  "PhoneNumber",
				"privacy_text": "用于用户注册和登录，方便您管理订单和个人信息",
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Errorf("序列化请求失败: %v", err)
		return err
	}

	requestURL := fmt.Sprintf("%s?access_token=%s", url, authorizerAccessToken)
	log.Infof("请求URL: %s", requestURL)
	log.Infof("请求体: %s", string(jsonData))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(requestURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Errorf("调用微信接口失败: %v", err)
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("读取响应失败: %v", err)
		return err
	}

	log.Infof("微信接口返回: %s", string(body))

	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		log.Errorf("解析响应失败: %v", err)
		return err
	}

	if result.ErrCode != 0 {
		log.Errorf("配置隐私协议失败: %d, %s", result.ErrCode, result.ErrMsg)
		return fmt.Errorf("微信接口错误(%d): %s", result.ErrCode, result.ErrMsg)
	}

	log.Infof("✅ 成功配置隐私协议，appid: %s", authorizerAppid)
	return nil
}

// ============================================================
// ★★★ 手动触发配置隐私协议（测试接口） ★★★
// ============================================================
func SetupPrivacy(c *gin.Context) {
	authorizerAppid := c.Query("appid")
	if authorizerAppid == "" {
		authorizerAppid = getDefaultAuthorizerAppid()
	}
	if authorizerAppid == "" {
		c.JSON(http.StatusOK, gin.H{
			"code":    400,
			"message": "未找到商户 AppID，请传入 appid 参数",
		})
		return
	}

	log.Infof("手动触发配置隐私协议, appid: %s", authorizerAppid)

	authorizerAccessToken, err := getAuthorizerAccessToken(authorizerAppid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": "获取 authorizer_access_token 失败: " + err.Error(),
		})
		return
	}

	err = setupPrivacySetting(authorizerAccessToken, authorizerAppid)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    500,
			"message": "配置隐私协议失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "隐私协议配置成功",
		"appid":   authorizerAppid,
	})
}

// ============================================================
// 获取微信手机号（通过 code）
// ============================================================
func getPhoneNumberByCode(code string, authorizerAppid string) (string, error) {
	log.Infof("开始换取手机号，code: %s, appid: %s", code, authorizerAppid)

	authorizerAccessToken, err := getAuthorizerAccessToken(authorizerAppid)
	if err != nil {
		log.Errorf("获取 authorizer_access_token 失败: %v", err)
		return "", fmt.Errorf("获取访问令牌失败: %v", err)
	}

	url := fmt.Sprintf(
		"https://api.weixin.qq.com/wxa/business/getuserphonenumber?access_token=%s",
		authorizerAccessToken,
	)

	reqBody := map[string]interface{}{
		"code": code,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("序列化请求失败: %v", err)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Errorf("调用微信接口失败: %v", err)
		return "", fmt.Errorf("调用微信接口失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}

	log.Infof("微信接口返回: %s", string(body))

	var result struct {
		ErrCode   int    `json:"errcode"`
		ErrMsg    string `json:"errmsg"`
		PhoneInfo struct {
			PhoneNumber     string `json:"phoneNumber"`
			PurePhoneNumber string `json:"purePhoneNumber"`
			CountryCode     string `json:"countryCode"`
		} `json:"phone_info"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		log.Errorf("解析微信返回结果失败: %v", err)
		return "", fmt.Errorf("解析响应失败: %v", err)
	}

	if result.ErrCode != 0 {
		log.Errorf("微信接口返回错误: %d, %s", result.ErrCode, result.ErrMsg)
		return "", fmt.Errorf("微信接口错误(%d): %s", result.ErrCode, result.ErrMsg)
	}

	phoneNumber := result.PhoneInfo.PurePhoneNumber
	if phoneNumber == "" {
		phoneNumber = result.PhoneInfo.PhoneNumber
	}
	log.Infof("成功换取手机号: %s", phoneNumber)

	if err := setupPrivacySetting(authorizerAccessToken, authorizerAppid); err != nil {
		log.Errorf("配置隐私协议失败（不影响登录）: %v", err)
	} else {
		log.Infof("✅ 隐私协议配置成功，appid: %s", authorizerAppid)
	}

	return phoneNumber, nil
}

// ============================================================
// 获取默认商户 AppID（从 authorizers 表）
// ============================================================
func getDefaultAuthorizerAppid() string {
	dbConn := db.Get()
	if dbConn == nil {
		return ""
	}
	var appid string
	err := dbConn.Raw(`SELECT appid FROM authorizers LIMIT 1`).Scan(&appid).Error
	if err != nil {
		return ""
	}
	return appid
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

	authorizerAppid := c.GetHeader("X-Appid")
	if authorizerAppid == "" {
		authorizerAppid = getDefaultAuthorizerAppid()
	}
	if authorizerAppid == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少商户 AppID"})
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

	phoneNumber, err := getPhoneNumberByCode(req.Code, authorizerAppid)
	if err != nil {
		log.Errorf("获取手机号失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "获取手机号失败: " + err.Error()})
		return
	}

	var userInfo struct {
		ID            string  `json:"_id"`
		Openid        string  `json:"_openid"`
		PhoneNumber   string  `json:"phoneNumber"`
		NickName      string  `json:"nickName"`
		IsSetPassword int     `json:"is_set_password"`
		Balance       float64 `json:"balance"`
	}

	// ★★★ 修复：使用 sql.ErrNoRows 判断用户是否存在 ★★★
	err = sqlDB.QueryRow(`
		SELECT id, _openid, phoneNumber, nickName, is_set_password, balance 
		FROM user WHERE phoneNumber = ? OR username = ?
	`, phoneNumber, phoneNumber).Scan(
		&userInfo.ID, &userInfo.Openid, &userInfo.PhoneNumber,
		&userInfo.NickName, &userInfo.IsSetPassword, &userInfo.Balance,
	)

	if err != nil {
		if err == sql.ErrNoRows {
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

	token := generateToken(userInfo.ID)
	expireTime := time.Now().Add(7 * 24 * time.Hour)

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

	hashed := hashPassword(req.Password)

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
		if err == sql.ErrNoRows {
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

	authorizerAppid := c.GetHeader("X-Appid")
	if authorizerAppid == "" {
		authorizerAppid = getDefaultAuthorizerAppid()
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

	phoneNumber, err := getPhoneNumberByCode(req.Code, authorizerAppid)
	if err != nil {
		log.Errorf("获取手机号失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "获取手机号失败"})
		return
	}

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

	hashed := hashPassword(req.Password)

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
