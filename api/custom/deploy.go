package custom

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/log"
    "github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/wx"
)

// DeployRequest 定义部署请求的参数结构
type DeployRequest struct {
    Appid        string `json:"appid"`         // 客户小程序的AppID
    TemplateID   string `json:"template_id"`   // 模板ID
    ExtJSON      string `json:"ext_json"`      // 扩展配置
    UserVersion  string `json:"user_version"`  // 版本号
    UserDesc     string `json:"user_desc"`     // 版本描述
}

// DeployHandler 处理部署请求
func DeployHandler(c *gin.Context) {
    var req DeployRequest
    // 1. 解析请求参数
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "参数解析失败: " + err.Error()})
        return
    }

    // 2. 通过Appid获取authorizer_access_token
    //    微管家内部方法，可以从数据库根据Appid获取有效的token
    accessToken, err := wx.GetAuthorizerAccessToken(req.Appid)
    if err != nil {
        log.Errorf("获取authorizer_access_token失败: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "获取token失败"})
        return
    }

    // 3. 调用微信commit接口，部署模板代码
    //    这里的路径 /wxa/commit 就是微信的commit接口
    wxCommErr, body, err := wx.PostWxJsonWithAuthToken(
        req.Appid,
        "/wxa/commit",
        "",
        map[string]interface{}{
            "template_id":   req.TemplateID,
            "ext_json":      req.ExtJSON,
            "user_version":  req.UserVersion,
            "user_desc":     req.UserDesc,
        },
    )

    // 4. 处理调用结果
    if err != nil {
        log.Errorf("调用commit接口失败: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "调用微信接口失败"})
        return
    }

    if wxCommErr.ErrCode != 0 {
        c.JSON(http.StatusOK, gin.H{
            "errcode": wxCommErr.ErrCode,
            "errmsg":  wxCommErr.ErrMsg,
        })
        return
    }

    // 5. 返回成功响应
    c.JSON(http.StatusOK, gin.H{
        "errcode": 0,
        "errmsg":  "ok",
        "data":    string(body),
    })
}
