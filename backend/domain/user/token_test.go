package user

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestGenerateToken(t *testing.T) {
	// 创建一个测试的UserDomain实例
	domain := &UserDomain{
		// 注意：在实际测试中，你需要提供真实的DB和Redis连接
	}

	// 测试生成token
	testUserID := uint64(123)
	tokenString, err := domain.generateToken(context.Background(), testUserID, "test_username")
	assert.NoError(t, err)
	assert.NotEmpty(t, tokenString)

	// 解析token验证其有效性
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 使用相同的密钥解析
		return []byte("your-secret-key-here"), nil
	})
	assert.NoError(t, err)
	assert.True(t, token.Valid)

	// 验证claims
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		assert.Equal(t, float64(testUserID), claims["user_id"])
		assert.Equal(t, "test_username", claims["username"])
		assert.NotNil(t, claims["exp"])
		assert.NotNil(t, claims["iat"])

		// 验证过期时间是否在未来
		exp, ok := claims["exp"].(float64)
		assert.True(t, ok)
		assert.Greater(t, exp, float64(time.Now().Unix()))
	}
}
