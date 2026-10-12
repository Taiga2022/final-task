package db

import (
	"github.com/Taiga2022/final-task/internal/shared/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func Open() (*gorm.DB, error) {
	mysqlConfig := config.MySQLConfig()
	mysqlConfig.Net = "tcp"
	mysqlConfig.ParseTime = true

	return gorm.Open(mysql.Open(mysqlConfig.FormatDSN()), &gorm.Config{})
}
