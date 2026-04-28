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
		"hasApiKey":      cfg.AIModel != "",
		"provider":       string(cfg.AIProvider),
		"model":          cfg.AIModel,
		"hasAccessCode":  len(cfg.AccessCodeList) > 0,
		"enablePdfInput": cfg.EnablePDFInput,
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
// 验证聊天消息中的文件附件是否符合规范
func ValidateFileParts(messages []model.UIMessage) (bool, string) { // 第一个返回值表示是否验证通过，第二个返回错误描述
	if len(messages) == 0 {
		return true, ""
	}

	lastMsg := messages[len(messages)-1] // 获取最后一条消息 --> 函数只检查最后一条消息中的文件附件，即当前session内用户最近一次的交互聊天
	var fileCount int                    // 默认为0
	for _, part := range lastMsg.Parts {
		if part.Type == "file" {
			fileCount++
			// Check file size (base64 encoded)
			if part.URL != "" && strings.HasPrefix(part.URL, "data:") { // example: part.URL="data:[<mediatype>][;base64],<data>" 则这个为true 否则为false
				parts := strings.SplitN(part.URL, ",", 2) //[<mediatype>][;base64]
				if len(parts) == 2 {
					decodedSize := int(math.Ceil(float64(len(parts[1])) * 3 / 4)) // 提取 Base64 编码的数据部分
					if decodedSize > 2*1024*1024 {
						return false, "File exceeds 2MB limit." //若解码后大小超过 2MB，返回失败及错误信息
					}
				}
			}
		}
	}

	if fileCount > 5 {
		return false, "Too many files. Maximum 5 allowed." // 限制文件数量不能大于5个，不然后端处理起来困难
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
