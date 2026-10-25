package controller

import (
	"bubble/dao"
	"bubble/models"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

/*
	url     --> controller  --> logic   -->    model

请求来了  -->  控制器      --> 业务逻辑  --> 模型层的增删改查
*/
func GetTodolist(c *gin.Context) {
	var todolist []models.Todo
	if err := dao.DB.Find(&todolist).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"err": err.Error()})
		return
	}
	c.JSON(http.StatusOK, todolist)
}

func CreateTodo(c *gin.Context) {
	var newTodo models.Todo
	if err := c.ShouldBindJSON(&newTodo); err != nil {
		c.JSON(400, gin.H{"err": err.Error()})
		return
	}
	if err := dao.DB.Create(&newTodo).Error; err != nil {
		c.JSON(500, gin.H{"err": err.Error()})
		return
	}
	c.JSON(200, newTodo)
}

func UpdateTodo(c *gin.Context) {
	idStr := c.Param("id") // 获取路径参数
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(400, gin.H{"err": "ID格式错误"})
		return
	}
	var todo models.Todo
	if err := dao.DB.First(&todo, id).Error; err != nil { // 查找表中对应数据并装进todo
		c.JSON(400, gin.H{"err": "未找到该待办"})
		return
	}
	// 新数据覆盖旧数据
	if err := c.ShouldBindJSON(&todo); err != nil {
		c.JSON(400, gin.H{"err": err.Error()})
		return
	}
	// 保存修改
	if err := dao.DB.Save(&todo).Error; err != nil {
		c.JSON(400, gin.H{"err": err.Error()})
		return
	}
	c.JSON(200, todo)
}

func DeleteTodo(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	if err := dao.DB.Where("id = ?", id).Delete(&models.Todo{}).Error; err != nil {
		c.JSON(500, gin.H{"err": err.Error()})
		return
	}
	c.JSON(200, gin.H{"msg": "删除成功"})
}
