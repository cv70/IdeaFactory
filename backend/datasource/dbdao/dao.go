package dbdao

import (
	"gorm.io/gorm"
)

// DB 是 gorm.DB 的类型别名，用于封装数据库操作。
type DB gorm.DB

// NewDB 创建一个新的DB实例，将*gorm.DB转换为*DB。
// 参数:
//   - db: GORM数据库实例
//
// 返回:
//   - *DB: DB实例
func NewDB(db *gorm.DB) *DB {
	return (*DB)(db)
}

// DB 方法将*DB转换回*gorm.DB，以便访问底层的GORM功能。
// 返回:
//   - *gorm.DB: GORM数据库实例
func (d *DB) DB() *gorm.DB {
	return (*gorm.DB)(d)
}
