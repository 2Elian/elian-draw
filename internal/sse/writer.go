package sse

import (
	"encoding/json"
	"fmt"
	"io"
)

// UIMessageChunk types - matching Vercel AI SDK v6 protocol
// Format: `data: <JSON>\n\n` with `data: [DONE]\n\n` at the end
// Each JSON object has a "type" field identifying the event.

type ChunkType string

const (
	TypeStart              ChunkType = "start"
	TypeTextStart          ChunkType = "text-start"
	TypeTextDelta          ChunkType = "text-delta"
	TypeTextEnd            ChunkType = "text-end"
	TypeReasoningStart     ChunkType = "reasoning-start"
	TypeReasoningDelta     ChunkType = "reasoning-delta"
	TypeReasoningEnd       ChunkType = "reasoning-end"
	TypeToolInputStart     ChunkType = "tool-input-start"
	TypeToolInputDelta     ChunkType = "tool-input-delta"
	TypeToolInputAvailable ChunkType = "tool-input-available"
	TypeToolInputError     ChunkType = "tool-input-error"
	TypeToolOutputAvail  ChunkType = "tool-output-available"
	TypeToolOutputErr    ChunkType = "tool-output-error"
	TypeStartStep          ChunkType = "start-step"
	TypeFinishStep         ChunkType = "finish-step"
	TypeFinish             ChunkType = "finish"
	TypeError              ChunkType = "error"
	TypeAbort              ChunkType = "abort"
	TypeMessageMetadata    ChunkType = "message-metadata"
)

// UIMessageChunk is the base type for all SSE chunks
type UIMessageChunk struct {
	Type ChunkType `json:"type"`
}

// StartChunk initiates the stream
type StartChunk struct {
	Type           ChunkType     `json:"type"`
	MessageID      string        `json:"messageId,omitempty"`
	MessageMetadata any          `json:"messageMetadata,omitempty"`
}

// TextDeltaChunk streams text content
type TextDeltaChunk struct {
	Type  ChunkType `json:"type"`
	ID    string    `json:"id"`
	Delta string    `json:"delta"`
}

// ReasoningDeltaChunk streams reasoning content
type ReasoningDeltaChunk struct {
	Type  ChunkType `json:"type"`
	ID    string    `json:"id"`
	Delta string    `json:"delta"`
}

// ToolInputStartChunk begins a tool call
type ToolInputStartChunk struct {
	Type             ChunkType `json:"type"`
	ToolCallID       string    `json:"toolCallId"`
	ToolName         string    `json:"toolName"`
	ProviderExecuted *bool     `json:"providerExecuted,omitempty"`
}

// ToolInputDeltaChunk streams tool call input
type ToolInputDeltaChunk struct {
	Type           ChunkType `json:"type"`
	ToolCallID     string    `json:"toolCallId"`
	InputTextDelta string    `json:"inputTextDelta"`
}

// ToolInputAvailableChunk signals complete tool input
type ToolInputAvailableChunk struct {
	Type             ChunkType `json:"type"`
	ToolCallID       string    `json:"toolCallId"`
	ToolName         string    `json:"toolName"`
	Input            any       `json:"input"`
	ProviderExecuted *bool     `json:"providerExecuted,omitempty"`
}

// ToolOutputAvailableChunk provides tool execution result
type ToolOutputAvailableChunk struct {
	Type             ChunkType `json:"type"`
	ToolCallID       string    `json:"toolCallId"`
	Output           any       `json:"output"`
	ProviderExecuted *bool     `json:"providerExecuted,omitempty"`
}

// ToolOutputErrorChunk signals tool execution error
type ToolOutputErrorChunk struct {
	Type       ChunkType `json:"type"`
	ToolCallID string    `json:"toolCallId"`
	ErrorText  string    `json:"errorText"`
}

// FinishChunk ends the stream
type FinishChunk struct {
	Type           ChunkType `json:"type"`
	FinishReason   string    `json:"finishReason,omitempty"`
	MessageMetadata any      `json:"messageMetadata,omitempty"`
}

// ErrorChunk sends an error
type ErrorChunk struct {
	Type       ChunkType `json:"type"`
	ErrorText  string    `json:"errorText"`
}

// Writer writes UIMessageStreamResponse SSE events to an io.Writer
type Writer struct {
	w io.Writer
}

// NewWriter creates a new SSE writer
func NewWriter(w io.Writer) *Writer {
	return &Writer{w: w}
}

// WriteChunk serializes and writes a chunk as an SSE event
func (w *Writer) WriteChunk(chunk any) error {
	data, err := json.Marshal(chunk)
	if err != nil {
		return fmt.Errorf("marshal SSE chunk: %w", err)
	}
	_, err = fmt.Fprintf(w.w, "data: %s\n\n", data)
	return err
}

// WriteDone sends the stream termination signal
func (w *Writer) WriteDone() error {
	_, err := fmt.Fprintf(w.w, "data: [DONE]\n\n")
	return err
}

// Helper methods for common chunk types

func (w *Writer) WriteStart(messageID string) error {
	return w.WriteChunk(StartChunk{
		Type:      TypeStart,
		MessageID: messageID,
	})
}

func (w *Writer) WriteTextStart(id string) error {
	return w.WriteChunk(UIMessageChunk{Type: TypeTextStart})
}

func (w *Writer) WriteTextDelta(id, delta string) error {
	return w.WriteChunk(TextDeltaChunk{
		Type:  TypeTextDelta,
		ID:    id,
		Delta: delta,
	})
}

func (w *Writer) WriteTextEnd(id string) error {
	return w.WriteChunk(UIMessageChunk{Type: TypeTextEnd})
}

func (w *Writer) WriteReasoningDelta(id, delta string) error {
	return w.WriteChunk(ReasoningDeltaChunk{
		Type:  TypeReasoningDelta,
		ID:    id,
		Delta: delta,
	})
}

func (w *Writer) WriteToolInputStart(toolCallID, toolName string) error {
	return w.WriteChunk(ToolInputStartChunk{
		Type:       TypeToolInputStart,
		ToolCallID: toolCallID,
		ToolName:   toolName,
	})
}

func (w *Writer) WriteToolInputDelta(toolCallID, delta string) error {
	return w.WriteChunk(ToolInputDeltaChunk{
		Type:           TypeToolInputDelta,
		ToolCallID:     toolCallID,
		InputTextDelta: delta,
	})
}

func (w *Writer) WriteToolInputAvailable(toolCallID, toolName string, input any) error {
	return w.WriteChunk(ToolInputAvailableChunk{
		Type:       TypeToolInputAvailable,
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Input:      input,
	})
}

func (w *Writer) WriteToolOutputAvailable(toolCallID string, output any) error {
	return w.WriteChunk(ToolOutputAvailableChunk{
		Type:       TypeToolOutputAvail,
		ToolCallID: toolCallID,
		Output:     output,
	})
}

func (w *Writer) WriteToolOutputError(toolCallID, errorText string) error {
	return w.WriteChunk(ToolOutputErrorChunk{
		Type:       TypeToolOutputErr,
		ToolCallID: toolCallID,
		ErrorText:  errorText,
	})
}

func (w *Writer) WriteStartStep() error {
	return w.WriteChunk(UIMessageChunk{Type: TypeStartStep})
}

func (w *Writer) WriteFinishStep() error {
	return w.WriteChunk(UIMessageChunk{Type: TypeFinishStep})
}

func (w *Writer) WriteFinish(finishReason string, messageMetadata any) error {
	return w.WriteChunk(FinishChunk{
		Type:           TypeFinish,
		FinishReason:   finishReason,
		MessageMetadata: messageMetadata,
	})
}

func (w *Writer) WriteError(errorText string) error {
	return w.WriteChunk(ErrorChunk{
		Type:      TypeError,
		ErrorText: errorText,
	})
}
