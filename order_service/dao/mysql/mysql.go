package mysql

import (
	"fmt"
	"order_service/config"
	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var db *gorm.DB

func Init(cfg *config.MySQLConfig) (err error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DbName)
	db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}

	sqlBD, err := db.DB()
	if err != nil {
		return
	}

	sqlBD.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlBD.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlBD.SetConnMaxLifetime(time.Hour)
	zap.L().Info("Init MySQL Success!")
	return
}
