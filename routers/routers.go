// routers/routers.go
package routers

import (
	"net/http"

	"github.com/WeixinCloud/wxcloudrun-wxcomponent/api/admin"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/api/authpage"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/api/custom"
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

	// ============================================================
	// 微信消息推送
	// ============================================================
	wxcallback.Routers(r)

	// ============================================================
	// 微管家（第三方平台管理界面）
	// ============================================================
	Include(admin.Routers, authpage.Routers)
	g := r.Group("/wxcomponent")
	for _, opt := range options {
		opt(g)
	}

	// ============================================================
	// 管理后台路由组
	// ============================================================
	adminGroup := r.Group("/admin")
	{
		// ✅ 新增：检查管理员是否首次创建（不需要认证）
		adminGroup.GET("/check", admin.CheckAdminFirstTime)

		// ---- 登录（不需要认证） ----
		adminGroup.POST("/login", admin.AdminLogin)

		// ---- 需要认证的接口 ----
		authGroup := adminGroup.Group("/", middleware.AdminAuth())
		{
			// 仪表盘
			authGroup.GET("/dashboard/stats", admin.GetDashboardStats)

			// 菜品管理
			authGroup.GET("/dish/list", admin.GetDishList)
			authGroup.POST("/dish/create", admin.CreateDish)
			authGroup.POST("/dish/update", admin.UpdateDish)
			authGroup.DELETE("/dish/delete/:id", admin.DeleteDish)

			// 分类管理
			authGroup.GET("/category/list", admin.GetCategoryListAdmin)
			authGroup.POST("/category/create", admin.CreateCategory)
			authGroup.POST("/category/update", admin.UpdateCategory)
			authGroup.DELETE("/category/delete/:id", admin.DeleteCategory)

			// 订单管理
			authGroup.GET("/order/list", admin.GetOrderListAdmin)
			authGroup.POST("/order/update-status", admin.UpdateOrderStatus)
			authGroup.GET("/order/detail/:id", admin.GetOrderDetailAdmin)
			authGroup.DELETE("/order/delete/:id", admin.DeleteOrder)

			// 会员管理
			authGroup.GET("/user/list", admin.GetUserListAdmin)
			authGroup.POST("/user/update-balance", admin.UpdateUserBalance)

			// 店铺设置
			authGroup.GET("/shop/settings", admin.GetShopSettingsAdmin)
			authGroup.POST("/shop/settings", admin.UpdateShopSettings)

			// 桌码管理
			authGroup.GET("/tablecode/list", admin.GetTableCodeList)
			authGroup.POST("/tablecode/create", admin.CreateTableCode)
			authGroup.DELETE("/tablecode/delete/:id", admin.DeleteTableCode)

			// 充值选项管理
			authGroup.GET("/recharge-options/list", admin.GetRechargeOptions)
			authGroup.POST("/recharge-options/save", admin.SaveRechargeOption)
			authGroup.DELETE("/recharge-options/delete/:id", admin.DeleteRechargeOption)
		}
	}

	// ============================================================
	// 自定义路由组（用户端 API）
	// ============================================================
	customGroup := r.Group("/custom")
	{
		// ---- 部署 ----
		customGroup.POST("/deploy", custom.DeployHandler)

		// ---- 测试接口 ----
		customGroup.GET("/setup-privacy", custom.SetupPrivacy)

		// ---- 用户登录相关 ----
		customGroup.POST("/user/login-phone", custom.LoginByPhone)
		customGroup.POST("/user/set-password", custom.SetPassword)
		customGroup.POST("/user/check-token", custom.CheckToken)
		customGroup.POST("/user/verify-phone-reset", custom.VerifyPhoneForReset)
		customGroup.POST("/user/reset-password", custom.ResetPassword)

		// ---- 用户信息 ----
		customGroup.GET("/user/info", custom.GetUserInfo)

		// ---- 店铺设置 ----
		customGroup.GET("/shop/settings", custom.GetShopSettings)

		// ---- 菜品分类 ----
		customGroup.GET("/category/list", custom.GetCategoryList)

		// ---- 菜品 ----
		customGroup.GET("/dish/list", custom.GetDishList)
		customGroup.GET("/dish/detail", custom.GetDishDetail)

		// ---- 充值套餐 ----
		customGroup.GET("/recharge/list", custom.GetRechargeList)

		// ---- 订单 ----
		customGroup.GET("/order/list", custom.GetOrderList)

		// ---- 预定 ----
		customGroup.GET("/reservation/list", custom.GetReservationList)

		// ---- 优惠券 ----
		customGroup.GET("/coupon/list", custom.GetCouponList)
		customGroup.POST("/coupon/receive", custom.ReceiveCoupon)
		customGroup.GET("/coupon/user/list", custom.GetUserCoupons)
		customGroup.POST("/coupon/available", custom.GetAvailableCoupons)
		customGroup.POST("/coupon/use", custom.UseCoupon)

		// ---- 优惠券管理后台 ----
		customGroup.GET("/coupon/admin/list", custom.GetAdminCouponList)
		customGroup.POST("/coupon/admin/save", custom.SaveCoupon)
		customGroup.POST("/coupon/admin/delete", custom.DeleteCoupon)
	}

	// ============================================================
	// 静态文件 & 首页
	// ============================================================
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
