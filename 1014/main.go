package main

import (
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.GET("/user", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"msg": "用户信息",
		})
	})
	r.NoMethod(func(c *gin.Context){
		c.JSON(405, gin.H{
			"msg":"方法不允许",
		})
	})
	r.NoRoute(func(c *gin.Context){
		c.JSON(404, gin.H{
			"msg":"页面不存在",
		})
	})
	r.Run(":8080")
}
