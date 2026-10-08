package main

import "github.com/gin-gonic/gin"

type Userform struct {
	Username string `form:"username" binding:"required,min=3,max=20"`
	Email    string `form:"email" binding:"required,email"`
	Age      int    `form:"age" binding:"required,gt=0"`
}

func main() {

	r := gin.Default()
	r.POST("/register", func(c *gin.Context) {
		var form Userform
		if err := c.ShouldBind(&form); err != nil {
			c.JSON(200, gin.H{"err": err.Error()})
			return
		}
		c.JSON(200, gin.H{
			"username": form.Username,
			"email":    form.Email,
			"age":      form.Age,
		})
	})
	r.Run()
}
