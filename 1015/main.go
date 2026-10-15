package main

import (
    "log"

    "gorm.io/driver/mysql"
    "gorm.io/gorm"
)

type Article struct {
	ID      uint   `gorm:"primaryKey"`
    Title   string `gorm:"type:varchar(100);not null"`
    Content string `gorm:"type:text"`
   
}

type User struct {
	Name string `gorm:"primaryKey"`
	Age int `gorm:"default:0"`
}
func main() {
    dsn := "root:123456@tcp(127.0.0.1:3306)/gorm_learn?charset=utf8mb4&parseTime=True&loc=Local"
    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal("连接数据库失败: ", err)
    }
    log.Println("连接成功")
    
	if err := db.AutoMigrate(&User{},&Article{}); err != nil {
		log.Fatal("建表失败:", err)
	}
	log.Println("建表成功")
}