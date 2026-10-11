package main

import (
	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(401, gin.H{"err":"未登录"})
			//c.Abort()
			//return
		}
		c.Next()
	}
}

func main() {
	r := gin.Default()
	r.Use(AuthMiddleware())
	r.GET("/profile", func(c *gin.Context){
		c.JSON(200, gin.H{
			"msg":"个人资料",
		})
	})
	r.Run(":8080")
}