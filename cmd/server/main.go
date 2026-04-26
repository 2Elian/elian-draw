package main

import (
	"fmt"
	"log"

	"github.com/2Elian/next-ai-draw-io/go-backend/internal/config"
	"github.com/2Elian/next-ai-draw-io/go-backend/internal/handler"
	"github.com/2Elian/next-ai-draw-io/go-backend/internal/util"
	"github.com/gin-gonic/gin"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()
	log.Printf("[Config] Port: %s, Provider: %s, Model: %s", cfg.Port, cfg.AIProvider, cfg.AIModel)

	// Load server models
	cfg.LoadServerModels()

	// Load cached responses
	util.LoadCachedResponses("data/cached_responses.json")

	// Set Gin mode
	if cfg.IsDev {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(handler.CORSMiddleware())
	r.Use(handler.ErrorRecoveryMiddleware())

	// API routes
	api := r.Group("/api")
	{
		// Chat route (core)
		api.POST("/chat", handler.StreamErrorHandler(), handler.HandleChat)

		// Config and model routes
		api.GET("/config", handler.HandleConfig)
		api.GET("/server-models", handler.HandleServerModels)

		// Validation routes
		api.POST("/validate-diagram", handler.HandleValidateDiagram)
		api.POST("/validate-model", handler.HandleValidateModel)

		// Misc routes
		api.POST("/log-feedback", handler.HandleLogFeedback)
		api.POST("/log-save", handler.HandleLogSave)
		api.POST("/parse-url", handler.HandleParseURL)
		api.POST("/verify-access-code", handler.HandleVerifyAccessCode)
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Start server
	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("[Server] Starting on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("[Server] Failed to start: %v", err)
	}
}
