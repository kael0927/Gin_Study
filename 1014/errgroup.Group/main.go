package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

func main() {
	// 创建两个engine 各自建立路由 一个引擎就是一个HTTP服务器 一个HTTP服务器监听一个端口
	r1 := gin.Default()
	r1.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"server": "00"})
	})
	r2 := gin.Default()
	r2.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"server": "01"})
	})
	var g errgroup.Group // 声明一个errgroup.Group类型的变量
	g.Go(func() error {
		return r1.Run(":8080")
	})
	g.Go(func() error {
		return r2.Run(":8081")
	})
	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}
}
