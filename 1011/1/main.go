package main

import (
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.GET("/html", func(c *gin.Context) {
		c.JSON(200, gin.H{"content":"<h1>标题</h1>"})
	})
	r.GET("/html-pure", func(c *gin.Context) {
		c.PureJSON(200, gin.H{"content":"<h1>标题</h1>"})
	})
	
	r.Run()
}
