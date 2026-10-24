package routers

import (
	"bubble/controller"
	"github.com/gin-gonic/gin"
)

func SetupRouter() *gin.Engine {
	r := gin.Default()
	r.GET("/todo", controller.GetTodolist)
	r.POST("/todo", controller.CreateTodo)
	r.PUT("/todo/:id", controller.UpdateTodo)
	r.DELETE("/todo/:id", controller.DeleteTodo)
	return r
}
