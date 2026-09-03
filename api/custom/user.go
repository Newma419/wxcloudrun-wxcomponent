// routers/routers.go

// ========== 自定义路由组 ==========
customGroup := r.Group("/custom")
{
    // ============================================================
    // 部署
    // ============================================================
    customGroup.POST("/deploy", custom.DeployHandler)

    // ============================================================
    // ★★★ 测试接口 - 手动配置隐私协议 ★★★
    // ============================================================
    customGroup.GET("/setup-privacy", custom.SetupPrivacy)

    // ============================================================
    // 用户登录相关
    // ============================================================
    customGroup.POST("/user/login-phone", custom.LoginByPhone)
    customGroup.POST("/user/set-password", custom.SetPassword)
    customGroup.POST("/user/check-token", custom.CheckToken)
    customGroup.POST("/user/verify-phone-reset", custom.VerifyPhoneForReset)
    customGroup.POST("/user/reset-password", custom.ResetPassword)

    // ... 其他路由
}
