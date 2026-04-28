package handler

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/2Elian/next-ai-draw-io/go-backend/internal/agent"
	"github.com/2Elian/next-ai-draw-io/go-backend/internal/config"
	"github.com/2Elian/next-ai-draw-io/go-backend/internal/model"
	"github.com/2Elian/next-ai-draw-io/go-backend/internal/provider"
	"github.com/2Elian/next-ai-draw-io/go-backend/internal/sse"
	"github.com/2Elian/next-ai-draw-io/go-backend/internal/util"
	"github.com/gin-gonic/gin"
)

// HandleChat handles POST /api/chat - the core chat endpoint
func HandleChat(c *gin.Context) {
	cfg := config.AppConfig
	var req model.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	if len(req.CustomSystemMessage) > 5000 {
		req.CustomSystemMessage = req.CustomSystemMessage[:5000]
	}

	sessionID := req.SessionID
	if sessionID == "" || len(sessionID) > 200 {
		sessionID = ""
	}

	var userInputText string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			for _, part := range req.Messages[i].Parts {
				if part.Type == "text" { // 如果是纯文本 后面没有必要遍历了
					userInputText = part.Text
					break
				}
			}
			break
		}
	}

	log.Printf("[Chat] User input: %q (messages: %d, session: %s)",
		truncate(userInputText, 100), len(req.Messages), sessionID)

	if valid, errMsg := ValidateFileParts(req.Messages); !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": errMsg})
		return
	}
	// cache logic
	if len(req.Messages) == 1 && agent.IsMinimalDiagram(req.XML) { // 新对话的第一条请求
		textPart := ""
		var hasFile bool
		for _, part := range req.Messages[0].Parts {
			if part.Type == "text" {
				textPart = part.Text
			}
			if part.Type == "file" {
				hasFile = true
			}
		}
		if cached := util.FindCachedResponse(textPart, hasFile); cached != nil {
			writeCachedResponse(c, cached.XML) // 将缓存以 SSE 流式响应输出
			return
		}
	}

	// 初始化AI Model
	selectedModelID := c.GetHeader("x-selected-model-id")
	providerHeader := c.GetHeader("x-ai-provider")
	baseURLHeader := c.GetHeader("x-ai-base-url")
	// Handle server model lookup
	var serverModelConfig *config.FlattenedServerModel
	if strings.HasPrefix(selectedModelID, "server:") {
		serverModelConfig = cfg.FindServerModelByID(selectedModelID)
		if serverModelConfig != nil {
			log.Printf("[Server Model] ID: %s, Provider: %s", selectedModelID, serverModelConfig.Provider)
		}
	}
	overrides := &model.ClientOverrides{
		Provider: func() config.ProviderName {
			if serverModelConfig != nil {
				return serverModelConfig.Provider
			}
			return config.ProviderName(providerHeader)
		}(),
		BaseURL:            baseURLHeader,
		APIKey:             c.GetHeader("x-ai-api-key"),
		ModelID:            c.GetHeader("x-ai-model"),
		AWSAccessKeyID:     c.GetHeader("x-aws-access-key-id"),
		AWSSecretAccessKey: c.GetHeader("x-aws-secret-access-key"),
		AWSRegion:          c.GetHeader("x-aws-region"),
		AWSSessionToken:    c.GetHeader("x-aws-session-token"),
		VertexAPIKey:       c.GetHeader("x-vertex-api-key"),
	}
	if serverModelConfig != nil {
		overrides.APIKeyEnvs = serverModelConfig.APIKeyEnv
		overrides.BaseURLEnv = serverModelConfig.BaseURLEnv
	}
	minimalStyle := c.GetHeader("x-minimal-style") == "true"
	modelCfg, err := provider.GetAIModel(overrides)
	if err != nil {
		log.Printf("[Chat] Model init error: %v", err)
		writeErrorJSON(c, err)
		return
	}

	log.Printf("[Chat] Provider: %s, Model: %s", modelCfg.Provider, modelCfg.ModelID)

	// build Prompt
	systemPrompt := agent.GetSystemPrompt(modelCfg.ModelID, minimalStyle)
	if req.CustomSystemMessage != "" {
		systemPrompt = systemPrompt + "\n\n## Custom Instructions\n" + req.CustomSystemMessage
	}
	// 对image进行验证
	var fileParts []model.UIPart
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			for _, part := range req.Messages[i].Parts {
				if part.Type == "file" {
					fileParts = append(fileParts, part)
				}
			}
			break
		}
	}

	if len(fileParts) > 0 && !config.SupportsImageInput(modelCfg.ModelID) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("The model \"%s\" does not support image input.", modelCfg.ModelID),
		})
		return
	}

	modelMessages := agent.ConvertUIMessagesToModelMessages(req.Messages)

	if cfg.EnableHistoryReplace {
		modelMessages = agent.ReplaceHistoricalToolInputs(modelMessages)
	}

	// Filter empty content messages
	filtered := make([]agent.Message, 0, len(modelMessages))
	for _, msg := range modelMessages {
		if len(msg.Content) > 0 {
			filtered = append(filtered, msg)
		}
	}
	modelMessages = filtered

	// Update last user message with formatted input
	formattedInput := agent.FormatUserInput(userInputText)
	if len(modelMessages) > 0 {
		lastIdx := len(modelMessages) - 1
		if modelMessages[lastIdx].Role == "user" {
			contentParts := []agent.ContentPart{
				{Type: "text", Text: formattedInput},
			}
			// Add image parts
			for _, fp := range fileParts {
				contentParts = append(contentParts, agent.ContentPart{
					Type:     "image",
					ImageURL: fp.URL,
					MimeType: fp.MediaType,
				})
			}
			modelMessages[lastIdx].Content = contentParts
		}
	}

	// Build system messages
	xmlContext := agent.BuildXMLContext(req.XML, req.PreviousXML)
	shouldCache := config.SupportsPromptCaching(modelCfg.ModelID)
	systemMessages := agent.BuildSystemMessages(systemPrompt, xmlContext, modelCfg.Provider, shouldCache)

	//  Run agent loop
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Vercel-Ai-Ui-Message-Stream", "v1")
	c.Header("X-Accel-Buffering", "no")

	sseWriter := sse.NewWriter(c.Writer)

	// Configure temperature
	var temperature *float64
	if cfg.Temperature > 0 {
		temperature = &cfg.Temperature
	}

	// Get tool definitions
	tools := agent.GetTools()

	err = agent.RunAgentLoop(
		sseWriter,
		modelCfg.ChatModel,
		systemMessages,
		modelMessages,
		tools,
		cfg.MaxOutputTokens,
		temperature,
		modelCfg.Provider,
	)

	if err != nil {
		log.Printf("[Chat] Agent loop error: %v", err)
		if !c.Writer.Written() {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		}
	}
}

// writeCachedResponse sends a cached response as SSE
func writeCachedResponse(c *gin.Context, xml string) {
	toolCallID := fmt.Sprintf("cached-%d", time.Now().UnixMilli())

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Vercel-Ai-Ui-Message-Stream", "v1")
	c.Header("X-Accel-Buffering", "no")

	w := sse.NewWriter(c.Writer)
	w.WriteStart("")
	w.WriteToolInputStart(toolCallID, "display_diagram")
	w.WriteToolInputDelta(toolCallID, xml)
	w.WriteToolInputAvailable(toolCallID, "display_diagram", map[string]string{"xml": xml})
	w.WriteFinish("stop", nil)
	w.WriteDone()
}

// writeErrorJSON writes an error response with appropriate status code
func writeErrorJSON(c *gin.Context, err error) {
	msg := err.Error()
	status := http.StatusInternalServerError

	// Check for auth errors
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "api key") || strings.Contains(lower, "authentication") ||
		strings.Contains(lower, "credential") {
		status = http.StatusUnauthorized
	}

	// Sanitize error message
	for _, kw := range []string{"key", "token", "secret", "password", "credential", "signature"} {
		if strings.Contains(lower, kw) {
			msg = "Authentication failed. Please check your credentials."
			break
		}
	}

	c.JSON(status, gin.H{"error": msg})
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
