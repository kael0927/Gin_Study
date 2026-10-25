package setting

import (
	"gopkg.in/ini.v1"
)

type Config struct {
	Server ServerConfig `ini:"server"`
	Mysql  MysqlConfig  `ini:"mysql"`
}

type ServerConfig struct {
	Port int
}

type MysqlConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DB       string
}

var Conf = new(Config)

func Init(filepath string) error {
	// 1. 正常加载文件
	cfg, err := ini.Load(filepath)
	if err != nil {
		return err
	}

	// 2. 手动、逐个取值，彻底绕开反射和BOM坑
	Conf.Server.Port = cfg.Section("server").Key("port").MustInt(9000)
	Conf.Mysql.Host = cfg.Section("mysql").Key("host").String()
	Conf.Mysql.Port = cfg.Section("mysql").Key("port").MustInt(3306)
	Conf.Mysql.User = cfg.Section("mysql").Key("user").String()
	Conf.Mysql.Password = cfg.Section("mysql").Key("password").String()
	Conf.Mysql.DB = cfg.Section("mysql").Key("db").String()

	return nil
}