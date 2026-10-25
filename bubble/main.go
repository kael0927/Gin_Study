package main

import (
	"bubble/dao"
	"bubble/models"
	"bubble/routers"
	"bubble/setting"
	"fmt"
	"os"
)

func main() {
	var err error
	// 检查有没有配置文件传进来
	if len(os.Args) < 2 {
		fmt.Println("Usage:go run main.go conf/config.ini")
		return 
	}
	// 强制读取并打印文件真实内容
	content, err := os.ReadFile(os.Args[1])
	if err != nil {
		panic("读不到文件：" + err.Error())
	}
	fmt.Printf("【文件真实内容】:\n%s\n", string(content))
	// 读取配置
	if err := setting.Init(os.Args[1]); err != nil {
		panic("读取配置失败：" + err.Error())
	}
	fmt.Printf("读到的配置: %+v\n", setting.Conf)
	// 连接数据库
	if err = dao.InitMysql(setting.Conf.Mysql); err != nil {
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
