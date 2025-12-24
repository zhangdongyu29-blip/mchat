package db

import (
	"log"

	"github.com/zhangdongyu29-blip/mchat/internal/config"
	"github.com/zhangdongyu29-blip/mchat/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var (
	// DB is the global database connection.
	DB *gorm.DB
)

// Connect initializes the database connection.
func Connect(cfg config.Config) error {
	var err error
	DB, err = gorm.Open(mysql.Open(cfg.MySQLDSN()), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := DB.DB()
	if err == nil {
		sqlDB.SetMaxOpenConns(10)
		sqlDB.SetMaxIdleConns(5)
	}
	log.Println("[db] connected to MySQL")
	return nil
}

// AutoMigrate migrates database schemas.
func AutoMigrate() error {
	return DB.AutoMigrate(&model.Role{}, &model.Conversation{}, &model.ChatMessage{}, &model.Memory{})
}
