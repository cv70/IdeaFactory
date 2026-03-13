package user

import (
	"backend/config"
	"backend/datasource/dbdao"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/cv70/pkgo/gstr"

	"github.com/golang-jwt/jwt/v5"
)

// createUserByPhone 根据手机号创建用户
func (d *UserDomain) createUserByPhone(ctx context.Context, phone, deviceID string) (*dbdao.User, error) {
	// 生成唯一用户名
	username, err := d.generateUniqueUsername(ctx, phone)
	if err != nil {
		return nil, fmt.Errorf("failed to generate username: %v", err)
	}

	// 创建用户记录
	user := dbdao.User{
		Phone:      phone,
		Username:   username,
		DeviceID:   deviceID,
		IsVerified: true,
	}

	err = d.DB.CreateUser(&user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
	}

	return &user, nil
}

// generateUniqueUsername 生成唯一用户名
func (d *UserDomain) generateUniqueUsername(ctx context.Context, phone string) (string, error) {
	// 基于手机号生成基础用户名（去除区号）
	baseUsername := strings.TrimPrefix(phone, "+86")

	// 如果手机号长度大于7位，取后7位作为用户名基础
	if len(baseUsername) > 7 {
		baseUsername = baseUsername[len(baseUsername)-7:]
	}

	// 添加随机数确保唯一性
	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)
	randomNum := rng.Intn(1000)

	// 组合用户名
	username := fmt.Sprintf("%s_%03d", baseUsername, randomNum)

	// 检查用户名是否已存在，如果存在则重新生成
	for i := 0; i < 10; i++ { // 最多重试10次
		existUser, err := d.DB.ExistUsername(username)
		if err == nil || !existUser {
			// 用户不存在，用户名可用
			break
		}
		// 用户已存在，生成新的用户名
		randomNum = rng.Intn(1000)
		username = fmt.Sprintf("%s_%03d", baseUsername, randomNum)
	}

	return username, nil
}

// generateToken 生成JWT token
func (d *UserDomain) generateToken(ctx context.Context, userID uint64, username string) (string, error) {
	// 从配置中获取JWT密钥
	cfg := config.GetConfig()
	if cfg.JWT == nil || cfg.JWT.SecretKey == "" {
		return "", errors.New("JWT secret key not configured")
	}
	secretKey := gstr.StringToBytes(cfg.JWT.SecretKey)

	now := time.Now()
	// 创建token claims
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"exp":      now.Add(time.Hour * 24).Unix(), // 24小时过期
		"iat":      now.Unix(),
	}

	// 创建token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// 签名token
	tokenString, err := token.SignedString(secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %v", err)
	}

	return tokenString, nil
}

// RegisterUser creates a new user with username and password
func (d *UserDomain) RegisterUser(ctx context.Context, username, password, email string) (*dbdao.User, error) {
	// Check if username already exists
	_, err := d.DB.ExistUsername(username)
	if err == nil {
		return nil, fmt.Errorf("username already exists")
	}

	// Hash the password
	hashedPassword, err := HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %v", err)
	}

	// Create user record
	user := dbdao.User{
		Username:   username,
		Password:   hashedPassword,
		Email:      email,
		IsVerified: true,
	}

	err = d.DB.CreateUser(&user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %v", err)
	}

	return &user, nil
}

// LoginUser authenticates user with username and password
func (d *UserDomain) LoginUser(ctx context.Context, username, password string) (*dbdao.User, error) {
	// Get user by username
	user, err := d.DB.GetUserByUsername(username)
	if err != nil {
		return nil, fmt.Errorf("invalid username or password")
	}

	// Verify password
	if !VerifyPassword(user.Password, password) {
		return nil, fmt.Errorf("invalid username or password")
	}

	return user, nil
}

// RefreshToken generates a new token for an existing user
func (d *UserDomain) RefreshToken(ctx context.Context, userID uint64) (string, error) {
	// Get user from database to ensure they still exist
	user, err := d.DB.GetUser(userID)
	if err != nil {
		return "", fmt.Errorf("user not found")
	}

	// Generate new token
	token, err := d.generateToken(ctx, user.ID, user.Username)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %v", err)
	}

	return token, nil
}

// ValidateToken validates a JWT token and returns user ID and username
func ValidateToken(tokenString string) (uint64, string, error) {
	cfg := config.GetConfig()
	if cfg.JWT == nil || cfg.JWT.SecretKey == "" {
		return 0, "", errors.New("JWT secret key not configured")
	}
	secretKey := gstr.StringToBytes(cfg.JWT.SecretKey)

	token, err := jwt.ParseWithClaims(tokenString, jwt.MapClaims{}, func(token *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})

	if err != nil {
		return 0, "", fmt.Errorf("invalid token: %v", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return 0, "", errors.New("invalid token claims")
	}

	userID, ok := claims["user_id"].(float64)
	if !ok {
		return 0, "", errors.New("missing user_id in token")
	}

	username, ok := claims["username"].(string)
	if !ok {
		return 0, "", errors.New("missing username in token")
	}

	return uint64(userID), username, nil
}
