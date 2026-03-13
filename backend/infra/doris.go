package infra

import (
	"backend/config"
	"context"
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func NewDoris(ctx context.Context, c *config.DorisConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True", c.User, c.Password, c.Host, c.DBName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	return db, err
}
