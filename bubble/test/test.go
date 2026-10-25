package main

import (
	"fmt"
	"os"
	"gopkg.in/ini.v1"
)

func main() {
	// 1. 打印当前终端所在的绝对目录
	pwd, _ := os.Getwd()
	fmt.Println("当前终端目录:", pwd)

	// 2. 强行使用绝对路径加载
	absPath := "E:/Jt_lin/Gin-study/bubble/conf/config.ini" // 👈 请确保这个路径和你的实际文件一致
	fmt.Println("尝试加载:", absPath)

	cfg, err := ini.Load(absPath)
	if err != nil {
		fmt.Println("加载失败:", err)
		return
	}
	
	fmt.Println("Host:", cfg.Section("mysql").Key("host").String())
	fmt.Println("Port:", cfg.Section("mysql").Key("port").MustInt())
}