package main

import (
	"bubble/dao"
	"bubble/models"
	"bubble/routers"
)

func main() {
	var err error
	// 连接数据库
	if err = dao.InitMysql(); err != nil {
		panic("连接数据库失败：" + err.Error())
	}
	// 自动建表
	err = dao.DB.AutoMigrate(&models.Todo{})
	if err != nil {
		panic("建表失败：" + err.Error())
	}
	// 创建Gin引擎
	r := routers.SetupRouter()
	r.Run(":9000")
}
