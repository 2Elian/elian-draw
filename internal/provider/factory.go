package provider

import (
	"fmt"
	"net/http"
	"os"

	"github.com/2Elian/next-ai-draw-io/go-backend/internal/config"
	"github.com/2Elian/next-ai-draw-io/go-backend/internal/model"
)

// ModelConfig holds the resolved AI model and its configuration
type ModelConfig struct {
	ChatModel       ChatModel
	ProviderOptions map[string]any
	Headers         map[string]string
	ModelID         string
	Provider        config.ProviderName
}

// ChatModel is the interface for AI chat models with tool calling support
type ChatModel interface {
	// Chat sends messages and returns a streaming response
	Chat(req *ChatRequest) (*ChatResponse, error)
}

// ChatRequest is the input to a ChatModel
type ChatRequest struct {
	Messages      []Message
	Tools         []ToolDef
	MaxOutputTokens int
	Temperature   *float64
	Abort         <-chan struct{}
}

// ChatResponse is the streaming response from a ChatModel
type ChatResponse struct {
	Stream <-chan StreamEvent
}

// StreamEvent represents a single event from the model's stream
type StreamEvent struct {
	Type StreamEventType
	// Text delta
	TextDelta string
	// Reasoning delta
	ReasoningDelta string
	// Text ID for text start/delta/end
	TextID string
	// Tool call fields
	ToolCallID   string
	ToolName     string
	ToolArgsDelta string
	ToolArgsJSON string
	// Finish fields
	FinishReason string
	Usage        *Usage
	// Error
	Error error
}

type StreamEventType int

const (
	EventTextDelta StreamEventType = iota
	EventReasoningDelta
	EventToolCallStart
	EventToolCallArgsDelta
	EventToolCallComplete
	EventFinish
	EventError
)

// Usage represents token usage
type Usage struct {
	InputTokens  int
	OutputTokens int
}

// Message represents a chat message in the model's format
type Message struct {
	Role    string    // "system", "user", "assistant", "tool"
	Content []ContentPart
}

// ContentPart is a part of a message's content
type ContentPart struct {
	Type     string // "text", "image", "tool-call", "tool-result"
	Text     string
	ImageURL string
	MimeType string
	// Tool call fields
	ToolCallID string
	ToolName   string
	ArgsJSON   string
	// Tool result fields
	Result string
}

// ToolDef defines a tool the model can call
type ToolDef struct {
	Name        string
	Description string
	InputSchema any // JSON Schema object
}

// GetAIModel creates an AI model based on the client overrides and config
func GetAIModel(overrides *model.ClientOverrides) (*ModelConfig, error) {
	cfg := config.AppConfig

	// Determine model ID
	modelID := overrides.ModelID
	if modelID == "" {
		modelID = cfg.AIModel
	}
	if modelID == "" {
		return nil, fmt.Errorf("AI_MODEL environment variable is required. Example: AI_MODEL=claude-sonnet-4-5")
	}

	// Determine provider
	var provider config.ProviderName
	if overrides.Provider != "" {
		if !config.AllowedClientProviders[overrides.Provider] {
			return nil, fmt.Errorf("invalid provider: %s", overrides.Provider)
		}
		provider = overrides.Provider
	} else if cfg.AIProvider != "" {
		provider = cfg.AIProvider
	} else {
		// Auto-detect provider
		detected := detectProvider()
		if detected != "" {
			provider = detected
		} else {
			return nil, fmt.Errorf("no AI provider configured. Set AI_PROVIDER or an API key environment variable")
		}
	}

	// SSRF protection
	if overrides.BaseURL != "" && overrides.APIKey == "" &&
		provider != config.ProviderEdgeOne &&
		!(provider == config.ProviderOllama && os.Getenv("OLLAMA_API_KEY") == "") &&
		!(provider == config.ProviderVertexAI && overrides.VertexAPIKey != "") {
		return nil, fmt.Errorf("API key is required when using a custom base URL")
	}

	// Resolve API key and base URL
	envVar := config.ProviderEnvVars[provider]
	apiKey := config.ResolveAPIKey(overrides.APIKey, overrides.APIKeyEnvs, envVar)

	var defaultBaseURL string
	if info, ok := config.ProviderInfoMap[provider]; ok {
		defaultBaseURL = info.DefaultBaseURL
	}
	baseURLEnvName := fmt.Sprintf("%s_BASE_URL", stringsToUpper(string(provider)))
	serverBaseURL := config.ResolveBaseURLEnv(baseURLEnvName)
	baseURL := config.ResolveBaseURL(overrides.APIKey, overrides.BaseURL, serverBaseURL, defaultBaseURL)

	// Create model based on provider
	var chatModel ChatModel
	var providerOptions map[string]any
	var headers map[string]string
	var err error

	switch provider {
	case config.ProviderOpenAI, config.ProviderDeepSeek, config.ProviderSiliconFlow,
		config.ProviderSGLang, config.ProviderGateway, config.ProviderEdgeOne,
		config.ProviderDoubao, config.ProviderModelScope,
		config.ProviderGLM, config.ProviderQwen, config.ProviderQiniu,
		config.ProviderKimi, config.ProviderNovita, config.ProviderOllama,
		config.ProviderOpenRouter:
		chatModel, err = NewOpenAICompatible(provider, apiKey, baseURL, modelID, overrides)
	case config.ProviderAnthropic:
		chatModel, err = NewAnthropicModel(apiKey, baseURL, modelID)
	case config.ProviderGoogle:
		chatModel, err = NewGoogleModel(apiKey, baseURL, modelID)
	case config.ProviderVertexAI:
		chatModel, err = NewVertexModel(overrides.VertexAPIKey, baseURL, modelID)
	case config.ProviderAzure:
		chatModel, err = NewAzureModel(apiKey, baseURL, modelID)
	case config.ProviderMiniMax:
		chatModel, err = NewMiniMaxModel(apiKey, baseURL, modelID)
	case config.ProviderBedrock:
		chatModel, err = NewBedrockModel(overrides, modelID)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create model for %s: %w", provider, err)
	}

	// Build provider-specific options
	providerOptions = buildProviderOptions(provider, modelID)

	// Anthropic beta headers
	if provider == config.ProviderAnthropic || provider == config.ProviderMiniMax {
		headers = map[string]string{
			"anthropic-beta": "fine-grained-tool-streaming-2025-05-14",
		}
	}

	return &ModelConfig{
		ChatModel:       chatModel,
		ProviderOptions: providerOptions,
		Headers:         headers,
		ModelID:         modelID,
		Provider:        provider,
	}, nil
}

func detectProvider() config.ProviderName {
	configuredCount := 0
	var lastProvider config.ProviderName

	for provider, envVar := range config.ProviderEnvVars {
		if envVar == "" {
			continue
		}
		if os.Getenv(envVar) != "" {
			// Azure requires additional config
			if provider == config.ProviderAzure {
				if os.Getenv("AZURE_BASE_URL") == "" && os.Getenv("AZURE_RESOURCE_NAME") == "" {
					continue
				}
			}
			configuredCount++
			lastProvider = provider
		}
	}

	if configuredCount == 1 {
		return lastProvider
	}
	return ""
}

func buildProviderOptions(provider config.ProviderName, modelID string) map[string]any {
	options := make(map[string]any)

	switch provider {
	case config.ProviderOpenAI:
		opts := make(map[string]any)
		if modelID != "" && (contains(modelID, "o1") || contains(modelID, "o3") ||
			contains(modelID, "o4") || contains(modelID, "gpt-5")) {
			opts["reasoningSummary"] = "auto"
		}
		if effort := os.Getenv("OPENAI_REASONING_EFFORT"); effort != "" {
			opts["reasoningEffort"] = effort
		}
		if len(opts) > 0 {
			options["openai"] = opts
		}

	case config.ProviderAnthropic:
		if budget := os.Getenv("ANTHROPIC_THINKING_BUDGET_TOKENS"); budget != "" {
			options["anthropic"] = map[string]any{
				"thinking": map[string]any{
					"type":         os.Getenv("ANTHROPIC_THINKING_TYPE"),
					"budgetTokens": budget,
				},
			}
		}

	case config.ProviderGoogle:
		if modelID != "" && (contains(modelID, "gemini-2") || contains(modelID, "gemini-3")) {
			opts := map[string]any{"includeThoughts": true}
			if contains(modelID, "2.5") || contains(modelID, "2-5") {
				if budget := os.Getenv("GOOGLE_THINKING_BUDGET"); budget != "" {
					opts["thinkingBudget"] = budget
				}
			}
			options["google"] = map[string]any{"thinkingConfig": opts}
		}
	}

	if len(options) == 0 {
		return nil
	}
	return options
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func stringsToUpper(s string) string {
	// Simple uppercase conversion for ASCII
	result := make([]byte, len(s))
	for i, c := range []byte(s) {
		if c >= 'a' && c <= 'z' {
			result[i] = c - 32
		} else {
			result[i] = c
		}
	}
	return string(result)
}

// stub implementations for providers not yet fully implemented
// These will be replaced with real implementations in Phase 4

func NewAnthropicModel(apiKey, baseURL, modelID string) (ChatModel, error) {
	return NewOpenAICompatible(config.ProviderAnthropic, apiKey, baseURL, modelID, nil)
}

func NewGoogleModel(apiKey, baseURL, modelID string) (ChatModel, error) {
	return NewOpenAICompatible(config.ProviderGoogle, apiKey, baseURL, modelID, nil)
}

func NewVertexModel(apiKey, baseURL, modelID string) (ChatModel, error) {
	if apiKey == "" {
		apiKey = os.Getenv("GOOGLE_VERTEX_API_KEY")
	}
	return NewOpenAICompatible(config.ProviderVertexAI, apiKey, baseURL, modelID, nil)
}

func NewAzureModel(apiKey, baseURL, modelID string) (ChatModel, error) {
	if baseURL == "" {
		if name := os.Getenv("AZURE_RESOURCE_NAME"); name != "" {
			baseURL = fmt.Sprintf("https://%s.openai.azure.com/openai/v1", name)
		}
	}
	return NewOpenAICompatible(config.ProviderAzure, apiKey, baseURL, modelID, nil)
}

func NewMiniMaxModel(apiKey, baseURL, modelID string) (ChatModel, error) {
	return NewOpenAICompatible(config.ProviderMiniMax, apiKey, baseURL, modelID, nil)
}

func NewBedrockModel(overrides *model.ClientOverrides, modelID string) (ChatModel, error) {
	return NewOpenAICompatible(config.ProviderBedrock, "", "", modelID, overrides)
}

// GetHTTPClient returns a shared HTTP client
var httpClient = &http.Client{}
