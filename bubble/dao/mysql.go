package dao

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"fmt"
	"bubble/setting"
)

var DB *gorm.DB

func InitMysql(c setting.MysqlConfig) (err error){
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",c.User, c.Password, c.Host, c.Port, c.DB)
	DB,err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	return err
}
