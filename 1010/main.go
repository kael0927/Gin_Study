package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()
	api := r.Group("/api/v1")
	{
		api.GET("/users", func(c *gin.Context) {
			c.JSON(200, gin.H{"msg": "用户列表"})
		})
		api.GET("/articles", func(c *gin.Context) {
			c.JSON(200, gin.H{"msg": "文章列表"})
		})
	}
	r.Run()
}
