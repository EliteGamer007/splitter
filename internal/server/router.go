package server

import (
	"splitter/internal/config"
	"splitter/internal/handlers"
	"splitter/internal/middleware"
	"splitter/internal/repository"
	"splitter/internal/service"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
)

// Server wraps the Echo server
type Server struct {
	echo *echo.Echo
	cfg  *config.Config
}

// NewServer creates a new server instance
func NewServer(cfg *config.Config) *Server {
	e := echo.New()

	// Initialize services
	// Use current working directory + uploads for local storage
	// in production this would come from config
	storageService := service.NewLocalStorage(".", cfg.Server.BaseURL)

	// Initialize repositories
	userRepo := repository.NewUserRepository()
	postRepo := repository.NewPostRepository()
	followRepo := repository.NewFollowRepository()
	interactionRepo := repository.NewInteractionRepository()
	messageRepo := repository.NewMessageRepository()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userRepo, cfg.JWT.Secret)
	userHandler := handlers.NewUserHandler(userRepo)
	postHandler := handlers.NewPostHandler(postRepo, storageService)
	followHandler := handlers.NewFollowHandler(followRepo, userRepo)
	interactionHandler := handlers.NewInteractionHandler(interactionRepo, userRepo)
	adminHandler := handlers.NewAdminHandler(userRepo)
	messageHandler := handlers.NewMessageHandler(messageRepo, userRepo)

	// Global middleware
	e.Use(echomiddleware.Logger())
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.BodyLimit("6M")) // Limit body size to 6MB (allow overhead for 5MB file)
	e.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:3000", "http://127.0.0.1:3000", "http://localhost:3001", "http://127.0.0.1:3001"},
		AllowMethods:     []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.OPTIONS},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization},
		AllowCredentials: true,
	}))

	// Static routes
	e.Static("/uploads", "uploads")

	// Routes
	setupRoutes(e, cfg, authHandler, userHandler, postHandler, followHandler, interactionHandler, adminHandler, messageHandler)

	return &Server{
		echo: e,
		cfg:  cfg,
	}
}

// setupRoutes registers all application routes
func setupRoutes(
	e *echo.Echo,
	cfg *config.Config,
	authHandler *handlers.AuthHandler,
	userHandler *handlers.UserHandler,
	postHandler *handlers.PostHandler,
	followHandler *handlers.FollowHandler,
	interactionHandler *handlers.InteractionHandler,
	adminHandler *handlers.AdminHandler,
	messageHandler *handlers.MessageHandler,
) {
	// API v1 group
	api := e.Group("/api/v1")

	// Health check
	api.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{
			"status": "ok",
		})
	})

	// Auth routes (public)
	auth := api.Group("/auth")
	auth.POST("/register", authHandler.Register)      // Register with username/email/password
	auth.POST("/login", authHandler.Login)            // Login with username/email + password
	auth.POST("/challenge", authHandler.GetChallenge) // Get challenge nonce for DID login (optional)
	auth.POST("/verify", authHandler.VerifyChallenge) // Verify signed challenge and get JWT (optional)

	// User routes
	users := api.Group("/users")
	users.GET("/:id", userHandler.GetProfile)      // Public - view any user profile by UUID
	users.GET("/did", userHandler.GetProfileByDID) // Public - view user profile by DID

	// Protected user routes (require authentication)
	usersAuth := api.Group("/users")
	usersAuth.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
	usersAuth.GET("/me", userHandler.GetCurrentUser)
	usersAuth.PUT("/me", userHandler.UpdateProfile)
	usersAuth.DELETE("/me", userHandler.DeleteAccount)

	// Post routes
	posts := api.Group("/posts")
	posts.GET("/:id", postHandler.GetPost)            // Public - view any post
	posts.GET("/user/:did", postHandler.GetUserPosts) // Public - view user's posts by DID
	posts.GET("/public", postHandler.GetPublicFeed)   // Public - get public feed

	// Protected post routes (require authentication)
	postsAuth := api.Group("/posts")
	postsAuth.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
	postsAuth.POST("", postHandler.CreatePost)
	postsAuth.GET("/feed", postHandler.GetFeed)
	postsAuth.PUT("/:id", postHandler.UpdatePost)
	postsAuth.DELETE("/:id", postHandler.DeletePost)

	// Follow routes (require authentication)
	followAuth := api.Group("/users")
	followAuth.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
	followAuth.POST("/:id/follow", followHandler.FollowUser)
	followAuth.DELETE("/:id/follow", followHandler.UnfollowUser)

	// Public follow info
	users.GET("/:id/followers", followHandler.GetFollowers)
	users.GET("/:id/following", followHandler.GetFollowing)
	users.GET("/:id/stats", followHandler.GetFollowStats)

	// Post interaction routes (require authentication)
	interactionAuth := api.Group("/posts")
	interactionAuth.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
	interactionAuth.POST("/:id/like", interactionHandler.LikePost)
	interactionAuth.DELETE("/:id/like", interactionHandler.UnlikePost)
	interactionAuth.POST("/:id/repost", interactionHandler.RepostPost)
	interactionAuth.DELETE("/:id/repost", interactionHandler.UnrepostPost)
	interactionAuth.POST("/:id/bookmark", interactionHandler.BookmarkPost)
	interactionAuth.DELETE("/:id/bookmark", interactionHandler.UnbookmarkPost)

	// Bookmarks
	usersAuth.GET("/me/bookmarks", interactionHandler.GetBookmarks)

	// Search users (authenticated)
	usersAuth.GET("/search", adminHandler.SearchUsers)

	// Message routes (require authentication)
	messagesAuth := api.Group("/messages")
	messagesAuth.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
	messagesAuth.GET("/threads", messageHandler.GetThreads)
	messagesAuth.GET("/threads/:threadId", messageHandler.GetMessages)
	messagesAuth.POST("/send", messageHandler.SendMessage)
	messagesAuth.POST("/conversation/:userId", messageHandler.StartConversation)
	messagesAuth.POST("/threads/:threadId/read", messageHandler.MarkAsRead)

	// Moderation request (authenticated users)
	usersAuth.POST("/me/request-moderation", adminHandler.RequestModeration)

	// Admin routes (require authentication + admin role)
	admin := api.Group("/admin")
	admin.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
	admin.GET("/users", adminHandler.GetAllUsers)
	admin.GET("/users/suspended", adminHandler.GetSuspendedUsers)
	admin.GET("/moderation-requests", adminHandler.GetModerationRequests)
	admin.POST("/moderation-requests/:id/approve", adminHandler.ApproveModerationRequest)
	admin.POST("/moderation-requests/:id/reject", adminHandler.RejectModerationRequest)
	admin.PUT("/users/:id/role", adminHandler.UpdateUserRole)
	admin.POST("/users/:id/suspend", adminHandler.SuspendUser)
	admin.POST("/users/:id/unsuspend", adminHandler.UnsuspendUser)
	admin.GET("/actions", adminHandler.GetAdminActions)
}

// Start starts the HTTP server
func (s *Server) Start(address string) error {
	return s.echo.Start(address)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	return s.echo.Close()
}
