package main

import (
    "log"

    "gorm.io/driver/mysql"
    "gorm.io/gorm"
)

func main() {
    dsn := "root:123456@tcp(127.0.0.1:3306)/gorm_learn?charset=utf8mb4&parseTime=True&loc=Local"
    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
    if err != nil {
        log.Fatal("连接数据库失败: ", err)
    }
    log.Println("连接成功")
    _ = db
}