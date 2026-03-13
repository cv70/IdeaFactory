package user

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"math/big"
	"time"
)

// SendSMSCode 发送短信验证码
func (d *UserDomain) SendSMSCode(ctx context.Context, req *SMSSendReq) error {
	// 生成6位随机验证码
	code, err := generateRandomCode()
	if err != nil {
		return fmt.Errorf("failed to generate SMS code: %v", err)
	}

	// 设置过期时间为5分钟
	expireAt := time.Now().Add(5 * time.Minute)

	// 使用Redis存储验证码
	redisKey := fmt.Sprintf("sms:code:%s", req.Phone)
	if err := d.Redis.Set(ctx, redisKey, code, expireAt.Sub(time.Now())).Err(); err != nil {
		slog.Error("failed to save SMS code to redis", slog.Any("e", err))
		return fmt.Errorf("failed to send SMS code")
	}

	// 这里应该调用短信服务商API发送验证码
	// 由于实际短信服务未实现，这里仅记录日志
	slog.Info("SMS code sent", slog.String("code", code), slog.String("phone", req.Phone))

	return nil
}

// VerifySMSCode 验证短信验证码并登录/注册用户
func (d *UserDomain) VerifySMSCode(ctx context.Context, req *SMSVerifyReq) (*SMSVerifyResp, error) {
	// 从Redis获取验证码
	redisKey := fmt.Sprintf("sms:code:%s", req.Phone)
	code, err := d.Redis.Get(ctx, redisKey).Result()
	if err != nil {
		return nil, fmt.Errorf("invalid or expired SMS code")
	}

	// 验证验证码是否匹配
	if code != req.Code {
		return nil, fmt.Errorf("invalid SMS code")
	}

	// 标记验证码已使用
	if err := d.Redis.Del(ctx, redisKey).Err(); err != nil {
		slog.Error("failed to delete SMS code from redis", slog.Any("e", err))
	}

	// 检查用户是否存在
	user, err := d.DB.GetUserByPhone(req.Phone)
	if err != nil {
		// 用户不存在，创建新用户
		user, err := d.createUserByPhone(ctx, req.Phone, req.DeviceID)
		if err != nil {
			return nil, fmt.Errorf("failed to create user: %v", err)
		}

		// 生成token
		token, err := d.generateToken(ctx, user.ID, user.Username)
		if err != nil {
			return nil, fmt.Errorf("failed to generate token: %v", err)
		}

		return &SMSVerifyResp{
			Token:  token,
			UserID: user.ID,
			Username: user.Username,
		}, nil
	}

	// 用户已存在，生成token
	token, err := d.generateToken(ctx, user.ID, user.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %v", err)
	}

	return &SMSVerifyResp{
		Token:  token,
		UserID: user.ID,
	}, nil
}

// generateRandomCode 生成6位随机验证码
func generateRandomCode() (string, error) {
	// 生成0-999999之间的随机数
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}

	// 转换为6位字符串
	code := fmt.Sprintf("%06d", n.Int64())
	return code, nil
}
