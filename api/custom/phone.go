package custom

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/log"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/config"
	"github.com/gin-gonic/gin"
)

// DecryptPhoneRequest 解密手机号请求
type DecryptPhoneRequest struct {
	Code string `json:"code" binding:"required"`
}

// DecryptPhoneResponse 解密手机号响应
type DecryptPhoneResponse struct {
	PhoneNumber string `json:"phoneNumber"`
}

// WechatPhoneResponse 微信手机号接口响应
type WechatPhoneResponse struct {
	ErrCode int `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
	PhoneInfo struct {
		PhoneNumber string `json:"phoneNumber"`
		PurePhoneNumber string `json:"purePhoneNumber"`
		CountryCode string `json:"countryCode"`
	} `json:"phone_info"`
}

// DecryptPhone 解密手机号（调用微信官方接口）
func DecryptPhone(c *gin.Context) {
	var req DecryptPhoneRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误：缺少 code",
		})
		return
	}

	if req.Code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "参数错误：code 不能为空",
		})
		return
	}

	// 获取 access_token
	accessToken, err := getAccessToken()
	if err != nil {
		log.Errorf("获取 access_token 失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "获取 access_token 失败",
		})
		return
	}

	// 调用微信接口解密手机号
	url := fmt.Sprintf("https://api.weixin.qq.com/wxa/business/getuserphonenumber?access_token=%s", accessToken)
	requestBody := map[string]string{
		"code": req.Code,
	}
	bodyBytes, _ := json.Marshal(requestBody)

	resp, err := http.Post(url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		log.Errorf("调用微信接口失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "调用微信接口失败",
		})
		return
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("读取微信接口响应失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "读取响应失败",
		})
		return
	}

	var wechatResp WechatPhoneResponse
	if err := json.Unmarshal(respBody, &wechatResp); err != nil {
		log.Errorf("解析微信接口响应失败: %v, body: %s", err, string(respBody))
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "解析响应失败",
		})
		return
	}

	if wechatResp.ErrCode != 0 {
		log.Errorf("微信接口返回错误: code=%d, msg=%s", wechatResp.ErrCode, wechatResp.ErrMsg)
		c.JSON(http.StatusOK, gin.H{
			"code":    wechatResp.ErrCode,
			"message": wechatResp.ErrMsg,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": DecryptPhoneResponse{
			PhoneNumber: wechatResp.PhoneInfo.PhoneNumber,
		},
	})
}

// getAccessToken 获取微信 access_token
func getAccessToken() (string, error) {
	// 从配置中读取 appid 和 secret
	cfg := config.Get()
	if cfg == nil {
		return "", fmt.Errorf("配置未初始化")
	}

	appID := cfg.Wechat.AppID
	appSecret := cfg.Wechat.AppSecret

	if appID == "" || appSecret == "" {
		return "", fmt.Errorf("appid 或 appsecret 未配置")
	}

	// 这里可以加缓存，避免每次请求都获取
	// 简单实现：直接获取
	url := fmt.Sprintf("https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s", appID, appSecret)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("获取 access_token 失败: %s", result.ErrMsg)
	}

	return result.AccessToken, nil
}
