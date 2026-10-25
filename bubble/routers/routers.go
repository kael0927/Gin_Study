package routers

import (
	"bubble/controller"
	"net/http"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()

	r.LoadHTMLGlob("templates/*")
	r.Static("/static", "./static")

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	// 👇 核心修改：加一个 /v1 的路由组
	v1 := r.Group("/v1")
	{
		v1.GET("/todo", controller.GetTodolist)
		v1.POST("/todo", controller.CreateTodo)
		v1.PUT("/todo/:id", controller.UpdateTodo)
		v1.DELETE("/todo/:id", controller.DeleteTodo)
	}

	return r
}