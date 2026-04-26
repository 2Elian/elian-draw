package model

import "github.com/2Elian/next-ai-draw-io/go-backend/internal/config"

// UIMessage represents a chat message from the frontend
type UIMessage struct {
	Role    string     `json:"role"`
	Parts   []UIPart   `json:"parts"`
	Content string     `json:"content,omitempty"`
}

// UIPart represents a part of a UIMessage
type UIPart struct {
	Type         string     `json:"type"`
	Text         string     `json:"text,omitempty"`
	URL          string     `json:"url,omitempty"`
	MediaType    string     `json:"mediaType,omitempty"`
	ToolName     string     `json:"toolName,omitempty"`
	ToolCallID   string     `json:"toolCallId,omitempty"`
	Input        any        `json:"input,omitempty"`
	Output       any        `json:"output,omitempty"`
	State        string     `json:"state,omitempty"` // "partial-input", "call", "result"
}

// ChatRequest is the request body for POST /api/chat
type ChatRequest struct {
	Messages           []UIMessage `json:"messages"`
	XML                string      `json:"xml"`
	PreviousXML        string      `json:"previousXml"`
	SessionID          string      `json:"sessionId"`
	CustomSystemMessage string     `json:"customSystemMessage"`
}

// ClientOverrides represents client-side AI provider overrides from request headers
type ClientOverrides struct {
	Provider         config.ProviderName
	BaseURL          string
	APIKey           string
	ModelID          string
	// AWS Bedrock
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	AWSRegion          string
	AWSSessionToken    string
	// Vertex AI
	VertexAPIKey string
	// Server model config
	APIKeyEnvs []string
	BaseURLEnv string
	// Custom headers
	Headers map[string]string
}

// ToolCall represents an AI model's tool call
type ToolCall struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ArgsJSON string `json:"args,omitempty"`
}

// FinishMessageMetadata is attached to the finish event
type FinishMessageMetadata struct {
	TotalTokens int    `json:"totalTokens"`
	FinishReason string `json:"finishReason"`
}
