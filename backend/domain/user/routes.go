package user

import "github.com/gin-gonic/gin"

// RegisterRoutes registers all user-related routes
func RegisterRoutes(router *gin.RouterGroup, domain *UserDomain) {
	// Public routes (no authentication required)
	public := router.Group("/users")
	{
		public.POST("/register", domain.ApiRegister)
		public.POST("/login", domain.ApiLogin)
		public.POST("/sms/send", domain.ApiSendSMSCode)
		public.POST("/sms/verify", domain.ApiVerifySMSCode)
	}

	// Protected routes (authentication required)
	protected := router.Group("/users")
	protected.Use(AuthMiddleware())
	{
		protected.POST("/logout", domain.ApiLogout)
		protected.POST("/refresh-token", domain.ApiRefreshToken)
		protected.GET("/profile", domain.ApiGetProfile)
		protected.PUT("/profile", domain.ApiUpdateProfile)
	}
}
