package user

import (
	"context"
	"fmt"
	"testing"
)

// 测试生成唯一用户名功能
func TestGenerateUniqueUsername(t *testing.T) {
	// 创建UserDomain实例
	domain := &UserDomain{}

	// 测试用例1: 基本手机号
	ctx := context.Background()
	username, err := domain.generateUniqueUsername(ctx, "13812345678")
	if err != nil {
		t.Fatalf("Failed to generate username: %v", err)
	}

	fmt.Printf("Generated username: %s\n", username)

	// 验证用户名格式
	if len(username) < 5 {
		t.Errorf("Username should be at least 5 characters long")
	}

	// 测试用例2: 带区号的手机号
	username2, err := domain.generateUniqueUsername(ctx, "+8613812345678")
	if err != nil {
		t.Fatalf("Failed to generate username with country code: %v", err)
	}

	fmt.Printf("Generated username with country code: %s\n", username2)

	// 验证用户名格式
	if len(username2) < 5 {
		t.Errorf("Username should be at least 5 characters long")
	}

	// 测试用例3: 短手机号
	username3, err := domain.generateUniqueUsername(ctx, "1234567")
	if err != nil {
		t.Fatalf("Failed to generate username for short phone: %v", err)
	}

	fmt.Printf("Generated username for short phone: %s\n", username3)
}
