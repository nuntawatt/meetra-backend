package http

import (
	"github.com/gin-gonic/gin"
	"github.com/nuntawatt/meetra-backend/internal/delivery/http/handler"
	"github.com/nuntawatt/meetra-backend/internal/delivery/http/middleware"
	ws "github.com/nuntawatt/meetra-backend/internal/delivery/websocket"
	"github.com/nuntawatt/meetra-backend/internal/auth"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RouterConfig bundles all handler and middleware dependencies.
type RouterConfig struct {
	UserHandler   *handler.UserHandler
	EventHandler  *handler.EventHandler
	UploadHandler *handler.UploadHandler
	WSHub         *ws.Hub

	JWTService  *auth.JWTService
	RedisClient *redis.Client
	Logger      *zap.Logger
}

// NewRouter builds and returns the configured Gin engine.
func NewRouter(cfg RouterConfig) *gin.Engine {
	r := gin.New() // use gin.New (not gin.Default) so we control middleware

	// ——— Global middleware ——————————————————————————————————————————————————
	r.Use(gin.Recovery())                       // recover from panics, return 500
	r.Use(middleware.Logger(cfg.Logger))        // structured request logging
	r.Use(middleware.RateLimit(cfg.RedisClient, 100, 60)) // 100 req/min per IP globally

	// ——— Health check ———————————————————————————————————————————————————————
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// ——— Static file serving (uploaded images) ——————————————————————————————
	r.Static("/static", "./uploads")

	// ——— WebSocket (authenticated) ——————————————————————————————————————————
	wsGroup := r.Group("/ws")
	wsGroup.Use(middleware.Auth(cfg.JWTService))
	{
		wsGroup.GET("/notifications", cfg.WSHub.ServeWS)
	}

	// ——— API v1 ————————————————————————————————————————————————————————————
	v1 := r.Group("/api/v1")
	{
		// Public auth routes
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/register", cfg.UserHandler.Register)
			authGroup.POST("/login", cfg.UserHandler.Login)
		}

		// Authenticated user routes
		userGroup := v1.Group("/users")
		userGroup.Use(middleware.Auth(cfg.JWTService))
		{
			userGroup.GET("/me", cfg.UserHandler.GetProfile)
			userGroup.PATCH("/me", cfg.UserHandler.UpdateProfile)
			userGroup.DELETE("/me", cfg.UserHandler.DeleteUser)
		}

		// Event routes (list/get are public; create/join/leave require auth)
		eventGroup := v1.Group("/events")
		{
			eventGroup.GET("", cfg.EventHandler.ListEvents)
			eventGroup.GET("/:id", cfg.EventHandler.GetEvent)

			// Authenticated event actions
			authedEvents := eventGroup.Group("")
			authedEvents.Use(middleware.Auth(cfg.JWTService))
			{
				authedEvents.POST("", cfg.EventHandler.CreateEvent)
				authedEvents.POST("/:id/join", cfg.EventHandler.JoinEvent)
				authedEvents.POST("/:id/leave", cfg.EventHandler.LeaveEvent)
				authedEvents.DELETE("/:id", cfg.EventHandler.DeleteEvent) // soft delete (host only)
			}
		}

		// Upload (authenticated)
		uploadGroup := v1.Group("/upload")
		uploadGroup.Use(middleware.Auth(cfg.JWTService))
		{
			uploadGroup.POST("/image", cfg.UploadHandler.UploadImage)
		}
	}

	return r
}
