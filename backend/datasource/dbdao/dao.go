package dbdao

import (
	"time"

	"gorm.io/gorm"
)

type DB gorm.DB

func NewDB(db *gorm.DB) *DB {
	return (*DB)(db)
}

func (d *DB) DB() *gorm.DB {
	return (*gorm.DB)(d)
}

type BaseModel struct {
	ID        int64 `gorm:"primarykey"`
	UpdatedAt time.Time
}
