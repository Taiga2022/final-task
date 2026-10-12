package config

import (
	"net"
	"os"

	drivermysql "github.com/go-sql-driver/mysql"
)

func MySQLConfig() *drivermysql.Config {
	config := drivermysql.NewConfig()
	config.User = envOrDefault("DB_USERNAME", "user")
	config.Passwd = envOrDefault("DB_PASSWORD", "password")
	config.Addr = net.JoinHostPort(envOrDefault("DB_HOST", "127.0.0.1"), envOrDefault("DB_PORT", "3306"))
	config.DBName = envOrDefault("DB_DATABASE", "go-db")
	return config
}

func envOrDefault(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
