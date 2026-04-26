package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ProviderName represents an AI provider identifier
type ProviderName string

const (
	ProviderOpenAI      ProviderName = "openai"
	ProviderAnthropic   ProviderName = "anthropic"
	ProviderGoogle      ProviderName = "google"
	ProviderVertexAI    ProviderName = "vertexai"
	ProviderAzure       ProviderName = "azure"
	ProviderBedrock     ProviderName = "bedrock"
	ProviderOllama      ProviderName = "ollama"
	ProviderOpenRouter  ProviderName = "openrouter"
	ProviderDeepSeek    ProviderName = "deepseek"
	ProviderSiliconFlow ProviderName = "siliconflow"
	ProviderSGLang      ProviderName = "sglang"
	ProviderGateway     ProviderName = "gateway"
	ProviderEdgeOne     ProviderName = "edgeone"
	ProviderDoubao      ProviderName = "doubao"
	ProviderModelScope  ProviderName = "modelscope"
	ProviderGLM         ProviderName = "glm"
	ProviderQwen        ProviderName = "qwen"
	ProviderQiniu       ProviderName = "qiniu"
	ProviderKimi        ProviderName = "kimi"
	ProviderMiniMax     ProviderName = "minimax"
	ProviderNovita      ProviderName = "novita"
)

// AllowedClientProviders are providers that can be selected from client settings
var AllowedClientProviders = map[ProviderName]bool{
	ProviderOpenAI: true, ProviderAnthropic: true, ProviderGoogle: true,
	ProviderVertexAI: true, ProviderAzure: true, ProviderBedrock: true,
	ProviderOpenRouter: true, ProviderDeepSeek: true, ProviderSiliconFlow: true,
	ProviderSGLang: true, ProviderGateway: true, ProviderEdgeOne: true,
	ProviderOllama: true, ProviderDoubao: true, ProviderModelScope: true,
	ProviderGLM: true, ProviderQwen: true, ProviderQiniu: true,
	ProviderKimi: true, ProviderMiniMax: true, ProviderNovita: true,
}

// SingleSystemProviders only support a single system message
var SingleSystemProviders = map[ProviderName]bool{
	ProviderMiniMax: true,
	ProviderGLM:     true,
	ProviderQwen:    true,
	ProviderKimi:    true,
	ProviderQiniu:   true,
	ProviderNovita:  true,
}

// ProviderInfo contains metadata about a provider
type ProviderInfo struct {
	Label          string
	DefaultBaseURL string
}

// ProviderInfoMap maps provider names to their metadata
var ProviderInfoMap = map[ProviderName]ProviderInfo{
	ProviderOpenAI:      {Label: "OpenAI", DefaultBaseURL: "https://api.openai.com/v1"},
	ProviderAnthropic:   {Label: "Anthropic", DefaultBaseURL: "https://api.anthropic.com/v1"},
	ProviderGoogle:      {Label: "Google", DefaultBaseURL: "https://generativelanguage.googleapis.com/v1beta"},
	ProviderVertexAI:    {Label: "Google Vertex AI"},
	ProviderAzure:       {Label: "Azure OpenAI", DefaultBaseURL: "https://your-resource.openai.azure.com/openai"},
	ProviderBedrock:     {Label: "Amazon Bedrock"},
	ProviderOllama:      {Label: "Ollama", DefaultBaseURL: "https://ollama.com/api"},
	ProviderOpenRouter:  {Label: "OpenRouter", DefaultBaseURL: "https://openrouter.ai/api/v1"},
	ProviderDeepSeek:    {Label: "DeepSeek", DefaultBaseURL: "https://api.deepseek.com/v1"},
	ProviderSiliconFlow: {Label: "SiliconFlow", DefaultBaseURL: "https://api.siliconflow.cn/v1"},
	ProviderSGLang:      {Label: "SGLang", DefaultBaseURL: "http://127.0.0.1:8000/v1"},
	ProviderGateway:     {Label: "AI Gateway", DefaultBaseURL: "https://ai-gateway.vercel.sh/v1/ai"},
	ProviderEdgeOne:     {Label: "EdgeOne Pages"},
	ProviderDoubao:      {Label: "Doubao (ByteDance)", DefaultBaseURL: "https://ark.cn-beijing.volces.com/api/v3"},
	ProviderModelScope:  {Label: "ModelScope", DefaultBaseURL: "https://api-inference.modelscope.cn/v1"},
	ProviderGLM:         {Label: "GLM (Zhipu)", DefaultBaseURL: "https://open.bigmodel.cn/api/paas/v4"},
	ProviderQwen:        {Label: "Qwen (Alibaba)", DefaultBaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1"},
	ProviderQiniu:       {Label: "Qiniu", DefaultBaseURL: "https://api.qnaigc.com/v1"},
	ProviderKimi:        {Label: "Kimi (Moonshot)", DefaultBaseURL: "https://api.moonshot.cn/v1"},
	ProviderMiniMax:     {Label: "MiniMax", DefaultBaseURL: "https://api.minimaxi.com/anthropic"},
	ProviderNovita:      {Label: "Novita AI", DefaultBaseURL: "https://api.novita.ai/openai"},
}

// ProviderEnvVars maps provider names to their default API key environment variable
var ProviderEnvVars = map[ProviderName]string{
	ProviderOpenAI:      "OPENAI_API_KEY",
	ProviderAnthropic:   "ANTHROPIC_API_KEY",
	ProviderGoogle:      "GOOGLE_GENERATIVE_AI_API_KEY",
	ProviderVertexAI:    "GOOGLE_VERTEX_API_KEY",
	ProviderAzure:       "AZURE_API_KEY",
	ProviderOpenRouter:  "OPENROUTER_API_KEY",
	ProviderDeepSeek:    "DEEPSEEK_API_KEY",
	ProviderSiliconFlow: "SILICONFLOW_API_KEY",
	ProviderSGLang:      "SGLANG_API_KEY",
	ProviderGateway:     "AI_GATEWAY_API_KEY",
	ProviderDoubao:      "DOUBAO_API_KEY",
	ProviderModelScope:  "MODELSCOPE_API_KEY",
	ProviderGLM:         "GLM_API_KEY",
	ProviderQwen:        "QWEN_API_KEY",
	ProviderQiniu:       "QINIU_API_KEY",
	ProviderKimi:        "KIMI_API_KEY",
	ProviderMiniMax:     "MINIMAX_API_KEY",
	ProviderNovita:      "NOVITA_API_KEY",
}

// ServerProviderConfig represents a provider config from ai-models.json
type ServerProviderConfig struct {
	Name      string   `json:"name"`
	Provider  string   `json:"provider"`
	Models    []string `json:"models"`
	APIKeyEnv any      `json:"apiKeyEnv,omitempty"` // string or []string
	BaseURL   string   `json:"baseUrlEnv,omitempty"`
	Default   bool     `json:"default,omitempty"`
}

// ServerModelsConfig represents the top-level ai-models.json structure
type ServerModelsConfig struct {
	Providers []ServerProviderConfig `json:"providers"`
}

// FlattenedServerModel is a resolved server model
type FlattenedServerModel struct {
	ID            string   // "server:<slug>:<modelId>"
	ModelID       string
	Provider      ProviderName
	ProviderLabel string
	IsDefault     bool
	APIKeyEnv     []string // resolved to always be a slice
	BaseURLEnv    string
}

// Config holds all application configuration
type Config struct {
	Port             string
	AccessCodeList   []string
	MaxOutputTokens  int
	Temperature      float64
	EnablePDFInput   bool
	EnableHistoryReplace bool
	IsDev            bool

	// AI provider defaults
	AIProvider ProviderName
	AIModel    string

	// Server models config
	serverModels     []FlattenedServerModel
	serverModelsOnce sync.Once
}

// AppConfig is the global configuration instance
var AppConfig *Config

// LoadConfig initializes configuration from environment variables
func LoadConfig() *Config {
	c := &Config{
		Port:     getEnv("PORT", "3001"),
		IsDev:    os.Getenv("NODE_ENV") == "development",
		AIModel:  os.Getenv("AI_MODEL"),
		MaxOutputTokens: getEnvInt("MAX_OUTPUT_TOKENS", 0),
		EnablePDFInput:  os.Getenv("ENABLE_PDF_INPUT") == "true",
		EnableHistoryReplace: os.Getenv("ENABLE_HISTORY_XML_REPLACE") == "true",
	}

	// Parse temperature
	if temp := os.Getenv("TEMPERATURE"); temp != "" {
		fmt.Sscanf(temp, "%f", &c.Temperature)
	}

	// Parse access codes
	if codes := os.Getenv("ACCESS_CODE_LIST"); codes != "" {
		for _, code := range strings.Split(codes, ",") {
			if trimmed := strings.TrimSpace(code); trimmed != "" {
				c.AccessCodeList = append(c.AccessCodeList, trimmed)
			}
		}
	}

	// Parse AI provider
	if provider := os.Getenv("AI_PROVIDER"); provider != "" {
		c.AIProvider = ProviderName(provider)
	}

	AppConfig = c
	return c
}

// LoadServerModels loads server model configuration from ai-models.json or env
func (c *Config) LoadServerModels() {
	c.serverModelsOnce.Do(func() {
		cfg := c.loadRawServerModelsConfig()
		if cfg == nil {
			c.serverModels = []FlattenedServerModel{}
			return
		}

		defaultProvider := c.AIProvider
		defaultModelID := c.AIModel

		for _, p := range cfg.Providers {
			providerLabel := p.Name
			if info, ok := ProviderInfoMap[ProviderName(p.Provider)]; ok && p.Name == "" {
				providerLabel = info.Label
			}
			nameSlug := slugify(p.Name)

			for _, modelID := range p.Models {
				isDefault := (p.Default && modelID == p.Models[0]) ||
					(defaultModelID != "" && modelID == defaultModelID &&
						(defaultProvider == "" || string(defaultProvider) == p.Provider))

				apiKeyEnvs := resolveAPIKeyEnvs(p.APIKeyEnv)

				c.serverModels = append(c.serverModels, FlattenedServerModel{
					ID:            fmt.Sprintf("server:%s:%s", nameSlug, modelID),
					ModelID:       modelID,
					Provider:      ProviderName(p.Provider),
					ProviderLabel: providerLabel,
					IsDefault:     isDefault,
					APIKeyEnv:     apiKeyEnvs,
					BaseURLEnv:    p.BaseURL,
				})
			}
		}
	})
}

// FindServerModelByID finds a server model by its ID
func (c *Config) FindServerModelByID(modelID string) *FlattenedServerModel {
	c.LoadServerModels()
	for i := range c.serverModels {
		if c.serverModels[i].ID == modelID {
			return &c.serverModels[i]
		}
	}
	return nil
}

// GetServerModels returns all flattened server models
func (c *Config) GetServerModels() []FlattenedServerModel {
	c.LoadServerModels()
	return c.serverModels
}

func (c *Config) loadRawServerModelsConfig() *ServerModelsConfig {
	// Priority 1: AI_MODELS_CONFIG env var
	if envConfig := os.Getenv("AI_MODELS_CONFIG"); strings.TrimSpace(envConfig) != "" {
		var cfg ServerModelsConfig
		if err := json.Unmarshal([]byte(envConfig), &cfg); err == nil {
			return &cfg
		}
	}

	// Priority 2: ai-models.json file
	configPath := os.Getenv("AI_MODELS_CONFIG_PATH")
	if configPath == "" {
		configPath = filepath.Join(".", "ai-models.json")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil
	}

	var cfg ServerModelsConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil
	}
	return &cfg
}

// ResolveAPIKey resolves the API key from overrides or environment variables
func ResolveAPIKey(userAPIKey string, apiKeyEnvs []string, defaultEnvVar string) string {
	if userAPIKey != "" {
		return userAPIKey
	}
	if len(apiKeyEnvs) > 0 {
		// Filter to only env vars that have values
		var valid []string
		for _, envVar := range apiKeyEnvs {
			if v := os.Getenv(envVar); v != "" {
				valid = append(valid, v)
			}
		}
		if len(valid) > 0 {
			return valid[0] // TODO: random load balancing
		}
	}
	if defaultEnvVar != "" {
		return os.Getenv(defaultEnvVar)
	}
	return ""
}

// ResolveBaseURL resolves the base URL considering user vs server configuration
func ResolveBaseURL(userAPIKey, userBaseURL, serverBaseURL, defaultBaseURL string) string {
	if userAPIKey != "" {
		if userBaseURL != "" {
			return userBaseURL
		}
		if defaultBaseURL != "" {
			return defaultBaseURL
		}
		return ""
	}
	if userBaseURL != "" {
		return userBaseURL
	}
	if serverBaseURL != "" {
		return serverBaseURL
	}
	if defaultBaseURL != "" {
		return defaultBaseURL
	}
	return ""
}

// ResolveBaseURLEnv resolves base URL from environment variable
func ResolveBaseURLEnv(envName string) string {
	if envName == "" {
		return ""
	}
	return os.Getenv(envName)
}

// SupportsImageInput checks if a model supports image/vision input
func SupportsImageInput(modelID string) bool {
	lower := strings.ToLower(modelID)
	hasVision := strings.Contains(lower, "vision") || strings.Contains(lower, "vl")

	// Kimi K2 doesn't support images
	if (strings.Contains(lower, "kimi-k2") || strings.Contains(lower, "kimi_k2")) && !hasVision &&
		!strings.Contains(lower, "2.5") && !strings.Contains(lower, "k2.5") {
		return false
	}
	if strings.Contains(lower, "moonshot-v1") && !hasVision {
		return false
	}
	if strings.Contains(lower, "minimax") && !hasVision {
		return false
	}
	if strings.Contains(lower, "deepseek") && !hasVision {
		return false
	}
	if strings.Contains(lower, "qwen") && !hasVision &&
		!strings.Contains(lower, "qwen3.5") && !strings.Contains(lower, "qvq") {
		return false
	}
	if strings.Contains(lower, "glm") && !hasVision {
		return true // Simplified: allow by default
	}

	return true
}

// SupportsPromptCaching checks if a model supports prompt caching
func SupportsPromptCaching(modelID string) bool {
	return strings.Contains(modelID, "claude") ||
		strings.Contains(modelID, "anthropic") ||
		strings.HasPrefix(modelID, "us.anthropic") ||
		strings.HasPrefix(modelID, "eu.anthropic")
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var result strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			result.WriteRune(r)
		} else {
			result.WriteByte('-')
		}
	}
	str := strings.Trim(result.String(), "-")
	return str
}

func resolveAPIKeyEnvs(raw any) []string {
	switch v := raw.(type) {
	case string:
		return []string{v}
	case []string:
		return v
	case []interface{}:
		var result []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil {
			return n
		}
	}
	return fallback
}
