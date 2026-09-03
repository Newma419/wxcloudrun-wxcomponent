package routers

import (
	"net/http"

	"github.com/WeixinCloud/wxcloudrun-wxcomponent/api/custom"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/api/admin"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/api/authpage"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/api/innerservice"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/api/proxy"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/api/wxcallback"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/middleware"
	"github.com/gin-gonic/gin"
)

type Option func(*gin.RouterGroup)

var options []Option

// Include 注册app的路由配置
func Include(opts ...Option) {
	options = append(options, opts...)
}

// Init 初始化
func Init() *gin.Engine {
	r := gin.Default()
	r.Use(middleware.LogMiddleWare)

	// 微信消息推送
	wxcallback.Routers(r)

	// 微管家
	Include(admin.Routers, authpage.Routers)
	g := r.Group("/wxcomponent")
	for _, opt := range options {
		opt(g)
	}

	// ========== 自定义路由组 ==========
	// 用于内部调用，走云托管固定出口 IP
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
		customGroup.POST("/user/login-phone", custom.LoginByPhone)              // 一键登录
		customGroup.POST("/user/set-password", custom.SetPassword)              // 设置密码
		customGroup.POST("/user/check-token", custom.CheckToken)                // 校验token
		customGroup.POST("/user/verify-phone-reset", custom.VerifyPhoneForReset) // 验证手机号（重置密码第一步）
		customGroup.POST("/user/reset-password", custom.ResetPassword)          // 重置密码

		// ============================================================
		// 用户信息
		// ============================================================
		customGroup.GET("/user/info", custom.GetUserInfo)

		// ============================================================
		// 店铺设置
		// ============================================================
		customGroup.GET("/shop/settings", custom.GetShopSettings)

		// ============================================================
		// 菜品分类
		// ============================================================
		customGroup.GET("/category/list", custom.GetCategoryList)

		// ============================================================
		// ★★★ 菜品列表 ★★★
		// ============================================================
		customGroup.GET("/dish/list", custom.GetDishList)

		// ============================================================
		// ★★★ 菜品详情 ★★★
		// ============================================================
		customGroup.GET("/dish/detail", custom.GetDishDetail)

		// ============================================================
		// ★★★ 充值套餐 ★★★
		// ============================================================
		customGroup.GET("/recharge/list", custom.GetRechargeList)

		// ============================================================
		// ★★★ 订单列表 ★★★
		// ============================================================
		customGroup.GET("/order/list", custom.GetOrderList)

		// ============================================================
		// ★★★ 预定列表（额外） ★★★
		// ============================================================
		customGroup.GET("/reservation/list", custom.GetReservationList)

		// ============================================================
		// 优惠券相关接口
		// ============================================================
		// 用户端
		customGroup.GET("/coupon/list", custom.GetCouponList)                // 获取可领取券列表
		customGroup.POST("/coupon/receive", custom.ReceiveCoupon)            // 领取优惠券
		customGroup.GET("/coupon/user/list", custom.GetUserCoupons)          // 获取用户可用券
		customGroup.POST("/coupon/available", custom.GetAvailableCoupons)    // 结算页筛选可用券
		customGroup.POST("/coupon/use", custom.UseCoupon)                    // 核销优惠券

		// 管理后台
		customGroup.GET("/coupon/admin/list", custom.GetAdminCouponList)     // 管理后台券列表
		customGroup.POST("/coupon/admin/save", custom.SaveCoupon)            // 创建/更新券
		customGroup.POST("/coupon/admin/delete", custom.DeleteCoupon)        // 删除券
	}
	// =====================================

	// 静态文件
	g.Static("/assets", "client/dist/wxcomponent/assets")
	r.LoadHTMLGlob("client/dist/index.html")
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})
	r.NoRoute(proxy.ProxyHandler)
	return r
}

// InnerServiceInit 内部服务初始化
func InnerServiceInit() *gin.Engine {
	r := gin.Default()
	r.Use(middleware.LogMiddleWare)
	innerservice.Routers(r)
	return r
}
