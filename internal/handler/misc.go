package handler

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/2Elian/next-ai-draw-io/go-backend/internal/config"
	"github.com/2Elian/next-ai-draw-io/go-backend/internal/model"
	"github.com/gin-gonic/gin"
)

// ServerModelsResponse is the response for GET /api/server-models
type ServerModelsResponse struct {
	Models []FlattenedModelResponse `json:"models"`
}

type FlattenedModelResponse struct {
	ID            string `json:"id"`
	ModelID       string `json:"modelId"`
	Provider      string `json:"provider"`
	ProviderLabel string `json:"providerLabel"`
	IsDefault     bool   `json:"isDefault"`
}

// HandleConfig returns the public app configuration
func HandleConfig(c *gin.Context) {
	cfg := config.AppConfig

	c.JSON(http.StatusOK, gin.H{
		"hasApiKey":         cfg.AIModel != "",
		"provider":          string(cfg.AIProvider),
		"model":             cfg.AIModel,
		"hasAccessCode":     len(cfg.AccessCodeList) > 0,
		"enablePdfInput":    cfg.EnablePDFInput,
	})
}

// HandleServerModels returns available server models from ai-models.json
func HandleServerModels(c *gin.Context) {
	cfg := config.AppConfig
	cfg.LoadServerModels()

	models := cfg.GetServerModels()
	result := make([]FlattenedModelResponse, 0, len(models))
	for _, m := range models {
		result = append(result, FlattenedModelResponse{
			ID:            m.ID,
			ModelID:       m.ModelID,
			Provider:      string(m.Provider),
			ProviderLabel: m.ProviderLabel,
			IsDefault:     m.IsDefault,
		})
	}

	c.JSON(http.StatusOK, ServerModelsResponse{Models: result})
}

// HandleLogFeedback logs user feedback (placeholder)
func HandleLogFeedback(c *gin.Context) {
	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	fmt.Printf("[Feedback] %v\n", body)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// HandleLogSave logs save events (placeholder)
func HandleLogSave(c *gin.Context) {
	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	fmt.Printf("[Save] %v\n", body)
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// HandleParseURL parses a URL and extracts content (placeholder)
func HandleParseURL(c *gin.Context) {
	var body struct {
		URL string `json:"url"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"url":     body.URL,
		"content": "",
		"title":   "",
	})
}

// HandleVerifyAccessCode verifies an access code
func HandleVerifyAccessCode(c *gin.Context) {
	cfg := config.AppConfig

	if len(cfg.AccessCodeList) == 0 {
		c.JSON(http.StatusOK, gin.H{"valid": true})
		return
	}

	var body struct {
		AccessCode string `json:"accessCode"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	for _, code := range cfg.AccessCodeList {
		if code == body.AccessCode {
			c.JSON(http.StatusOK, gin.H{"valid": true})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"valid": false})
}

// HandleValidateDiagram validates a diagram (placeholder - would need AI model)
func HandleValidateDiagram(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"valid": true})
}

// HandleValidateModel validates a model configuration (placeholder)
func HandleValidateModel(c *gin.Context) {
	var body struct {
		Provider string `json:"provider"`
		APIKey   string `json:"apiKey"`
		BaseURL  string `json:"baseUrl"`
		ModelID  string `json:"modelId"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid": true,
		"model": gin.H{
			"id":      fmt.Sprintf("test-%d", len(body.ModelID)),
			"modelId": body.ModelID,
		},
	})
}

// ValidateFileParts validates file parts in messages
func ValidateFileParts(messages []model.UIMessage) (bool, string) {
	if len(messages) == 0 {
		return true, ""
	}

	lastMsg := messages[len(messages)-1]
	var fileCount int
	for _, part := range lastMsg.Parts {
		if part.Type == "file" {
			fileCount++
			// Check file size (base64 encoded)
			if part.URL != "" && strings.HasPrefix(part.URL, "data:") {
				parts := strings.SplitN(part.URL, ",", 2)
				if len(parts) == 2 {
					decodedSize := int(math.Ceil(float64(len(parts[1])) * 3 / 4))
					if decodedSize > 2*1024*1024 {
						return false, "File exceeds 2MB limit."
					}
				}
			}
		}
	}

	if fileCount > 5 {
		return false, "Too many files. Maximum 5 allowed."
	}

	return true, ""
}

// Helper to safely marshal JSON
func mustMarshal(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage("{}")
	}
	return data
}
