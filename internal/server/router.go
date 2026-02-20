package server

import (
	"log"

	"splitter/internal/config"
	"splitter/internal/federation"
	"splitter/internal/handlers"
	"splitter/internal/middleware"
	"splitter/internal/repository"

	govalidator "github.com/go-playground/validator/v10"
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
	e.Validator = &CustomValidator{validator: govalidator.New()}

	// Initialize repositories
	userRepo := repository.NewUserRepository()
	postRepo := repository.NewPostRepository()
	followRepo := repository.NewFollowRepository()
	interactionRepo := repository.NewInteractionRepository()
	messageRepo := repository.NewMessageRepository()

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userRepo, cfg.JWT.Secret)
	userHandler := handlers.NewUserHandler(userRepo, cfg)
	postHandler := handlers.NewPostHandler(postRepo, userRepo, cfg)
	mediaHandler := handlers.NewMediaHandler(postRepo)
	followHandler := handlers.NewFollowHandler(followRepo, userRepo)
	interactionHandler := handlers.NewInteractionHandler(interactionRepo, userRepo, postRepo, cfg)
	adminHandler := handlers.NewAdminHandler(userRepo)
	messageHandler := handlers.NewMessageHandler(messageRepo, userRepo, cfg)
	replyHandler := handlers.NewReplyHandler()

	// Federation handlers
	webfingerHandler := handlers.NewWebFingerHandler(userRepo, cfg)
	actorHandler := handlers.NewActorHandler(userRepo, cfg)
	inboxHandler := handlers.NewInboxHandler(userRepo, messageRepo, cfg)
	outboxHandler := handlers.NewOutboxHandler(userRepo, cfg)
	federationHandler := handlers.NewFederationHandler(userRepo, cfg)

	// Initialize federation keys if federation is enabled
	if cfg.Federation.Enabled {
		if err := federation.EnsureInstanceKeys(cfg.Federation.Domain); err != nil {
			log.Printf("[Federation] WARNING: Failed to initialize keys: %v", err)
		} else {
			log.Printf("[Federation] Initialized for domain '%s' at %s", cfg.Federation.Domain, cfg.Federation.URL)
		}
	}

	// Global middleware
	e.Use(echomiddleware.Logger())
	e.Use(echomiddleware.Recover())
	e.Use(echomiddleware.BodyLimit("6M")) // Limit body size to 6MB (allow overhead for 5MB file)
	e.Use(echomiddleware.CORSWithConfig(echomiddleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:3000", "http://127.0.0.1:3000", "http://localhost:3001", "http://127.0.0.1:3001", "http://localhost:8000", "http://localhost:8001"},
		AllowMethods:     []string{echo.GET, echo.POST, echo.PUT, echo.DELETE, echo.OPTIONS},
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAuthorization, "Signature", "Date", "Digest"},
		AllowCredentials: true,
	}))

	// Static routes (legacy uploads fallback)
	e.Static("/uploads", "uploads")

	// Routes
	setupRoutes(e, cfg, authHandler, userHandler, postHandler, mediaHandler, followHandler, interactionHandler, adminHandler, messageHandler, replyHandler, webfingerHandler, actorHandler, inboxHandler, outboxHandler, federationHandler)

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
	mediaHandler *handlers.MediaHandler,
	followHandler *handlers.FollowHandler,
	interactionHandler *handlers.InteractionHandler,
	adminHandler *handlers.AdminHandler,
	messageHandler *handlers.MessageHandler,
	replyHandler *handlers.ReplyHandler,
	webfingerHandler *handlers.WebFingerHandler,
	actorHandler *handlers.ActorHandler,
	inboxHandler *handlers.InboxHandler,
	outboxHandler *handlers.OutboxHandler,
	federationHandler *handlers.FederationHandler,
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
	users.GET("/:id/avatar", userHandler.GetAvatar)

	// Protected user routes (require authentication)
	usersAuth := api.Group("/users")
	usersAuth.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
	usersAuth.GET("/me", userHandler.GetCurrentUser)
	usersAuth.PUT("/me", userHandler.UpdateProfile)
	usersAuth.POST("/me/avatar", userHandler.UploadAvatar)
	usersAuth.PUT("/me/encryption-key", userHandler.UpdateEncryptionKey) // Add encryption key for existing users
	usersAuth.DELETE("/me", userHandler.DeleteAccount)

	// Media routes
	api.GET("/media/:id/content", mediaHandler.GetMediaContent)

	// Post routes
	posts := api.Group("/posts")
	posts.Use(middleware.OptionalAuthMiddleware(cfg.JWT.Secret))
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
	// Replies (Authenticated)
	postsAuth.POST("/:id/replies", replyHandler.CreateReply)

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

	// Replies
	posts.GET("/:id/replies", replyHandler.GetReplies)

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
	messagesAuth.DELETE("/:messageId", messageHandler.DeleteMessage) // Delete message (3-hour window)
	messagesAuth.PUT("/:messageId", messageHandler.EditMessage)      // Edit message (3-hour window)

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
	admin.GET("/moderation-queue", adminHandler.GetModerationQueue)
	admin.POST("/moderation-queue/:id/approve", adminHandler.ApproveModerationItem)
	admin.POST("/moderation-queue/:id/remove", adminHandler.RemoveModerationContent)
	admin.POST("/users/:id/warn", adminHandler.WarnUser)
	admin.POST("/domains/block", adminHandler.BlockDomain)
	admin.GET("/domains/blocked", adminHandler.GetBlockedDomains)
	admin.DELETE("/domains/:domain/block", adminHandler.UnblockDomain)
	admin.PUT("/users/:id/role", adminHandler.UpdateUserRole)
	admin.POST("/users/:id/suspend", adminHandler.SuspendUser)
	admin.POST("/users/:id/unsuspend", adminHandler.UnsuspendUser)
	admin.GET("/actions", adminHandler.GetAdminActions)
	admin.GET("/federation-inspector", adminHandler.GetFederationInspector)

	// ============================================================
	// FEDERATION ROUTES
	// ============================================================

	// WebFinger & ActivityPub (public, no auth)
	e.GET("/.well-known/webfinger", webfingerHandler.Handle)
	e.GET("/ap/users/:username", actorHandler.GetActor)          // ActivityPub Actor
	e.POST("/ap/users/:username/inbox", inboxHandler.Handle)     // Receive activities (per-user)
	e.POST("/ap/shared-inbox", inboxHandler.Handle)              // Shared inbox (federation)
	e.GET("/ap/users/:username/outbox", outboxHandler.GetOutbox) // List activities

	// Federation API (public, no auth required for cross-instance discovery)
	fed := api.Group("/federation")
	fed.GET("/users", federationHandler.SearchRemoteUsers)        // Search remote users
	fed.GET("/timeline", federationHandler.GetFederatedTimeline)  // Federated timeline
	fed.GET("/all-users", federationHandler.GetAllFederatedUsers) // All users across instances
	fed.GET("/public-users", federationHandler.GetPublicUserList) // Public user list for federation

	// Federation API (authenticated)
	fedAuth := api.Group("/federation")
	fedAuth.Use(middleware.AuthMiddleware(cfg.JWT.Secret))
	fedAuth.POST("/follow", federationHandler.FollowRemoteUser) // Follow remote user
}

// Start starts the HTTP server
func (s *Server) Start(address string) error {
	return s.echo.Start(address)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() error {
	return s.echo.Close()
}

// Echo returns the underlying Echo instance (for testing)
func (s *Server) Echo() *echo.Echo {
	return s.echo
}
