// api/custom/coupon.go
package custom

import (
	"database/sql"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/log"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/db"
	"gorm.io/gorm"
)

// Coupon 优惠券模板结构
type Coupon struct {
	ID           int       `json:"id"`
	ShopID       string    `json:"shop_id"`
	Name         string    `json:"name"`
	Type         int       `json:"type"`          // 1-满减券，2-折扣券
	Value        float64   `json:"value"`         // 减免金额或折扣比例
	Threshold    float64   `json:"threshold"`     // 使用门槛
	Stock        int       `json:"stock"`         // 发行总量
	UsedCount    int       `json:"used_count"`    // 已领取数量
	PerUserLimit int       `json:"per_user_limit"` // 每人限领
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Status       int       `json:"status"`        // 1-启用，0-停用
	Description  string    `json:"description"`
}

// UserCoupon 用户优惠券结构
type UserCoupon struct {
	ID           int        `json:"id"`
	UserID       string     `json:"user_id"`
	CouponID     int        `json:"coupon_id"`
	Code         string     `json:"code"`
	Status       int        `json:"status"` // 0-未使用，1-已使用，2-已过期
	ReceivedTime time.Time  `json:"received_time"`
	UsedTime     *time.Time `json:"used_time"`
	OrderID      string     `json:"order_id"`
	ExpireTime   time.Time  `json:"expire_time"`
	// 关联字段
	CouponName  string  `json:"coupon_name,omitempty"`
	CouponType  int     `json:"coupon_type,omitempty"`
	CouponValue float64 `json:"coupon_value,omitempty"`
	Threshold   float64 `json:"threshold,omitempty"`
}

// ============================================================
// 1. 获取可领取的优惠券列表
// ============================================================
func GetCouponList(c *gin.Context) {
	shopID := c.Query("shop_id")
	if shopID == "" {
		shopID = "default"
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "数据库连接失败"})
		return
	}

	sqlDB, err := dbConn.DB()
	if err != nil {
		log.Errorf("获取数据库连接失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	rows, err := sqlDB.Query(`
		SELECT id, shop_id, name, type, value, threshold, stock, used_count, 
		       per_user_limit, start_time, end_time, status, description
		FROM coupons 
		WHERE shop_id = ? AND status = 1 
		  AND start_time <= ? AND end_time >= ?
		  AND stock > used_count
		ORDER BY id DESC
	`, shopID, now, now)

	if err != nil {
		log.Errorf("查询优惠券失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询失败"})
		return
	}
	defer rows.Close()

	var coupons []Coupon
	for rows.Next() {
		var cp Coupon
		err := rows.Scan(
			&cp.ID, &cp.ShopID, &cp.Name, &cp.Type, &cp.Value, &cp.Threshold,
			&cp.Stock, &cp.UsedCount, &cp.PerUserLimit,
			&cp.StartTime, &cp.EndTime, &cp.Status, &cp.Description,
		)
		if err != nil {
			log.Errorf("扫描数据失败: %v", err)
			continue
		}
		coupons = append(coupons, cp)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": coupons})
}

// ============================================================
// 2. 领取优惠券
// ============================================================
func ReceiveCoupon(c *gin.Context) {
	var req struct {
		CouponID int    `json:"coupon_id"`
		UserID   string `json:"user_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if req.CouponID <= 0 || req.UserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数不完整"})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "数据库连接失败"})
		return
	}

	// 开启事务
	tx := dbConn.Begin()
	if tx.Error != nil {
		log.Errorf("开启事务失败: %v", tx.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	// 1. 查询优惠券信息并锁定
	var cp Coupon
	err := tx.Raw(`
		SELECT id, name, type, value, threshold, stock, used_count, per_user_limit, 
		       start_time, end_time, status
		FROM coupons WHERE id = ? FOR UPDATE
	`, req.CouponID).Scan(&cp).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			tx.Rollback()
			c.JSON(http.StatusOK, gin.H{"code": 404, "message": "优惠券不存在"})
			return
		}
		log.Errorf("查询优惠券失败: %v", err)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	// 2. 验证券是否可用
	now := time.Now()
	if cp.Status != 1 {
		tx.Rollback()
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "该优惠券已停用"})
		return
	}
	if now.Before(cp.StartTime) || now.After(cp.EndTime) {
		tx.Rollback()
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "该优惠券已过期或未开始"})
		return
	}
	if cp.Stock <= cp.UsedCount {
		tx.Rollback()
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "优惠券已领完"})
		return
	}

	// 3. 检查用户是否已达到领取上限
	var userCount int
	err = tx.Raw(`
		SELECT COUNT(*) FROM user_coupons 
		WHERE user_id = ? AND coupon_id = ? AND status = 0
	`, req.UserID, req.CouponID).Scan(&userCount).Error
	if err != nil {
		log.Errorf("查询用户领取数量失败: %v", err)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}
	if userCount >= cp.PerUserLimit {
		tx.Rollback()
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "您已达到领取上限"})
		return
	}

	// 4. 生成唯一券码
	code := fmt.Sprintf("%d%s%d", req.CouponID, time.Now().Format("20060102150405"), rand.Intn(10000))

	// 5. 插入用户优惠券
	result := tx.Exec(`
		INSERT INTO user_coupons 
		(user_id, coupon_id, code, expire_time)
		VALUES (?, ?, ?, ?)
	`, req.UserID, req.CouponID, code, cp.EndTime)

	if result.Error != nil {
		log.Errorf("插入用户优惠券失败: %v", result.Error)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "领取失败"})
		return
	}

	// 6. 更新优惠券领取数量
	result = tx.Exec(`UPDATE coupons SET used_count = used_count + 1 WHERE id = ?`, req.CouponID)
	if result.Error != nil {
		log.Errorf("更新优惠券库存失败: %v", result.Error)
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "领取失败"})
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		log.Errorf("提交事务失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "领取失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "领取成功"})
}

// ============================================================
// 3. 获取用户可用优惠券列表
// ============================================================
func GetUserCoupons(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少user_id参数"})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "数据库连接失败"})
		return
	}

	sqlDB, err := dbConn.DB()
	if err != nil {
		log.Errorf("获取数据库连接失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")

	rows, err := sqlDB.Query(`
		SELECT 
			uc.id, uc.user_id, uc.coupon_id, uc.code, uc.status, 
			uc.received_time, uc.used_time, uc.order_id, uc.expire_time,
			c.name, c.type, c.value, c.threshold
		FROM user_coupons uc
		LEFT JOIN coupons c ON uc.coupon_id = c.id
		WHERE uc.user_id = ? AND uc.status = 0 AND uc.expire_time >= ?
		ORDER BY uc.expire_time ASC
	`, userID, now)

	if err != nil {
		log.Errorf("查询用户优惠券失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询失败"})
		return
	}
	defer rows.Close()

	var coupons []UserCoupon
	for rows.Next() {
		var uc UserCoupon
		var usedTime sql.NullTime
		err := rows.Scan(
			&uc.ID, &uc.UserID, &uc.CouponID, &uc.Code, &uc.Status,
			&uc.ReceivedTime, &usedTime, &uc.OrderID, &uc.ExpireTime,
			&uc.CouponName, &uc.CouponType, &uc.CouponValue, &uc.Threshold,
		)
		if err != nil {
			log.Errorf("扫描数据失败: %v", err)
			continue
		}
		if usedTime.Valid {
			uc.UsedTime = &usedTime.Time
		}
		coupons = append(coupons, uc)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": coupons})
}

// ============================================================
// 4. 结算页获取可用优惠券（根据订单金额筛选）
// ============================================================
func GetAvailableCoupons(c *gin.Context) {
	var req struct {
		UserID     string  `json:"user_id"`
		TotalPrice float64 `json:"total_price"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if req.UserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少user_id"})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "数据库连接失败"})
		return
	}

	sqlDB, err := dbConn.DB()
	if err != nil {
		log.Errorf("获取数据库连接失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")

	rows, err := sqlDB.Query(`
		SELECT 
			uc.id, uc.user_id, uc.coupon_id, uc.code, uc.status, 
			uc.received_time, uc.used_time, uc.order_id, uc.expire_time,
			c.name, c.type, c.value, c.threshold
		FROM user_coupons uc
		LEFT JOIN coupons c ON uc.coupon_id = c.id
		WHERE uc.user_id = ? AND uc.status = 0 AND uc.expire_time >= ?
		  AND c.threshold <= ?
		ORDER BY c.value DESC
	`, req.UserID, now, req.TotalPrice)

	if err != nil {
		log.Errorf("查询可用优惠券失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询失败"})
		return
	}
	defer rows.Close()

	var coupons []UserCoupon
	for rows.Next() {
		var uc UserCoupon
		var usedTime sql.NullTime
		err := rows.Scan(
			&uc.ID, &uc.UserID, &uc.CouponID, &uc.Code, &uc.Status,
			&uc.ReceivedTime, &usedTime, &uc.OrderID, &uc.ExpireTime,
			&uc.CouponName, &uc.CouponType, &uc.CouponValue, &uc.Threshold,
		)
		if err != nil {
			log.Errorf("扫描数据失败: %v", err)
			continue
		}
		if usedTime.Valid {
			uc.UsedTime = &usedTime.Time
		}
		coupons = append(coupons, uc)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": coupons})
}

// ============================================================
// 5. 使用优惠券（核销）
// ============================================================
func UseCoupon(c *gin.Context) {
	var req struct {
		UserCouponID int    `json:"user_coupon_id"`
		OrderID      string `json:"order_id"`
		UserID       string `json:"user_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if req.UserCouponID <= 0 || req.OrderID == "" || req.UserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数不完整"})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "数据库连接失败"})
		return
	}

	sqlDB, err := dbConn.DB()
	if err != nil {
		log.Errorf("获取数据库连接失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	// 验证该优惠券属于当前用户且未被使用
	var status int
	err = sqlDB.QueryRow(`
		SELECT status FROM user_coupons 
		WHERE id = ? AND user_id = ?
	`, req.UserCouponID, req.UserID).Scan(&status)

	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusOK, gin.H{"code": 404, "message": "优惠券不存在"})
			return
		}
		log.Errorf("查询优惠券失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	if status != 0 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "该优惠券已被使用"})
		return
	}

	// 更新状态为已使用
	result := dbConn.Exec(`
		UPDATE user_coupons 
		SET status = 1, used_time = ?, order_id = ?
		WHERE id = ? AND user_id = ?
	`, time.Now(), req.OrderID, req.UserCouponID, req.UserID)

	if result.Error != nil {
		log.Errorf("更新优惠券状态失败: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "使用失败"})
		return
	}

	// 获取优惠券详细信息用于返回
	var couponInfo struct {
		CouponID int     `json:"coupon_id"`
		Type     int     `json:"type"`
		Value    float64 `json:"value"`
	}
	err = sqlDB.QueryRow(`
		SELECT c.id, c.type, c.value
		FROM user_coupons uc
		LEFT JOIN coupons c ON uc.coupon_id = c.id
		WHERE uc.id = ?
	`, req.UserCouponID).Scan(&couponInfo.CouponID, &couponInfo.Type, &couponInfo.Value)

	if err != nil {
		log.Errorf("获取优惠券信息失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "使用成功"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "使用成功",
		"data": gin.H{
			"coupon_id": couponInfo.CouponID,
			"type":      couponInfo.Type,
			"value":     couponInfo.Value,
		},
	})
}

// ============================================================
// 6. （管理后台）获取所有优惠券列表
// ============================================================
func GetAdminCouponList(c *gin.Context) {
	shopID := c.Query("shop_id")
	if shopID == "" {
		shopID = "default"
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "数据库连接失败"})
		return
	}

	sqlDB, err := dbConn.DB()
	if err != nil {
		log.Errorf("获取数据库连接失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	rows, err := sqlDB.Query(`
		SELECT id, shop_id, name, type, value, threshold, stock, used_count, 
		       per_user_limit, start_time, end_time, status, description
		FROM coupons 
		WHERE shop_id = ?
		ORDER BY id DESC
	`, shopID)

	if err != nil {
		log.Errorf("查询优惠券失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "查询失败"})
		return
	}
	defer rows.Close()

	var coupons []Coupon
	for rows.Next() {
		var cp Coupon
		err := rows.Scan(
			&cp.ID, &cp.ShopID, &cp.Name, &cp.Type, &cp.Value, &cp.Threshold,
			&cp.Stock, &cp.UsedCount, &cp.PerUserLimit,
			&cp.StartTime, &cp.EndTime, &cp.Status, &cp.Description,
		)
		if err != nil {
			log.Errorf("扫描数据失败: %v", err)
			continue
		}
		coupons = append(coupons, cp)
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": coupons})
}

// ============================================================
// 7. （管理后台）创建/更新优惠券
// ============================================================
func SaveCoupon(c *gin.Context) {
	var req struct {
		ID           int     `json:"id"`
		ShopID       string  `json:"shop_id"`
		Name         string  `json:"name"`
		Type         int     `json:"type"`
		Value        float64 `json:"value"`
		Threshold    float64 `json:"threshold"`
		Stock        int     `json:"stock"`
		PerUserLimit int     `json:"per_user_limit"`
		StartTime    string  `json:"start_time"`
		EndTime      string  `json:"end_time"`
		Status       int     `json:"status"`
		Description  string  `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "请输入优惠券名称"})
		return
	}
	if req.Value <= 0 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "请输入正确的优惠金额"})
		return
	}
	if req.Stock <= 0 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "库存至少为1"})
		return
	}

	if req.ShopID == "" {
		req.ShopID = "default"
	}
	if req.PerUserLimit <= 0 {
		req.PerUserLimit = 1
	}
	if req.Status == 0 {
		req.Status = 1
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "数据库连接失败"})
		return
	}

	var result *gorm.DB
	if req.ID > 0 {
		// 更新
		result = dbConn.Exec(`
			UPDATE coupons SET 
				name = ?, type = ?, value = ?, threshold = ?, 
				stock = ?, per_user_limit = ?, start_time = ?, end_time = ?, 
				status = ?, description = ?
			WHERE id = ? AND shop_id = ?
		`, req.Name, req.Type, req.Value, req.Threshold,
			req.Stock, req.PerUserLimit, req.StartTime, req.EndTime,
			req.Status, req.Description, req.ID, req.ShopID)
	} else {
		// 新增
		result = dbConn.Exec(`
			INSERT INTO coupons 
			(shop_id, name, type, value, threshold, stock, per_user_limit, 
			 start_time, end_time, status, description)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, req.ShopID, req.Name, req.Type, req.Value, req.Threshold,
			req.Stock, req.PerUserLimit, req.StartTime, req.EndTime,
			req.Status, req.Description)
	}

	if result.Error != nil {
		log.Errorf("保存优惠券失败: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "保存失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "保存成功"})
}

// ============================================================
// 8. （管理后台）删除优惠券
// ============================================================
func DeleteCoupon(c *gin.Context) {
	var req struct {
		ID     int    `json:"id"`
		ShopID string `json:"shop_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if req.ID <= 0 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if req.ShopID == "" {
		req.ShopID = "default"
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "数据库连接失败"})
		return
	}

	result := dbConn.Exec(`DELETE FROM coupons WHERE id = ? AND shop_id = ?`, req.ID, req.ShopID)
	if result.Error != nil {
		log.Errorf("删除优惠券失败: %v", result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}
