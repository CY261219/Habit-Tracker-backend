package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"zenith/internal/config"
	"zenith/internal/handlers"
)

func main() {
	// Initialize the database connection
	db, err := config.ConnectDB()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Create Gin router
	r := gin.Default()

	// Register API routes
	api := r.Group("/api/v1")
	{
		syncGroup := api.Group("/sync")
		{
			syncGroup.POST("/push", handlers.PushMutations(db))
		}
	}

	// Start HTTP server
	log.Println("Starting server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
