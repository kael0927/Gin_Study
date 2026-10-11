package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

func LogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		nowPath := c.Request.URL.Path
		fmt.Println("请求开始：" + nowPath)
		c.Next()
		fmt.Println("请求结束：" + nowPath)
	}
}

func main() {
	r := gin.Default()
	r.Use(LogMiddleware())
	r.GET("/test", func(c *gin.Context) {
		fmt.Println("我是处理函数")
		c.JSON(200, gin.H{
			"msg": "ok",
		})
	})
	r.Run(":8080")
}
