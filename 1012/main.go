package main

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/cors"
)

func main() {
	r := gin.Default()
	r.Use(cors.Default())
	r.LoadHTMLGlob("1012/template/*")
	r.GET("/index", func(c *gin.Context) {
		c.HTML(200, "index.html", gin.H{
			"Title":   "首页",
			"Content": "欢迎来到我的首页",
			"Time":	   "2025-2-2",
		})
	})
	r.Static("/static","1012/static")
	
	r.Run(":8080")
}
