package main

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
)

//自定义的Writer 用来偷看响应体
type bodyLogWriter struct {
	body *bytes.Buffer	// 记录用的缓冲区
	gin.ResponseWriter	// 嵌入原始Writer
}
// 重写Write方法：先记录 再转发
func (w *bodyLogWriter) Write(b []byte) (int,error) {
	w.body.Write(b)						// 记录到内存
	return w.ResponseWriter.Write(b)	// 真正写给浏览器
}
// 替换c.Write
func BodyLogMiddleware(c *gin.Context) {
	blw := &bodyLogWriter{
		body: bytes.NewBuffer([]byte{}),
		ResponseWriter: c.Writer,
	}
	c.Writer = blw // 替换 
	c.Next()
	// 处理函数执行完 读buffer
	fmt.Println("响应体：",blw.body.String())
}
func main() {
    r := gin.Default()
    r.Use(BodyLogMiddleware)

    r.GET("/test", func(c *gin.Context) {
        c.JSON(200, gin.H{"msg": "hello"})
    })

    r.Run(":8080")
}