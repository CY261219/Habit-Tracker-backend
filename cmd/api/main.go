package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "zenith/docs"
	"zenith/internal/config"
	"zenith/internal/handlers"
	"zenith/internal/middleware"
	repopostgres "zenith/internal/repository/postgres"
)

// @title           Habit Tracker API
// @version         1.0
// @description     Backend API for Habit Tracker with JWT authentication and habit synchronization.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support
// @contact.email  support@swagger.io

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Type "Bearer <your_token>" to authenticate.

func main() {
	// -------------------------------------------------------------------------
	// 1. Load environment variables from .env (development only; ignored in prod)
	// -------------------------------------------------------------------------
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found — using system environment variables")
	}

	// -------------------------------------------------------------------------
	// 2. Run database migrations BEFORE opening the application connection
	// -------------------------------------------------------------------------
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable must be set")
	}

	if err := config.RunMigrations(dsn); err != nil {
		log.Fatalf("Database migration failed: %v", err)
	}

	// -------------------------------------------------------------------------
	// 3. Open GORM connection with connection pool
	// -------------------------------------------------------------------------
	db, err := config.ConnectDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// -------------------------------------------------------------------------
	// 4. Wire up repositories (Dependency Injection)
	// -------------------------------------------------------------------------
	userRepo := repopostgres.NewUserRepository(db)
	habitRepo := repopostgres.NewHabitRepository(db)
	habitLogRepo := repopostgres.NewHabitLogRepository(db)

	// -------------------------------------------------------------------------
	// 5. Configure Gin
	// -------------------------------------------------------------------------
	if os.Getenv("APP_ENV") == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger())   // request logging
	r.Use(gin.Recovery()) // panic recovery → 500 instead of crash

	// -------------------------------------------------------------------------
	// 6. Register routes
	// -------------------------------------------------------------------------
	api := r.Group("/api/v1")

	// Swagger route
	api.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Public routes (no auth required)
	auth := api.Group("/auth")
	{
		auth.POST("/register", handlers.Register(userRepo))
		auth.POST("/login", handlers.Login(userRepo))
	}

	// Protected routes (JWT required)
	protected := api.Group("/")
	protected.Use(middleware.Auth())
	{
		// Habit Management
		habitsGroup := protected.Group("/habits")
		{
			habitsGroup.POST("", handlers.CreateHabit(habitRepo))
			habitsGroup.GET("", handlers.ListHabits(habitRepo))
			habitsGroup.GET("/:id", handlers.GetHabit(habitRepo))
			habitsGroup.PUT("/:id", handlers.UpdateHabit(habitRepo))
			habitsGroup.DELETE("/:id", handlers.DeleteHabit(habitRepo))
		}

		sync := protected.Group("/sync")
		{
			sync.POST("/push", handlers.PushMutations(habitRepo, habitLogRepo, userRepo))
		}
	}

	// -------------------------------------------------------------------------
	// 7. Start HTTP server with graceful shutdown
	// -------------------------------------------------------------------------
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine so it doesn't block the shutdown listener
	go func() {
		log.Printf("Server starting on :%s (env: %s)", port, os.Getenv("APP_ENV"))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for SIGINT or SIGTERM
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutdown signal received, draining connections...")

	// Give in-flight requests up to 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited gracefully")
}
