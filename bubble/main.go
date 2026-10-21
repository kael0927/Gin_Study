package main

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Todo struct {
	ID int `json:id`
	Title string `json:title`
	Status bool `json:status`
}

var db *gorm.DB

func main() {
	var err error
	dsn := "root:123456@tcp(127.0.0.1:3306)/bubble?charset=utf8mb4&parseTime=True&loc=Local"
	db,err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("数据库连接失败:" + err.Error())
	}
	// 自动建表
	err = db.AutoMigrate(&Todo{})
	if err != nil {
		panic("建表失败：" + err.Error())
	}
	// 创建Gin引擎
	r := gin.Default()
	r.GET("/todo", func(c *gin.Context){
		var todolist []Todo
		if err = db.Find(&todolist).Error; err != nil {
			c.JSON(http.StatusOK,gin.H{"err":err.Error()})
		}else{
			c.JSON(http.StatusOK,todolist)
		}
	})
	r.Run(":9000")

}