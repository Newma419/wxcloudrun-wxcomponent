// api/admin/category.go
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
// 分类管理（管理后台）
// ============================================================

type Category struct {
	ID   int    `json:"id" gorm:"column:id"`
	Name string `json:"name" gorm:"column:name"`
	Sort int    `json:"sort" gorm:"column:sort"`
	Icon string `json:"icon" gorm:"column:icon"`
}

// GetCategoryListAdmin 获取分类列表（管理后台）
func GetCategoryListAdmin(c *gin.Context) {
	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	var list []Category
	err := dbConn.Table("dish_category").Order("sort ASC, id ASC").Find(&list).Error
	if err != nil {
		log.Errorf("查询分类列表失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 0, "data": []interface{}{}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "data": list})
}

// CreateCategory 创建分类
func CreateCategory(c *gin.Context) {
	var req Category
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "分类名称不能为空"})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	err := dbConn.Exec(`
		INSERT INTO dish_category (name, sort, icon, create_time)
		VALUES (?, ?, ?, ?)
	`, req.Name, req.Sort, req.Icon, time.Now().Format("2006-01-02 15:04:05")).Error

	if err != nil {
		log.Errorf("创建分类失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "创建成功"})
}

// UpdateCategory 更新分类
func UpdateCategory(c *gin.Context) {
	var req Category
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "参数错误"})
		return
	}
	if req.ID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少分类ID"})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	err := dbConn.Exec(`
		UPDATE dish_category SET name = ?, sort = ?, icon = ? WHERE id = ?
	`, req.Name, req.Sort, req.Icon, req.ID).Error

	if err != nil {
		log.Errorf("更新分类失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功"})
}

// DeleteCategory 删除分类
func DeleteCategory(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "缺少分类ID"})
		return
	}

	dbConn := db.Get()
	if dbConn == nil {
		log.Error("数据库连接为空")
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "系统错误"})
		return
	}

	// 先检查该分类下是否有菜品
	var count int64
	dbConn.Table("dish").Where("category_id = ?", id).Count(&count)
	if count > 0 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "该分类下存在菜品，无法删除"})
		return
	}

	err := dbConn.Exec("DELETE FROM dish_category WHERE id = ?", id).Error
	if err != nil {
		log.Errorf("删除分类失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}
