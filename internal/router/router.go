package router

import (
	"net/http"
	"time"

	"carenav_backend_go/docs"
	"carenav_backend_go/internal/handlers"
	"carenav_backend_go/internal/middleware"

	"github.com/gin-gonic/gin"
	swaggerfiles "github.com/swaggo/files"
	ginswagger "github.com/swaggo/gin-swagger"
)

// New wires up the same routes as src/index.ts (minus /consultations,
// which is deliberately not ported yet).
func New(auth *handlers.AuthHandler, user *handlers.UserHandler) *gin.Engine {
	r := gin.Default()

	r.Use(middleware.SecureHeaders())
	r.Use(middleware.RateLimiter(100, 15*time.Minute))

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "Welcome to my server")
	})

	docs.SwaggerInfo.BasePath = "/"
	r.GET("/swagger/*any", ginswagger.WrapHandler(swaggerfiles.Handler))

	r.POST("/register", auth.RegisterUser)
	r.POST("/verify-registration-Otp", auth.VerifyRegistrationOtp)
	r.POST("/login", auth.LoginUser)
	r.POST("/forgot-password", auth.ForgotPassword)
	r.POST("/verify-otp", auth.VerifyOtp)
	r.POST("/reset-password", auth.ResetPassword)
	r.POST("/create-pin", auth.CreatePin)
	r.POST("/verify-pin", auth.VerifyPin)
	r.POST("/forgot-pin", auth.ForgotPin)
	r.POST("/reset-pin", auth.ResetPin)
	r.POST("/create-doctor", auth.CreateDoctor)
	r.POST("/doctor-login", auth.LoginDoctor)

	// NOTE: matches src/routers/user.routes.ts exactly — these routes have
	// no auth middleware in the original Node app (only /consultations
	// does). Not adding auth here to avoid a silent behavior change; add
	// middleware.AuthenticateToken(...) if/when that's deliberately fixed.
	usersGroup := r.Group("/api/users")
	usersGroup.GET("/:id", user.GetUser)
	usersGroup.DELETE("/:id", user.DeleteUser)

	return r
}
