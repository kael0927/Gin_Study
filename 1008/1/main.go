package main

import (
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.GET("/article/:year/:month", func(c *gin.Context) {
		year := c.Param("year")
		month := c.Param("month")
		c.JSON(200, gin.H{
			"year":  year,
			"month": month,
		})
	})
	r.Run(":8080")
}
