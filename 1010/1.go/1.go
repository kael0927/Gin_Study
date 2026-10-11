package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
)
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		fmt.Println("鉴权通过")
		c.Next()
	}
}

func main() {
	r := gin.Default()
	admin := r.Group("/admin", AuthMiddleware())
	{
   		admin.GET("/dashboard", func(c *gin.Context) {
   	    	c.JSON(200, gin.H{"msg": "admin dashboard"})
    	})
	}
	public := r.Group("/public")
	{
		public.GET("/public",func(c *gin.Context){
			c.JSON(200, gin.H{
				
			})
		})
	}
	r.Run()
}