package pkg

import (
	"blog/config"
	"blog/model"
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type MysqlDatabase struct {
	DB *gorm.DB
}

func InitMysql_or_sqlite(cfg *config.DatabaseConfig) (*MysqlDatabase, error) {
	var db *gorm.DB
	var err error
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Printf("MySQL彼得帕克了%v", err)
		log.Println("但愿SQlite不会彼得帕克")
		db, err = gorm.Open(sqlite.Open("database.db"), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("SQlite也失败的man了%v", err)
		}
		log.Println("SQlite成功了")
	} else {
		log.Println("成功连接到MySQl")
	}
	if err := model.AutoMigrate(db); err != nil {
		return nil, err
	}
	log.Println("成功自动建表")
	return &MysqlDatabase{DB: db}, nil
}
