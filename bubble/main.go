package main

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"strconv"
	"bubble/models"
)



var DB *gorm.DB

func main() {
	var err error
	dsn := "root:123456@tcp(127.0.0.1:3306)/bubble?charset=utf8mb4&parseTime=True&loc=Local"
	DB,err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("数据库连接失败:" + err.Error())
	}
	// 自动建表
	err = DB.AutoMigrate(&models.Todo{})
	if err != nil {
		panic("建表失败：" + err.Error())
	}
	// 创建Gin引擎
	r := gin.Default()
	r.GET("/todo", func(c *gin.Context){
		var todolist []models.Todo
		if err = DB.Find(&todolist).Error; err != nil {
			c.JSON(http.StatusInternalServerError,gin.H{"err":err.Error()})
			return
		}else{
			c.JSON(http.StatusOK,todolist)
		}
	})
	r.POST("/todo", func(c *gin.Context){
		var newTodo models.Todo
		if err = c.ShouldBindJSON(&newTodo); err != nil {
			c.JSON(400, gin.H{"err": err.Error(),})
			return
		}
		if err = DB.Create(&newTodo).Error; err != nil{	
			c.JSON(500, gin.H{"err": err.Error()})	
			return
		}				
		c.JSON(200, newTodo)
	})
	r.PUT("/todo/:id", func(c *gin.Context){
		idStr := c.Param("id")// 获取路径参数
		id,err := strconv.Atoi(idStr); 
		if err != nil {
			c.JSON(400, gin.H{"err":"ID格式错误"})
			return
		}
		var todo models.Todo
		if err := DB.First(&todo, id).Error; err != nil { // 查找表中对应数据并装进todo
			c.JSON(400, gin.H{"err":"未找到该待办"})
			return
		}
		// 新数据覆盖旧数据
		if err := c.ShouldBindJSON(&todo); err != nil {
			c.JSON(400, gin.H{"err": err.Error()})
			return
		}
		// 保存修改
		if err := DB.Save(&todo).Error; err != nil {
			c.JSON(400, gin.H{"err": err.Error()})
			return
		}
		c.JSON(200, todo)
	})
	r.DELETE("/todo/:id", func(c *gin.Context){
		// 获取ID
		idStr := c.Param("id") 
		id,_ := strconv.Atoi(idStr)
		// 直接执行行删除 Where中指定 ID 
		if err := DB.Where("id = ?",id).Delete(&models.Todo{}).Error; err != nil {
			c.JSON(500, gin.H{"err":err.Error()})
			return
		}
		c.JSON(200, gin.H{"msg":"删除成功"})
	})
	r.Run(":9000")
}