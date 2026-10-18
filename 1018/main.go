package main

import (
	"log"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Article struct {
	ID      uint   `gorm:"primaryKey;autoIncrement"`
	Title   string
	Content string
}

func main() {
	dsn := "root:123456@tcp(127.0.0.1:3306)/gorm_learn?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("连接数据库失败: ", err)
	}
	log.Println("连接成功")

	// 自动迁移：没有表就创建，已有表不会删数据
	err = db.AutoMigrate(&Article{})
	if err != nil {
		log.Fatal("自动建表失败：", err)
	}

	a1 := Article{
		Title:   "GORM实战",
		Content: "内容",
	}
	//if err := db.Create(&a1).Error; err != nil {
	//	log.Fatal("插入失败：", err)
	//}
	//log.Println("插入成功，文章ID：", a1.ID)
	if err := db.Delete(&a1,1).Error;err != nil {
		log.Fatal("删除失败：",err)
	}
}
