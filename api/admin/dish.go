// api/admin/dish.go
package admin

import (
	"net/http"
	"strconv"
	"time"

	"github.com/WeixinCloud/wxcloudrun-wxcomponent/comm/log"
	"github.com/WeixinCloud/wxcloudrun-wxcomponent/db"
	"github.com/gin-gonic/gin"
)

// ============================================================
// 菜品管理（管理后台）
// ============================================================

// Dish 菜品结构
type Dish struct {
	ID          int     `json:"id" gorm:"column:id"`
	Name        string  `json:"name" gorm:"column:name"`
	Price       float64 `json:"price" gorm:"column:price"`
	CategoryID  int     `json:"category_id" gorm:"column:category_id"`
	Image       string  `json:"image" gorm:"column:image"`
	Description string  `json:"description" gorm:"column:description"`
	Status      int     `json:"status" gorm:"column:status"` // 1-上架 0-下架
	Sort        int     `json:"sort" gorm:"column:sort"`
	CreateTime  string  `json:"create_time" gorm:"column:create_time"`
	UpdateTime  string  `json:"update_time" gorm:"column:update_time"`
}

// GetDishList 获取菜品列表（管理后台）
func GetDishList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "0"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	categoryId := c.Query("category_id")
	keyword := c.Query("keyword")
	status := c.Query("status")

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	query := dbConn.Table("dish")
	if categoryId != "" {
		query = query.Where("category_id = ?", categoryId)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		log.Errorf("统计菜品总数失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": gin.H{"list": []interface{}{}, "total": 0}})
		return
	}

	var list []Dish
	err := query.Offset(page * size).Limit(size).Order("sort ASC, id DESC").Find(&list).Error
	if err != nil {
		log.Errorf("查询菜品列表失败: %v", err)
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

// CreateDish 创建菜品
func CreateDish(c *gin.Context) {
	var req Dish
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "菜品名称不能为空"})
		return
	}
	if req.Price <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "价格必须大于0"})
		return
	}
	if req.CategoryID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "请选择分类"})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	err := dbConn.Exec(`
		INSERT INTO dish (name, price, category_id, image, description, status, sort, create_time, update_time)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, req.Name, req.Price, req.CategoryID, req.Image, req.Description, req.Status, req.Sort, now, now).Error

	if err != nil {
		log.Errorf("创建菜品失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功"})
}

// UpdateDish 更新菜品
func UpdateDish(c *gin.Context) {
	var req Dish
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}

	if req.ID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少菜品ID"})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	err := dbConn.Exec(`
		UPDATE dish SET 
			name = ?, price = ?, category_id = ?, image = ?, description = ?, 
			status = ?, sort = ?, update_time = ?
		WHERE id = ?
	`, req.Name, req.Price, req.CategoryID, req.Image, req.Description,
		req.Status, req.Sort, now, req.ID).Error

	if err != nil {
		log.Errorf("更新菜品失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteDish 删除菜品
func DeleteDish(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少菜品ID"})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	err := dbConn.Exec("DELETE FROM dish WHERE id = ?", id).Error
	if err != nil {
		log.Errorf("删除菜品失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}
