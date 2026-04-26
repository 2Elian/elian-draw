package provider

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/2Elian/next-ai-draw-io/go-backend/internal/config"
	"github.com/2Elian/next-ai-draw-io/go-backend/internal/model"
)

// OpenAICompatibleModel implements ChatModel using the OpenAI Chat Completions API
// This covers: OpenAI, DeepSeek, SiliconFlow, SGLang, Gateway, EdgeOne, Doubao,
// ModelScope, GLM, Qwen, Qiniu, Kimi, Novita, Ollama, OpenRouter
type OpenAICompatibleModel struct {
	apiKey  string
	baseURL string
	modelID string
	headers map[string]string
}

// NewOpenAICompatible creates a new OpenAI-compatible model
func NewOpenAICompatible(provider config.ProviderName, apiKey, baseURL, modelID string, overrides *model.ClientOverrides) (*OpenAICompatibleModel, error) {
	m := &OpenAICompatibleModel{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		modelID: modelID,
	}

	// Ensure base URL ends with the correct path
	if !strings.HasSuffix(m.baseURL, "/chat/completions") &&
		!strings.HasSuffix(m.baseURL, "/v1") &&
		!strings.HasSuffix(m.baseURL, "/v1/") {
		// Don't modify for providers that have specific paths
	}

	return m, nil
}

// openAIRequest is the request body for OpenAI Chat Completions API
type openAIRequest struct {
	Model          string          `json:"model"`
	Messages       []openAIMessage `json:"messages"`
	Stream         bool            `json:"stream"`
	StreamOptions  *streamOptions  `json:"stream_options,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	Temperature    *float64        `json:"temperature,omitempty"`
	Tools          []openAITool    `json:"tools,omitempty"`
}

type streamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type openAIMessage struct {
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
}

type openAIToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function openAIFunction   `json:"function"`
}

type openAIFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAITool struct {
	Type     string          `json:"type"`
	Function openAIToolFunc  `json:"function"`
}

type openAIToolFunc struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// Chat sends a request to the OpenAI-compatible API and returns a streaming response
func (m *OpenAICompatibleModel) Chat(req *ChatRequest) (*ChatResponse, error) {
	// Build request messages
	messages := make([]openAIMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		oaiMsg := openAIMessage{Role: msg.Role}

		// Handle content parts
		if len(msg.Content) == 1 && msg.Content[0].Type == "text" {
			oaiMsg.Content = json.RawMessage(`"` + jsonEscape(msg.Content[0].Text) + `"`)
		} else if len(msg.Content) > 0 {
			parts := make([]any, 0, len(msg.Content))
			for _, part := range msg.Content {
				switch part.Type {
				case "text":
					parts = append(parts, map[string]any{
						"type": "text",
						"text": part.Text,
					})
				case "image":
					parts = append(parts, map[string]any{
						"type":      "image_url",
						"image_url": map[string]any{"url": part.ImageURL},
					})
				case "tool-call":
					oaiMsg.ToolCalls = append(oaiMsg.ToolCalls, openAIToolCall{
						ID:   part.ToolCallID,
						Type: "function",
						Function: openAIFunction{
							Name:      part.ToolName,
							Arguments: part.ArgsJSON,
						},
					})
				case "tool-result":
					oaiMsg.Role = "tool"
					oaiMsg.ToolCallID = part.ToolCallID
					oaiMsg.Content = json.RawMessage(`"` + jsonEscape(part.Result) + `"`)
				}
			}
			if len(parts) > 0 && oaiMsg.Role != "tool" {
				data, _ := json.Marshal(parts)
				oaiMsg.Content = data
			}
		}

		if len(oaiMsg.Content) == 0 && len(oaiMsg.ToolCalls) == 0 && msg.Role != "assistant" {
			oaiMsg.Content = json.RawMessage(`""`)
		} else if len(oaiMsg.Content) == 0 && msg.Role == "assistant" && len(oaiMsg.ToolCalls) > 0 {
			oaiMsg.Content = json.RawMessage(`null`)
		}

		messages = append(messages, oaiMsg)
	}

	// Build tools
	tools := make([]openAITool, 0, len(req.Tools))
	for _, t := range req.Tools {
		schemaData, _ := json.Marshal(t.InputSchema)
		tools = append(tools, openAITool{
			Type: "function",
			Function: openAIToolFunc{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  schemaData,
			},
		})
	}

	body := openAIRequest{
		Model:    m.modelID,
		Messages: messages,
		Stream:   true,
		StreamOptions: &streamOptions{IncludeUsage: true},
		Tools:    tools,
	}

	if req.MaxOutputTokens > 0 {
		body.MaxTokens = req.MaxOutputTokens
	}
	if req.Temperature != nil {
		body.Temperature = req.Temperature
	}

	bodyData, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Build URL
	url := m.baseURL
	if !strings.Contains(url, "/chat/completions") {
		url = strings.TrimRight(url, "/") + "/chat/completions"
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(bodyData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if m.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+m.apiKey)
	}
	for k, v := range m.headers {
		httpReq.Header.Set(k, v)
	}

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	// Parse SSE stream
	eventCh := make(chan StreamEvent, 100)
	go m.parseSSEStream(resp.Body, eventCh)

	return &ChatResponse{Stream: eventCh}, nil
}

func (m *OpenAICompatibleModel) parseSSEStream(body io.ReadCloser, eventCh chan<- StreamEvent) {
	defer close(eventCh)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var currentToolCalls map[int]*toolCallAccumulator

	for scanner.Scan() {
		line := scanner.Text()

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) == 0 {
			// Usage-only chunk
			if chunk.Usage != nil {
				eventCh <- StreamEvent{
					Type:  EventFinish,
					Usage: &Usage{InputTokens: chunk.Usage.PromptTokens, OutputTokens: chunk.Usage.CompletionTokens},
				}
			}
			continue
		}

		choice := chunk.Choices[0]
		delta := choice.Delta

		// Handle content (text)
		if delta.Content != "" {
			eventCh <- StreamEvent{
				Type:      EventTextDelta,
				TextDelta: delta.Content,
				TextID:    "text-" + chunk.ID,
			}
		}

		// Handle reasoning
		if delta.ReasoningContent != nil && *delta.ReasoningContent != "" {
			eventCh <- StreamEvent{
				Type:           EventReasoningDelta,
				ReasoningDelta: *delta.ReasoningContent,
				TextID:         "reasoning-" + chunk.ID,
			}
		}

		// Handle tool calls
		if len(delta.ToolCalls) > 0 {
			for _, tc := range delta.ToolCalls {
				idx := tc.Index
				if currentToolCalls == nil {
					currentToolCalls = make(map[int]*toolCallAccumulator)
				}
				acc, exists := currentToolCalls[idx]
				if !exists {
					acc = &toolCallAccumulator{}
					currentToolCalls[idx] = acc
				}

				if tc.ID != "" {
					acc.id = tc.ID
					acc.name = tc.Function.Name
					eventCh <- StreamEvent{
						Type:       EventToolCallStart,
						ToolCallID: acc.id,
						ToolName:   acc.name,
					}
				}

				if tc.Function.Arguments != "" {
					acc.args.WriteString(tc.Function.Arguments)
					eventCh <- StreamEvent{
						Type:           EventToolCallArgsDelta,
						ToolCallID:     acc.id,
						ToolArgsDelta:  tc.Function.Arguments,
					}
				}
			}
		}

		// Handle finish
		if choice.FinishReason != nil && *choice.FinishReason != "" {
			// Flush any remaining tool calls
			for _, acc := range currentToolCalls {
				eventCh <- StreamEvent{
					Type:         EventToolCallComplete,
					ToolCallID:   acc.id,
					ToolName:     acc.name,
					ToolArgsJSON: acc.args.String(),
				}
			}
			currentToolCalls = nil

			finishReason := *choice.FinishReason
			if finishReason == "tool_calls" {
				finishReason = "stop"
			}

			eventCh <- StreamEvent{
				Type:         EventFinish,
				FinishReason: finishReason,
			}
		}
	}
}

type toolCallAccumulator struct {
	id   string
	name string
	args strings.Builder
}

// OpenAI SSE stream types

type openAIStreamChunk struct {
	ID      string              `json:"id"`
	Choices []openAIStreamChoice `json:"choices"`
	Usage   *openAIUsage        `json:"usage,omitempty"`
}

type openAIStreamChoice struct {
	Index        int                `json:"index"`
	Delta        openAIStreamDelta  `json:"delta"`
	FinishReason *string            `json:"finish_reason"`
}

type openAIStreamDelta struct {
	Role             string               `json:"role,omitempty"`
	Content          string               `json:"content,omitempty"`
	ReasoningContent *string              `json:"reasoning_content,omitempty"`
	ToolCalls        []openAIStreamToolCall `json:"tool_calls,omitempty"`
}

type openAIStreamToolCall struct {
	Index    int                      `json:"index"`
	ID       string                   `json:"id,omitempty"`
	Type     string                   `json:"type,omitempty"`
	Function openAIStreamToolFunction `json:"function"`
}

type openAIStreamToolFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}

type openAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
}

func jsonEscape(s string) string {
	var buf strings.Builder
	buf.Grow(len(s) + 10)
	for _, r := range s {
		switch r {
		case '"':
			buf.WriteString(`\"`)
		case '\\':
			buf.WriteString(`\\`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		default:
			buf.WriteRune(r)
		}
	}
	return buf.String()
}
