package custom

import (
    "net/http"
    "github.com/gin-gonic/gin"
    "github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/log"
    "github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/wx"
)

// DeployRequest 定义部署请求的参数结构
type DeployRequest struct {
    Appid       string `json:"appid"`
    TemplateID  string `json:"template_id"`
    ExtJSON     string `json:"ext_json"`
    UserVersion string `json:"user_version"`
    UserDesc    string `json:"user_desc"`
}

// DeployHandler 处理部署请求
func DeployHandler(c *gin.Context) {
    var req DeployRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "参数解析失败: " + err.Error()})
        return
    }

    // 调用微信commit接口，部署模板代码
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

    c.JSON(http.StatusOK, gin.H{
        "errcode": 0,
        "errmsg":  "ok",
        "data":    string(body),
    })
}
