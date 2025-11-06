package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/swagger"

	_ "immobilier-saas/api/docs" // swagger docs
	"immobilier-saas/api/internal/config"
	"immobilier-saas/api/internal/handler"
	"immobilier-saas/api/internal/middleware"
	"immobilier-saas/api/internal/pkg/postgres"
	"immobilier-saas/api/internal/pkg/redis"
	"immobilier-saas/api/internal/repository"
	"immobilier-saas/api/internal/service"
)

// @title           Immobilier SaaS API
// @version         1.0
// @description     Real-estate SaaS platform API with JWT authentication
// @termsOfService  https://immobilier.example.com/terms

// @contact.name   API Support
// @contact.url    https://immobilier.example.com/support
// @contact.email  support@immobilier.example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize database
	db, err := postgres.NewConnection(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer postgres.Close(db)

	// Initialize Redis
	redisClient := redis.NewClient(cfg.RedisURL)
	defer redisClient.Close()

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	listingRepo := repository.NewListingRepository(db)
	bookingRepo := repository.NewBookingRepository(db)
	paymentRepo := repository.NewPaymentRepository(db)

	// Initialize services
	authService := service.NewAuthService(userRepo, redisClient, cfg)
	listingService := service.NewListingService(listingRepo, cfg)
	bookingService := service.NewBookingService(bookingRepo, listingRepo, cfg)
	paymentService := service.NewPaymentService(paymentRepo, bookingRepo, cfg)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	listingHandler := handler.NewListingHandler(listingService)
	bookingHandler := handler.NewBookingHandler(bookingService)
	paymentHandler := handler.NewPaymentHandler(paymentService)

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Immobilier SaaS API",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		ErrorHandler: middleware.CustomErrorHandler,
	})

	// Global middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "${time} | ${status} | ${latency} | ${method} | ${path}\n",
	}))
	app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))
	app.Use(helmet.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     "GET,POST,PUT,DELETE,PATCH,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowCredentials: true,
		MaxAge:           86400,
	}))

	// Rate limiting
	app.Use(limiter.New(limiter.Config{
		Max:        100,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "Rate limit exceeded",
			})
		},
	}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "ok",
			"time":   time.Now(),
		})
	})

	// Swagger documentation
	app.Get("/swagger/*", swagger.HandlerDefault)

	// API routes
	api := app.Group("/api/v1")

	// Auth routes (public)
	auth := api.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)
	auth.Post("/logout", authHandler.Logout)
	auth.Post("/refresh", authHandler.RefreshToken)
	auth.Post("/forgot-password", authHandler.ForgotPassword)
	auth.Post("/reset-password", authHandler.ResetPassword)
	auth.Get("/verify-email/:token", authHandler.VerifyEmail)

	// Listings routes (public read, auth write)
	listings := api.Group("/listings")
	listings.Get("/", listingHandler.GetListings)
	listings.Get("/:id", listingHandler.GetListingByID)
	listings.Get("/search", listingHandler.SearchListings)

	// Protected listing routes
	listings.Use(middleware.AuthMiddleware(cfg.JWTPublicKey))
	listings.Post("/", middleware.RequireRole("landlord"), listingHandler.CreateListing)
	listings.Put("/:id", middleware.RequireRole("landlord"), listingHandler.UpdateListing)
	listings.Delete("/:id", middleware.RequireRole("landlord"), listingHandler.DeleteListing)

	// Bookings routes (all protected)
	bookings := api.Group("/bookings")
	bookings.Use(middleware.AuthMiddleware(cfg.JWTPublicKey))
	bookings.Get("/", bookingHandler.GetUserBookings)
	bookings.Get("/:id", bookingHandler.GetBookingByID)
	bookings.Post("/", bookingHandler.CreateBooking)
	bookings.Patch("/:id/cancel", bookingHandler.CancelBooking)

	// Payment routes (protected)
	payments := api.Group("/payments")
	payments.Use(middleware.AuthMiddleware(cfg.JWTPublicKey))
	payments.Post("/", paymentHandler.CreatePayment)
	payments.Get("/:id", paymentHandler.GetPaymentByID)

	// Webhook routes (public with signature verification)
	webhooks := api.Group("/webhooks")
	webhooks.Post("/payment/wave", paymentHandler.WaveWebhook)
	webhooks.Post("/payment/orange", paymentHandler.OrangeMoneyWebhook)
	webhooks.Post("/payment/free", paymentHandler.FreeMoneyWebhook)

	// Admin routes
	admin := api.Group("/admin")
	admin.Use(middleware.AuthMiddleware(cfg.JWTPublicKey))
	admin.Use(middleware.RequireRole("admin"))
	// Add admin handlers here

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := app.ShutdownWithContext(ctx); err != nil {
			log.Fatalf("Server forced to shutdown: %v", err)
		}
	}()

	// Start server
	port := cfg.Port
	if port == "" {
		port = "8080"
	}
	log.Printf("Starting server on port %s...", port)
	if err := app.Listen(fmt.Sprintf(":%s", port)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
