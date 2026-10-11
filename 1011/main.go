package main

import (
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func main() {
	_ = os.MkdirAll("./upload", 0755)

	r := gin.Default()
	r.Static("/static", "./static")
	r.POST("/upload", func(c *gin.Context) {
		file, err := c.FormFile("avatar")
		if err != nil {
			c.JSON(500, gin.H{"err": "获取文件失败" + err.Error()})
			return
		}
		dst := filepath.Join("./upload", filepath.Base(file.Filename))
		if err := c.SaveUploadedFile(file, dst); err != nil {
			c.JSON(500, gin.H{"err": "文件保存失败" + err.Error()})
			return
		}
		c.JSON(200, gin.H{
			"msg":      "上传成功",
			"filename": filepath.Base(file.Filename),
			"size":     file.Size,
		})
	})
	r.Run()
}
