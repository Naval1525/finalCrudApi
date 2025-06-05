// main.go - Application entry point
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
)

func main() {
	// Load configuration from environment variables
	config := LoadConfig()

	// Initialize database connection
	db, err := InitDatabase(config.DatabaseUrl)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Run database migrations (create tables if they don't exist)
	if err := RunMigrations(db); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	// Initialize repository layer (handles database operations)
	repo := NewRepository(db)

	// Initialize service layer (handles business logic)
	service := NewService(repo)

	// Initialize handler layer (handles HTTP requests)
	handler := NewHandler(service)

	// Setup Gin router with middleware
	router := gin.Default()

	// Add middleware for CORS, logging, etc.
	router.Use(CORSMiddleware())
	router.Use(LoggingMiddleware())

	// Setup routes
	setupRoutes(router, handler)

	// Create HTTP server
	server := &http.Server{
		Addr:    ":" + config.Port,
		Handler: router,
	}

	// Start server in a goroutine so it doesn't block
	go func() {
		log.Printf("Server starting on port %s", config.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server:", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exited")
}

// setupRoutes configures all API routes
func setupRoutes(router *gin.Engine, handler *Handler) {
	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API version 1 routes
	v1 := router.Group("/api/v1")
	{
		// Authentication routes (no auth required)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", handler.Register)
			auth.POST("/login", handler.Login)
		}

		// Protected routes (require authentication)
		protected := v1.Group("/")
		protected.Use(AuthMiddleware()) // Apply authentication middleware
		{
			// User routes
			users := protected.Group("/users")
			{
				users.GET("", handler.GetUsers)          // GET /api/v1/users
				users.GET("/:id", handler.GetUser)       // GET /api/v1/users/123
				users.PUT("/:id", handler.UpdateUser)    // PUT /api/v1/users/123
				users.DELETE("/:id", handler.DeleteUser) // DELETE /api/v1/users/123
			}

			// You can add more resource routes here (posts, products, etc.)
			// Example:
			// posts := protected.Group("/posts")
			// {
			//     posts.GET("", handler.GetPosts)
			//     posts.POST("", handler.CreatePost)
			//     posts.GET("/:id", handler.GetPost)
			//     posts.PUT("/:id", handler.UpdatePost)
			//     posts.DELETE("/:id", handler.DeletePost)
			// }
		}
	}
}
