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

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userRepo, cfg.JWT.Secret)
	userHandler := handlers.NewUserHandler(userRepo)
	postHandler := handlers.NewPostHandler(postRepo)

	// Global middleware
	e.Use(echomiddleware.Logger())
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.CORS())

	// Routes
	setupRoutes(e, cfg, authHandler, userHandler, postHandler)

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
}

// Start starts the HTTP server
func (s *Server) Start(address string) error {
	return s.echo.Start(address)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	return s.echo.Close()
}
