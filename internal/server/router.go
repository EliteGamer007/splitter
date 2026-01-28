package server

import (
	"splitter/internal/config"
	"splitter/internal/handlers"
	"splitter/internal/middleware"
	"splitter/internal/repository"

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

	// Initialize repositories
	userRepo := repository.NewUserRepository()
	postRepo := repository.NewPostRepository()
	followRepo := repository.NewFollowRepository()
	interactionRepo := repository.NewInteractionRepository()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userRepo, cfg.JWT.Secret)
	userHandler := handlers.NewUserHandler(userRepo)
	postHandler := handlers.NewPostHandler(postRepo)
	followHandler := handlers.NewFollowHandler(followRepo, userRepo)
	interactionHandler := handlers.NewInteractionHandler(interactionRepo, userRepo)

	// Global middleware
	e.Use(echomiddleware.Logger())
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:3000", "http://127.0.0.1:3000"},
		AllowMethods:     []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.OPTIONS},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization},
		AllowCredentials: true,
	}))

	// Routes
	setupRoutes(e, cfg, authHandler, userHandler, postHandler, followHandler, interactionHandler)

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
) {
	// API v1 group
	api := e.Group("/api/v1")

	// Health check
	api.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{
			"status": "ok",
		})
	})

	// Auth routes (public) - DID-based authentication
	auth := api.Group("/auth")
	auth.POST("/register", authHandler.Register)      // Register with DID + public key
	auth.POST("/challenge", authHandler.GetChallenge) // Get challenge nonce for login
	auth.POST("/verify", authHandler.VerifyChallenge) // Verify signed challenge and get JWT

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
	posts.GET("/:id", postHandler.GetPost)           // Public - view any post
	posts.GET("/user/:id", postHandler.GetUserPosts) // Public - view user's posts

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
}

// Start starts the HTTP server
func (s *Server) Start(address string) error {
	return s.echo.Start(address)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	return s.echo.Close()
}
