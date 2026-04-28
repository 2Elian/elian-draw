package model

import "github.com/2Elian/next-ai-draw-io/go-backend/internal/config"

// UIMessage represents a chat message from the frontend
type UIMessage struct {
	Role    string   `json:"role"`
	Parts   []UIPart `json:"parts"`             // 如果是非纯文本聊天 那么用户的text-input 将被塞入UIPart中
	Content string   `json:"content,omitempty"` // 如果是纯文本聊天 那么parts为空
}

// UIPart represents a part of a UIMessage
/*
处理工具调用的生命周期:
	    "parts": [
		  {
			"type": "tool-invocation",
			"toolName": "searchWeb",
			"toolCallId": "call_123",
			"state": "call",        // 状态：发起调用
			"input": { "query": "Go语言" }
		  },
		  {
			"type": "tool-invocation",
			"toolName": "searchWeb",
			"toolCallId": "call_123",
			"state": "result",      // 状态：调用完成
			"output": { "result": "Go是谷歌开发的语言..." }
		  }
		]

*/
type UIPart struct {
	Type       string `json:"type"`
	Text       string `json:"text,omitempty"`
	URL        string `json:"url,omitempty"`
	MediaType  string `json:"mediaType,omitempty"`
	ToolName   string `json:"toolName,omitempty"`
	ToolCallID string `json:"toolCallId,omitempty"`
	Input      any    `json:"input,omitempty"`
	Output     any    `json:"output,omitempty"`
	State      string `json:"state,omitempty"` // "partial-input", "call", "result"
}

// ChatRequest is the request body for POST /api/chat
type ChatRequest struct {
	Messages            []UIMessage `json:"messages"`
	XML                 string      `json:"xml"`         // 搞清楚这个是什么
	PreviousXML         string      `json:"previousXml"` // 搞清楚这个是什么
	SessionID           string      `json:"sessionId"`
	CustomSystemMessage string      `json:"customSystemMessage"` // 用户自定义的一些信息，比如：配色方案等
}

// ClientOverrides represents client-side AI provider overrides from request headers
type ClientOverrides struct {
	Provider config.ProviderName
	BaseURL  string
	APIKey   string
	ModelID  string
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
	TotalTokens  int    `json:"totalTokens"`
	FinishReason string `json:"finishReason"`
}
