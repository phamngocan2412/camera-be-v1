package main

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	"github.com/phamngocan2412/camera-be-v1/internal/config"
	"github.com/phamngocan2412/camera-be-v1/internal/handlers"
	"github.com/phamngocan2412/camera-be-v1/internal/middleware"
	"github.com/phamngocan2412/camera-be-v1/internal/platform/db"
	"github.com/phamngocan2412/camera-be-v1/internal/platform/logger"
	"github.com/phamngocan2412/camera-be-v1/internal/repository"
	"github.com/phamngocan2412/camera-be-v1/internal/service"

	_ "github.com/phamngocan2412/camera-be-v1/docs" // swagger docs
)

// @title           Camera Security API
// @version         1.0
// @description     API documentation for Camera Security Backend
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@camerasecurity.com

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	zapLogger, err := logger.NewLogger(cfg.Log.Level)
	if err != nil {
		panic(err)
	}
	defer zapLogger.Sync()

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(logger.GinLogger(zapLogger))

	dbConn, err := db.NewDatabase(cfg.Database.URL)
	if err != nil {
		zapLogger.Fatal("database connection failed", zap.Error(err))
	}

	// Redis Initialization
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Verify Redis connection
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		zapLogger.Fatal("redis connection failed", zap.Error(err))
	}

	userRepo := repository.NewGORMUserRepository(dbConn)
	authService := service.NewAuthService(userRepo, cfg.JWT.Secret, rdb, zapLogger, cfg.SMTP)
	userService := service.NewUserService(userRepo)

	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService)

	auth := r.Group("/auth")
	{
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/forgot-password", authHandler.ForgotPassword)
		auth.POST("/reset-password", authHandler.ResetPassword)
		auth.POST("/request-otp", authHandler.RequestOTP)
		auth.POST("/verify-otp", authHandler.VerifyOTP)
		auth.POST("/verify-reset-otp", authHandler.VerifyResetOTP)
	}

	// Protected
	api := r.Group("/api")
	api.Use(middleware.JWTAuth(cfg.JWT.Secret, userRepo))
	{
		users := api.Group("/users")
		{
			users.GET("/me", userHandler.GetMe)
			users.PUT("/me", userHandler.UpdateMe)
			users.PUT("/me/password", userHandler.ChangePassword)
		}
	}

	// Swagger documentation
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	zapLogger.Info("VerifyResetOTP route registered")
	zapLogger.Info("Server starting", zap.String("port", cfg.Server.Port))
	zapLogger.Info("Swagger UI available at", zap.String("url", "http://localhost"+cfg.Server.Port+"/swagger/index.html"))
	r.Run(cfg.Server.Port)
}
