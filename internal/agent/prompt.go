package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/2Elian/next-ai-draw-io/go-backend/internal/config"
)

var (
	defaultPrompt     string
	extendedPrompt    string
	styleInstructions string    // 风格指令，例如“用友好语气回复”、“输出JSON格式”等
	minimalStyleInstr string    // 精简风格指令，将去除掉所有颜色和样式只保留黑白
	promptsOnce       sync.Once // 确保 loadPrompts 中的初始化逻辑只执行一次，无论被多少个 goroutine 并发调用
)

func loadPrompts() {
	promptsOnce.Do(func() {
		// Try to load from prompts/ directory, fall back to embedded defaults
		// 尝试从 prompts/ 目录加载，否则使用下面传入的默认字符串
		// TODO: 将prompt写入txt里面
		defaultPrompt = loadPromptFile("prompts/system_default.txt", defaultSystemPrompt)
		extendedPrompt = loadPromptFile("prompts/system_extended.txt", defaultSystemPrompt+extendedAdditions)
		styleInstructions = loadPromptFile("prompts/style.txt", styleInstructionsDefault)
		minimalStyleInstr = loadPromptFile("prompts/minimal_style.txt", minimalStyleInstructionDefault)
	})
}

func loadPromptFile(path, fallback string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return fallback
	}
	return string(data)
}

// GetSystemPrompt returns the appropriate system prompt for the given model
func GetSystemPrompt(modelID string, minimalStyle bool) string {
	loadPrompts()

	modelName := modelID
	if modelName == "" {
		modelName = "AI"
	}

	var prompt string
	// Use extended prompt for Opus/Haiku 4.5
	// TODO why need to use extendedPrompt for Opus/Haiku 4.5?
	if modelID != "" && (strings.Contains(modelID, "claude-opus-4-5") || strings.Contains(modelID, "claude-haiku-4-5")) {
		prompt = extendedPrompt
	} else {
		prompt = defaultPrompt
	}

	if minimalStyle {
		prompt = minimalStyleInstr + prompt
	} else {
		prompt += styleInstructions
	}

	return strings.ReplaceAll(prompt, "{{MODEL_NAME}}", modelName) // 将变量 prompt（一个字符串）中 所有出现 的占位符 "{{MODEL_NAME}}" 替换为变量 modelName 的值，并返回替换后的新字符串。
}

// BuildXMLContext builds the XML context string from current and previous diagram XML
func BuildXMLContext(xml, previousXML string) string {
	var sb strings.Builder
	// TODO: 工具以及这些xml都写成: <xml> </xml>的xml形式
	if previousXML != "" {
		sb.WriteString("Previous diagram XML (before user's last message):\n")
		sb.WriteString("\"\"\"xml\n")
		sb.WriteString(previousXML)
		sb.WriteString("\n\"\"\"\n\n")
	}

	sb.WriteString("Current diagram XML (AUTHORITATIVE - the source of truth):\n")
	sb.WriteString("\"\"\"xml\n")
	sb.WriteString(xml)
	sb.WriteString("\n\"\"\"\n\n")
	sb.WriteString("IMPORTANT: The \"Current diagram XML\" is the SINGLE SOURCE OF TRUTH for what's on the canvas right now. The user can manually add, delete, or modify shapes directly in draw.io. Always count and describe elements based on the CURRENT XML, not on what you previously generated. If both previous and current XML are shown, compare them to understand what the user changed. When using edit_diagram, COPY search patterns exactly from the CURRENT XML - attribute order matters!")

	return sb.String()
}

// BuildSystemMessages constructs the system messages for the AI model
func BuildSystemMessages(systemPrompt, xmlContext string, provider config.ProviderName, shouldCache bool) []Message {
	isSingleSystem := config.SingleSystemProviders[provider]

	if isSingleSystem {
		return []Message{
			{
				Role: "system",
				Content: []ContentPart{
					{Type: "text", Text: systemPrompt + "\n\n" + xmlContext},
				},
			},
		}
	}

	return []Message{
		{
			Role: "system",
			Content: []ContentPart{
				{Type: "text", Text: systemPrompt},
			},
		},
		{
			Role: "system",
			Content: []ContentPart{
				{Type: "text", Text: xmlContext},
			},
		},
	}
}

// FormatUserInput formats the user's text input
func FormatUserInput(text string) string {
	return fmt.Sprintf("User input:\n\"\"\"md\n%s\n\"\"\"", text)
}

// IsMinimalDiagram checks if a diagram XML is essentially empty
func IsMinimalDiagram(xml string) bool {
	stripped := strings.ReplaceAll(xml, " ", "")
	stripped = strings.ReplaceAll(stripped, "\n", "")
	stripped = strings.ReplaceAll(stripped, "\t", "")
	stripped = strings.ReplaceAll(stripped, "\r", "")
	return !strings.Contains(stripped, `id="2"`)
}

// GetShapeLibrary reads a shape library file from disk
func GetShapeLibrary(library string) (string, error) {
	// Sanitize input - prevent path traversal
	sanitized := strings.ToLower(library)
	sanitized = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return -1
	}, sanitized)

	if sanitized != strings.ToLower(library) {
		return fmt.Sprintf("Invalid library name \"%s\". Use only letters, numbers, underscores, and hyphens.", library), nil
	}

	baseDir := filepath.Join(".", "docs", "shape-libraries")
	filePath := filepath.Join(baseDir, sanitized+".md")

	// Verify path stays within expected directory
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(absPath, absBase) {
		return "Invalid library path.", nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Sprintf("Library \"%s\" not found. Available: aws4, azure2, gcp2, alibaba_cloud, cisco19, kubernetes, network, bpmn, flowchart, basic, arrows2, vvd, salesforce, citrix, sap, mscae, atlassian, fluidpower, electrical, pid, cabinets, floorplan, webicons, infographic, sitemap, android, material_design, lean_mapping, openstack, rack", library), nil
		}
		return "", fmt.Errorf("read library file: %w", err)
	}

	return string(data), nil
}

// Embedded prompt defaults (used when prompt files aren't available)

const defaultSystemPrompt = `
You are an expert diagram creation assistant specializing in draw.io XML generation.
Your primary function is chat with user and crafting clear, well-organized visual diagrams through precise XML specifications.
You can see images that users upload, and you can read the text content extracted from PDF documents they upload.
ALWAYS respond in the same language as the user's last message.

When you are asked to create a diagram, briefly describe your plan about the layout and structure to avoid object overlapping or edge cross the objects. (2-3 sentences max), then use display_diagram tool to generate the XML.
After generating or editing a diagram, you don't need to say anything. The user can see the diagram - no need to describe it.

## App Context
You are an AI agent (powered by {{MODEL_NAME}}) inside a web app. The interface has:
- **Left panel**: Draw.io diagram editor where diagrams are rendered
- **Right panel**: Chat interface where you communicate with the user

You can read and modify diagrams by generating draw.io XML code through tool calls.

You utilize the following tools:
---Tool1---
tool name: display_diagram
description: Display a NEW diagram on draw.io. Use this when creating a diagram from scratch or when major structural changes are needed.
---Tool2---
tool name: edit_diagram
description: Edit specific parts of the EXISTING diagram. Use this when making small targeted changes.
---Tool3---
tool name: append_diagram
description: Continue generating diagram XML when display_diagram was truncated.
---Tool4---
tool name: get_shape_library
description: Get shape/icon library documentation.
---End of tools---

Core capabilities:
- Generate valid, well-formed XML strings for draw.io diagrams
- Create professional flowcharts, mind maps, entity diagrams, and technical illustrations
- Convert user descriptions into visually appealing diagrams using basic shapes and connectors
- Apply proper spacing, alignment and visual hierarchy in diagram layouts
- Optimize element positioning to prevent overlapping and maintain readability

Layout constraints:
- CRITICAL: Keep all diagram elements within a single page viewport to avoid page breaks
- Position all elements with x coordinates between 0-800 and y coordinates between 0-600
- Maximum width for containers: 700 pixels
- Maximum height for containers: 550 pixels

Note that:
- Use proper tool calls to generate or edit diagrams;
  - never return raw XML in text responses
- Return XML only via tool calls, never in text responses.
- For cloud/tech diagrams, call get_shape_library first to discover available icon shapes.
- NEVER include XML comments in your generated XML.
`

const extendedAdditions = `
## Extended Tool Reference

### display_diagram Details
VALIDATION RULES:
1. Generate ONLY mxCell elements - wrapper tags and root cells are added automatically
2. All mxCell elements must be siblings - never nested
3. Every mxCell needs a unique id attribute (start from "2")
4. Every mxCell needs a valid parent attribute

### edit_diagram Details
edit_diagram uses ID-based operations to modify cells directly by their id attribute.
Operations: update (modify cell by id), add (new cell), delete (remove cell by id)
Cascade is automatic: children AND edges are auto-deleted.

### Edge Routing Rules
Always specify exitX, exitY, entryX, entryY explicitly in edge styles.
Route edges AROUND intermediate shapes using waypoints.
`

const styleInstructionsDefault = `
Common styles:
- Shapes: rounded=1 (rounded corners), fillColor=#hex, strokeColor=#hex
- Edges: endArrow=classic/block/open/none, startArrow=none/classic, curved=1, edgeStyle=orthogonalEdgeStyle
- Text: fontSize=14, fontStyle=1 (bold), align=center/left/right
`

const minimalStyleInstructionDefault = `
## MINIMAL STYLE MODE ACTIVE
No Styling - Plain Black/White Only
- NO fillColor, NO strokeColor, NO rounded, NO fontSize, NO fontStyle
- Style: "whiteSpace=wrap;html=1;" for shapes, "html=1;endArrow=classic;" for edges
Focus on Layout Quality - follow Edge Routing Rules strictly.
`
