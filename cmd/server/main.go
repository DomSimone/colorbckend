package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/greens-marketplace/internal/config"
	"github.com/greens-marketplace/internal/database"
	"github.com/greens-marketplace/internal/handlers"
	"github.com/greens-marketplace/internal/middleware"
	"github.com/greens-marketplace/internal/services"
	"github.com/greens-marketplace/internal/utils"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/httprate"
	"github.com/go-chi/jwtauth/v5"
)

var (
	configFile = flag.String("config", "config.yaml", "Path to configuration file")
)

func main() {
	flag.Parse()

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Warn().Msg("No .env file found, using system environment variables")
	}

	// Load configuration
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Setup logging
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.With().Str("service", "greens-marketplace").Logger()
	if cfg.Environment == "development" {
		log.Logger = log.Logger.Level(zerolog.DebugLevel)
	} else {
		log.Logger = log.Logger.Level(zerolog.InfoLevel)
	}

	// Initialize database
	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	// Initialize Redis
	redisClient, err := database.NewRedisClient(cfg.Redis)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	defer redisClient.Close()

	// Initialize services
	userService := services.NewUserService(db, redisClient)
	productService := services.NewProductService(db, redisClient)
	orderService := services.NewOrderService(db, redisClient)
	searchService := services.NewSearchService(db, redisClient)
	notificationService := services.NewNotificationService(db, redisClient)

	// Initialize handlers
	userHandler := handlers.NewUserHandler(userService)
	productHandler := handlers.NewProductHandler(productService, searchService)
	orderHandler := handlers.NewOrderHandler(orderService)
	notificationHandler := handlers.NewNotificationHandler(notificationService)

	// Setup JWT authentication
	tokenAuth := jwtauth.New("HS256", []byte(cfg.JWT.Secret), nil)

	// Create router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	
	// CORS
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://localhost:3001"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Rate limiting
	r.Use(httprate.LimitByIP(100, 1*time.Minute))

	// Health check
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "healthy", "timestamp": "` + time.Now().Format(time.RFC3339) + `"}`))
	})

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes
		r.Post("/auth/register", userHandler.Register)
		r.Post("/auth/login", userHandler.Login)
		r.Post("/auth/refresh", userHandler.RefreshToken)
		r.Get("/categories", productHandler.GetCategories)

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(middleware.JWTAuth(tokenAuth))
			r.Use(middleware.SetHeader("Authorization", "Bearer"))

			// User routes
			r.Get("/users/profile", userHandler.GetProfile)
			r.Put("/users/profile", userHandler.UpdateProfile)
			r.Get("/users/preferences", userHandler.GetPreferences)
			r.Put("/users/preferences", userHandler.UpdatePreferences)

			// Product routes
			r.Post("/products", productHandler.CreateProduct)
			r.Get("/products", productHandler.GetProducts)
			r.Get("/products/{id}", productHandler.GetProduct)
			r.Put("/products/{id}", productHandler.UpdateProduct)
			r.Delete("/products/{id}", productHandler.DeleteProduct)
			r.Get("/products/{id}/similar", productHandler.GetSimilarProducts)
			r.Post("/products/{id}/reviews", productHandler.CreateReview)
			r.Get("/products/{id}/reviews", productHandler.GetReviews)

			// Search routes
			r.Get("/search", productHandler.SearchProducts)
			r.Post("/search/semantic", productHandler.SemanticSearch)

			// Cart routes
			r.Get("/cart", productHandler.GetCart)
			r.Post("/cart", productHandler.AddToCart)
			r.Put("/cart/{productId}", productHandler.UpdateCartItem)
			r.Delete("/cart/{productId}", productHandler.RemoveFromCart)

			// Wishlist routes
			r.Get("/wishlist", productHandler.GetWishlist)
			r.Post("/wishlist/{productId}", productHandler.AddToWishlist)
			r.Delete("/wishlist/{productId}", productHandler.RemoveFromWishlist)

			// Order routes
			r.Post("/orders", orderHandler.CreateOrder)
			r.Get("/orders", orderHandler.GetOrders)
			r.Get("/orders/{id}", orderHandler.GetOrder)
			r.Put("/orders/{id}/status", orderHandler.UpdateOrderStatus)
			r.Post("/orders/{id}/payment", orderHandler.ProcessPayment)

			// Notification routes
			r.Get("/notifications", notificationHandler.GetNotifications)
			r.Put("/notifications/{id}/read", notificationHandler.MarkAsRead)
			r.Delete("/notifications/{id}", notificationHandler.DeleteNotification)
		})
	})

	// Start server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Info().Int("port", cfg.Server.Port).Msg("Starting server")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Info().Msg("Shutting down server...")

	// Give the server 5 seconds to finish current requests
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited")
}