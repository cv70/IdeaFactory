package user

// RegisterReq 注册请求
type RegisterReq struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterResp 注册响应
type RegisterResp struct {
	UserID string `json:"user_id"`
}

// SMSSendReq 短信发送请求
type SMSSendReq struct {
	Phone string `json:"phone" binding:"required,mobile"`
}

// SMSVerifyReq 短信验证请求
type SMSVerifyReq struct {
	Phone    string `json:"phone" binding:"required,mobile"`
	Code     string `json:"code" binding:"required"`
	DeviceID string `json:"device_id" binding:"required"`
}

// SMSVerifyResp 短信验证响应
type SMSVerifyResp struct {
	Token  string `json:"token"`
	UserID uint64 `json:"user_id,string"`
	Username string `json:"username"`
}

// LoginReq username/password login request
type LoginReq struct {
	Username string `json:"username" binding:"required,min=3,max=20"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginResp login response
type LoginResp struct {
	Token    string `json:"token"`
	UserID   uint64 `json:"user_id,string"`
	Username string `json:"username"`
}

// RefreshTokenReq token refresh request
type RefreshTokenReq struct {
	Token string `json:"token" binding:"required"`
}

// RefreshTokenResp token refresh response
type RefreshTokenResp struct {
	Token string `json:"token"`
}

// UpdateProfileReq update user profile request
type UpdateProfileReq struct {
	Email    string `json:"email" binding:"omitempty,email"`
	Nickname string `json:"nickname" binding:"omitempty,max=50"`
	Avatar   string `json:"avatar" binding:"omitempty,url"`
}

// UserProfileResp user profile response
type UserProfileResp struct {
	UserID   uint64 `json:"user_id,string"`
	Username string `json:"username"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}
