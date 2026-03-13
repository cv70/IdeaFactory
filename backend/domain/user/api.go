package user

import (
	"backend/utils"
	"log/slog"

	"github.com/gin-gonic/gin"
)

// ApiSendSMSCode 发送短信验证码
func (d *UserDomain) ApiSendSMSCode(c *gin.Context) {
	var req SMSSendReq
	err := c.ShouldBind(&req)
	if err != nil {
		slog.Error("failed to parse body", slog.Any("e", err))
		utils.RespError(c, 400, "failed to parse body")
		return
	}

	err = d.SendSMSCode(c, &req)
	if err != nil {
		slog.Error("failed to send SMS code", slog.Any("e", err))
		utils.RespError(c, 500, "failed to send SMS code")
		return
	}

	utils.RespSuccess(c, gin.H{"message": "SMS code sent successfully"})
}

// ApiVerifySMSCode 验证短信验证码并登录
func (d *UserDomain) ApiVerifySMSCode(c *gin.Context) {
	var req SMSVerifyReq
	err := c.ShouldBind(&req)
	if err != nil {
		slog.Error("failed to parse body", slog.Any("e", err))
		utils.RespError(c, 400, "failed to parse body")
		return
	}

	resp, err := d.VerifySMSCode(c, &req)
	if err != nil {
		slog.Error("failed to verify SMS code", slog.Any("e", err))
		utils.RespError(c, 400, "invalid SMS code")
		return
	}

	utils.RespSuccess(c, resp)
}

// ApiRegister handles user registration with username and password
func (d *UserDomain) ApiRegister(c *gin.Context) {
	var req RegisterReq
	err := c.ShouldBind(&req)
	if err != nil {
		slog.Error("failed to parse register request", slog.Any("e", err))
		utils.RespError(c, 400, "failed to parse request")
		return
	}

	user, err := d.RegisterUser(c, req.Username, req.Password, "")
	if err != nil {
		slog.Error("failed to register user", slog.Any("e", err))
		utils.RespError(c, 400, err.Error())
		return
	}

	resp := RegisterResp{
		UserID: user.Username,
	}
	utils.RespSuccess(c, resp)
}

// ApiLogin handles user login with username and password
func (d *UserDomain) ApiLogin(c *gin.Context) {
	var req LoginReq
	err := c.ShouldBind(&req)
	if err != nil {
		slog.Error("failed to parse login request", slog.Any("e", err))
		utils.RespError(c, 400, "failed to parse request")
		return
	}

	user, err := d.LoginUser(c, req.Username, req.Password)
	if err != nil {
		slog.Error("failed to login user", slog.Any("e", err))
		utils.RespError(c, 401, "invalid username or password")
		return
	}

	token, err := d.generateToken(c, user.ID, user.Username)
	if err != nil {
		slog.Error("failed to generate token", slog.Any("e", err))
		utils.RespError(c, 500, "failed to generate token")
		return
	}

	resp := LoginResp{
		Token:    token,
		UserID:   user.ID,
		Username: user.Username,
	}
	utils.RespSuccess(c, resp)
}

// ApiRefreshToken handles token refresh
func (d *UserDomain) ApiRefreshToken(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.RespError(c, 401, "unauthorized")
		return
	}

	token, err := d.RefreshToken(c, userID)
	if err != nil {
		slog.Error("failed to refresh token", slog.Any("e", err))
		utils.RespError(c, 500, "failed to refresh token")
		return
	}

	resp := RefreshTokenResp{Token: token}
	utils.RespSuccess(c, resp)
}

// ApiLogout handles user logout
func (d *UserDomain) ApiLogout(c *gin.Context) {
	// In a real application, you might want to blacklist the token
	// For now, we just return success since the client will discard the token
	utils.RespSuccess(c, gin.H{"message": "logged out successfully"})
}

// ApiGetProfile retrieves user profile
func (d *UserDomain) ApiGetProfile(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.RespError(c, 401, "unauthorized")
		return
	}

	user, err := d.DB.GetUser(userID)
	if err != nil {
		slog.Error("failed to get user", slog.Any("e", err))
		utils.RespError(c, 500, "failed to get user")
		return
	}

	resp := UserProfileResp{
		UserID:   user.ID,
		Username: user.Username,
		Phone:    user.Phone,
		Email:    user.Email,
		Nickname: user.Nickname,
		Avatar:   user.Avatar,
	}
	utils.RespSuccess(c, resp)
}

// ApiUpdateProfile updates user profile
func (d *UserDomain) ApiUpdateProfile(c *gin.Context) {
	userID, ok := GetUserID(c)
	if !ok {
		utils.RespError(c, 401, "unauthorized")
		return
	}

	var req UpdateProfileReq
	err := c.ShouldBind(&req)
	if err != nil {
		slog.Error("failed to parse update profile request", slog.Any("e", err))
		utils.RespError(c, 400, "failed to parse request")
		return
	}

	// Check if email is already in use
	if req.Email != "" {
		exists, err := d.DB.EmailExists(req.Email)
		if err == nil && exists {
			utils.RespError(c, 400, "email already in use")
			return
		}
	}

	updates := make(map[string]interface{})
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.Nickname != "" {
		updates["nickname"] = req.Nickname
	}
	if req.Avatar != "" {
		updates["avatar"] = req.Avatar
	}

	user, err := d.DB.UpdateUserProfile(userID, updates)
	if err != nil {
		slog.Error("failed to update user profile", slog.Any("e", err))
		utils.RespError(c, 500, "failed to update profile")
		return
	}

	resp := UserProfileResp{
		UserID:   user.ID,
		Username: user.Username,
		Phone:    user.Phone,
		Email:    user.Email,
		Nickname: user.Nickname,
		Avatar:   user.Avatar,
	}
	utils.RespSuccess(c, resp)
}
