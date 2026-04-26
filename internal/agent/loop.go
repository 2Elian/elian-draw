package agent

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/2Elian/next-ai-draw-io/go-backend/internal/config"
	"github.com/2Elian/next-ai-draw-io/go-backend/internal/model"
	"github.com/2Elian/next-ai-draw-io/go-backend/internal/provider"
	"github.com/2Elian/next-ai-draw-io/go-backend/internal/sse"
)

// Message represents a message in the agent loop
type Message = provider.Message

// ContentPart represents a content part of a message
type ContentPart = provider.ContentPart

const maxSteps = 5

// RunAgentLoop executes the custom tool-calling loop
func RunAgentLoop(
	sseWriter *sse.Writer,
	chatModel provider.ChatModel,
	systemMessages []Message,
	conversationMessages []Message,
	tools []provider.ToolDef,
	maxOutputTokens int,
	temperature *float64,
	providerName config.ProviderName,
) error {
	messageID := fmt.Sprintf("msg-%d", time.Now().UnixMilli())

	if err := sseWriter.WriteStart(messageID); err != nil {
		return fmt.Errorf("write start: %w", err)
	}

	allMessages := make([]Message, 0, len(systemMessages)+len(conversationMessages))
	allMessages = append(allMessages, systemMessages...)
	allMessages = append(allMessages, conversationMessages...)

	textIDCounter := 0
	finishReason := ""
	var usage *provider.Usage

	for step := 0; step < maxSteps; step++ {
		if err := sseWriter.WriteStartStep(); err != nil {
			return err
		}

		req := &provider.ChatRequest{
			Messages:        allMessages,
			Tools:           tools,
			MaxOutputTokens: maxOutputTokens,
			Temperature:     temperature,
		}

		resp, err := chatModel.Chat(req)
		if err != nil {
			sseWriter.WriteError(fmt.Sprintf("AI model error: %v", err))
			sseWriter.WriteFinish("error", nil)
			sseWriter.WriteDone()
			return nil
		}

		var toolCalls []model.ToolCall
		var currentToolCall *toolCallBuilder

		for event := range resp.Stream {
			switch event.Type {
			case provider.EventTextDelta:
				textIDCounter++
				textID := fmt.Sprintf("text-%d", textIDCounter)
				if err := sseWriter.WriteTextDelta(textID, event.TextDelta); err != nil {
					return err
				}

			case provider.EventReasoningDelta:
				textIDCounter++
				reasoningID := fmt.Sprintf("reasoning-%d", textIDCounter)
				if err := sseWriter.WriteReasoningDelta(reasoningID, event.ReasoningDelta); err != nil {
					return err
				}

			case provider.EventToolCallStart:
				currentToolCall = &toolCallBuilder{
					id:   event.ToolCallID,
					name: event.ToolName,
				}
				if err := sseWriter.WriteToolInputStart(event.ToolCallID, event.ToolName); err != nil {
					return err
				}

			case provider.EventToolCallArgsDelta:
				if currentToolCall != nil {
					currentToolCall.args.WriteString(event.ToolArgsDelta)
					if err := sseWriter.WriteToolInputDelta(currentToolCall.id, event.ToolArgsDelta); err != nil {
						return err
					}
				}

			case provider.EventToolCallComplete:
				if currentToolCall != nil {
					currentToolCall.args.WriteString(event.ToolArgsJSON)
					argsStr := currentToolCall.args.String()
					toolCalls = append(toolCalls, model.ToolCall{
						ID:       currentToolCall.id,
						Name:     currentToolCall.name,
						ArgsJSON: argsStr,
					})

					input := parseToolInput(currentToolCall.name, argsStr)
					if err := sseWriter.WriteToolInputAvailable(currentToolCall.id, currentToolCall.name, input); err != nil {
						return err
					}
					currentToolCall = nil
				}

			case provider.EventFinish:
				finishReason = event.FinishReason
				usage = event.Usage

			case provider.EventError:
				sseWriter.WriteError(event.Error.Error())
			}
		}

		if err := sseWriter.WriteFinishStep(); err != nil {
			return err
		}

		if len(toolCalls) == 0 {
			break
		}

		// Process tool calls: server-side first, then stop if client-side tools present
		hasServerTool := false
		hasClientTool := false

		for _, tc := range toolCalls {
			if ClientSideTools[tc.Name] {
				hasClientTool = true
				continue
			}

			// Server-side tool execution
			hasServerTool = true
			result, err := executeServerTool(tc.Name, tc.ArgsJSON)
			if err != nil {
				sseWriter.WriteToolOutputError(tc.ID, err.Error())
			} else {
				sseWriter.WriteToolOutputAvailable(tc.ID, result)
			}

			// Add to conversation for next iteration
			allMessages = append(allMessages, Message{
				Role: "assistant",
				Content: []ContentPart{
					{Type: "tool-call", ToolCallID: tc.ID, ToolName: tc.Name, ArgsJSON: tc.ArgsJSON},
				},
			})
			resultStr := fmt.Sprintf("%v", result)
			allMessages = append(allMessages, Message{
				Role: "tool",
				Content: []ContentPart{
					{Type: "tool-result", ToolCallID: tc.ID, Result: resultStr},
				},
			})
		}

		// If client-side tools were called, stop (frontend handles them)
		if hasClientTool {
			break
		}

		// If only server tools, continue the loop
		if !hasServerTool {
			break
		}
	}

	metadata := buildFinishMetadata(finishReason, usage)
	if err := sseWriter.WriteFinish(finishReason, metadata); err != nil {
		return err
	}

	return sseWriter.WriteDone()
}

type toolCallBuilder struct {
	id   string
	name string
	args strings.Builder
}

func parseToolInput(toolName, argsJSON string) any {
	var input any
	if err := json.Unmarshal([]byte(argsJSON), &input); err != nil {
		repaired := repairJSON(argsJSON)
		if err2 := json.Unmarshal([]byte(repaired), &input); err2 != nil {
			return map[string]any{"_raw": argsJSON}
		}
	}
	return input
}

func repairJSON(s string) string {
	s = strings.ReplaceAll(s, ":=", ": ")
	s = strings.ReplaceAll(s, "= \"", ": \"")
	return s
}

func executeServerTool(toolName, argsJSON string) (any, error) {
	switch toolName {
	case ToolGetShapeLibrary:
		var args struct {
			Library string `json:"library"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return nil, fmt.Errorf("parse arguments: %w", err)
		}
		result, err := GetShapeLibrary(args.Library)
		if err != nil {
			return nil, err
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unknown server tool: %s", toolName)
	}
}

func buildFinishMetadata(finishReason string, usage *provider.Usage) *model.FinishMessageMetadata {
	if usage == nil && finishReason == "" {
		return nil
	}
	meta := &model.FinishMessageMetadata{
		FinishReason: finishReason,
	}
	if usage != nil {
		meta.TotalTokens = usage.InputTokens + usage.OutputTokens
	}
	return meta
}

// ConvertUIMessagesToModelMessages converts frontend UIMessages to provider Messages
func ConvertUIMessagesToModelMessages(messages []model.UIMessage) []Message {
	result := make([]Message, 0, len(messages))

	for _, msg := range messages {
		m := Message{Role: msg.Role}

		if len(msg.Parts) == 0 {
			m.Content = []ContentPart{{Type: "text", Text: msg.Content}}
		} else {
			for _, part := range msg.Parts {
				switch part.Type {
				case "text":
					m.Content = append(m.Content, ContentPart{
						Type: "text",
						Text: part.Text,
					})
				case "file":
					m.Content = append(m.Content, ContentPart{
						Type:     "image",
						ImageURL: part.URL,
						MimeType: part.MediaType,
					})
				case "tool-invocation", "tool-call":
					m.Content = append(m.Content, ContentPart{
						Type:       "tool-call",
						ToolCallID: part.ToolCallID,
						ToolName:   part.ToolName,
					})
				case "tool-result":
					m.Content = append(m.Content, ContentPart{
						Type:       "tool-result",
						ToolCallID: part.ToolCallID,
						Result:     fmt.Sprintf("%v", part.Output),
					})
				}
			}
		}

		if len(m.Content) > 0 {
			result = append(result, m)
		}
	}

	return result
}

// ReplaceHistoricalToolInputs replaces large XML in tool call history with placeholders
func ReplaceHistoricalToolInputs(messages []Message) []Message {
	for i := range messages {
		if messages[i].Role != "assistant" {
			continue
		}
		for j := range messages[i].Content {
			part := &messages[i].Content[j]
			if part.Type == "tool-call" && (part.ToolName == ToolDisplayDiagram || part.ToolName == ToolEditDiagram) {
				part.ArgsJSON = `{"placeholder":"[XML content replaced - see current diagram XML in system context]"}`
			}
		}
	}
	return messages
}
