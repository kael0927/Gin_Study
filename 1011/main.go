package main

import (
	_ "net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	_ = os.MkdirAll("./uploads", 0755)

	r := gin.Default()

	// ★ 实验 1 的关键：把 static 目录挂到 /static 路径下
	r.Static("/static", "./static")
	r.Run()
}
