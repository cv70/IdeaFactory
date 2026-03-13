package dbdao

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"time"
)

// User 用户实体
type User struct {
	ID         uint64    `gorm:"column:id;primaryKey" json:"id"`
	Phone      string    `gorm:"column:phone;uniqueIndex" json:"phone"`
	Username   string    `gorm:"column:username;uniqueIndex" json:"username"`
	Password   string    `gorm:"column:password" json:"-"` // 不返回密码
	Email      string    `gorm:"column:email" json:"email"` // 用户邮箱（可选）
	Nickname   string    `gorm:"column:nickname" json:"nickname"` // 用户显示名（可选）
	Avatar     string    `gorm:"column:avatar" json:"avatar"` // 用户头像URL（可选）
	DeviceID   string    `gorm:"column:device_id" json:"device_id"`
	Role       int8      `gorm:"column:role" json:"role"` // 角色 0: 普通用户 1: 投资人/投资机构 2: 创业者
	Industry   []string  `gorm:"column:industry" json:"industry"` // 关注行业
	Introduction string  `gorm:"column:introduction" json:"introduction"` // 自我介绍
	IsVerified bool      `gorm:"column:is_verified" json:"is_verified"`
	CreatedAt  time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (d *DB) GetUserByPhone(phone string) (*User, error) {
	var user User
	err := d.DB().Where("phone = ?", phone).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errors.New("user not found")
	}
	return &user, nil
}

func (d *DB) GetUsers(ids ...uuid.UUID) (*User, error) {
	var user User
	result := d.DB().Where("id IN ?", ids).First(&user)
	if result.Error != nil {
		return nil, errors.New("user not found")
	}
	return &user, nil
}

// ExistUsername 检查用户名是否存在
func (d *DB) ExistUsername(username string) (bool, error) {
	var exists bool
	result := d.DB().Raw("SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)", username).Scan(&exists)
	if result.Error != nil {
		return false, result.Error
	}
	return exists, nil
}

func (d *DB) CreateUser(user *User) error {
	if user == nil {
		return errors.New("user is nil")
	}
	result := d.DB().Create(user)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("failed to insert user")
	}
	return nil
}

// GetUserByUsername retrieves a user by username
func (d *DB) GetUserByUsername(username string) (*User, error) {
	var user User
	result := d.DB().Where("username = ?", username).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &user, nil
}

// UpdateUserProfile updates user profile information
func (d *DB) UpdateUserProfile(userID uint64, updates map[string]interface{}) (*User, error) {
	var user User
	result := d.DB().Model(&user).Where("id = ?", userID).Updates(updates).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return &user, nil
}

// EmailExists checks if an email already exists
func (d *DB) EmailExists(email string) (bool, error) {
	var count int64
	result := d.DB().Where("email = ?", email).Model(&User{}).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}

// GetUser retrieves a user by ID
func (d *DB) GetUser(userID uint64) (*User, error) {
	var user User
	result := d.DB().Where("id = ?", userID).First(&user)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, result.Error
		}
		return nil, result.Error
	}
	return &user, nil
}
