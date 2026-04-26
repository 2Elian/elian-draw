package agent

// Tool definitions matching the original route.ts tool schemas

import "github.com/2Elian/next-ai-draw-io/go-backend/internal/provider"

// Tool name constants
const (
	ToolDisplayDiagram  = "display_diagram"
	ToolEditDiagram     = "edit_diagram"
	ToolAppendDiagram   = "append_diagram"
	ToolGetShapeLibrary = "get_shape_library"
)

// ClientSideTools are tools that execute on the client (no server-side execution)
var ClientSideTools = map[string]bool{
	ToolDisplayDiagram: true,
	ToolEditDiagram:    true,
	ToolAppendDiagram:  true,
}

// GetTools returns all tool definitions for the AI model
func GetTools() []provider.ToolDef {
	return []provider.ToolDef{
		{
			Name:        ToolDisplayDiagram,
			Description: displayDiagramDesc,
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"xml": map[string]any{
						"type":        "string",
						"description": "XML string to be displayed on draw.io",
					},
				},
				"required": []string{"xml"},
			},
		},
		{
			Name:        ToolEditDiagram,
			Description: editDiagramDesc,
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"operations": map[string]any{
						"type":        "array",
						"description": "Array of operations to apply",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"operation": map[string]any{
									"type":        "string",
									"enum":        []string{"update", "add", "delete"},
									"description": "Operation to perform: add, update, or delete",
								},
								"cell_id": map[string]any{
									"type":        "string",
									"description": "The id of the mxCell. Must match the id attribute in new_xml.",
								},
								"new_xml": map[string]any{
									"type":        "string",
									"description": "Complete mxCell XML element (required for update/add)",
								},
							},
							"required": []string{"operation", "cell_id"},
						},
					},
				},
				"required": []string{"operations"},
			},
		},
		{
			Name:        ToolAppendDiagram,
			Description: appendDiagramDesc,
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"xml": map[string]any{
						"type":        "string",
						"description": "Continuation XML fragment to append (NO wrapper tags)",
					},
				},
				"required": []string{"xml"},
			},
		},
		{
			Name:        ToolGetShapeLibrary,
			Description: getShapeLibraryDesc,
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"library": map[string]any{
						"type":        "string",
						"description": "Library name (e.g., 'aws4', 'kubernetes', 'flowchart')",
					},
				},
				"required": []string{"library"},
			},
		},
	}
}

const displayDiagramDesc = `Display a diagram on draw.io. Pass ONLY the mxCell elements - wrapper tags and root cells are added automatically.

VALIDATION RULES (XML will be rejected if violated):
1. Generate ONLY mxCell elements - NO wrapper tags (<mxfile>, <mxGraphModel>, <root>)
2. Do NOT include root cells (id="0" or id="1") - they are added automatically
3. All mxCell elements must be siblings - never nested
4. Every mxCell needs a unique id (start from "2")
5. Every mxCell needs a valid parent attribute (use "1" for top-level)
6. Escape special chars in values: &lt; &gt; &amp; &quot;

Example (generate ONLY this - no wrapper tags):
<mxCell id="lane1" value="Frontend" style="swimlane;" vertex="1" parent="1">
  <mxGeometry x="40" y="40" width="200" height="200" as="geometry"/>
</mxCell>
<mxCell id="step1" value="Step 1" style="rounded=1;" vertex="1" parent="lane1">
  <mxGeometry x="20" y="60" width="160" height="40" as="geometry"/>
</mxCell>

Notes:
- For AWS diagrams, use **AWS 2025 icons**.
- For animated connectors, add "flowAnimation=1" to edge style.`

const editDiagramDesc = `Edit the current diagram by ID-based operations (update/add/delete cells).

Operations:
- update: Replace an existing cell by its id. Provide cell_id and complete new_xml.
- add: Add a new cell. Provide cell_id (new unique id) and new_xml.
- delete: Remove a cell. Cascade is automatic: children AND edges (source/target) are auto-deleted. Only specify ONE cell_id.

For update/add, new_xml must be a complete mxCell element including mxGeometry.

⚠️ JSON ESCAPING: Every " inside new_xml MUST be escaped as \\". Example: id=\\"5\\" value=\\"Label\\"

Example - Add a rectangle:
{"operations": [{"operation": "add", "cell_id": "rect-1", "new_xml": "<mxCell id=\\"rect-1\\" value=\\"Hello\\" style=\\"rounded=0;\\" vertex=\\"1\\" parent=\\"1\\"><mxGeometry x=\\"100\\" y=\\"100\\" width=\\"120\\" height=\\"60\\" as=\\"geometry\\"/></mxCell>"}]}

Example - Delete container (children & edges auto-deleted):
{"operations": [{"operation": "delete", "cell_id": "2"}]}`

const appendDiagramDesc = `Continue generating diagram XML when previous display_diagram output was truncated due to length limits.

WHEN TO USE: Only call this tool after display_diagram was truncated (you'll see an error message about truncation).

CRITICAL INSTRUCTIONS:
1. Do NOT include any wrapper tags - just continue the mxCell elements
2. Continue from EXACTLY where your previous output stopped
3. Complete the remaining mxCell elements
4. If still truncated, call append_diagram again with the next fragment

Example: If previous output ended with '<mxCell id="x" style="rounded=1', continue with ';" vertex="1">...' and complete the remaining elements.`

const getShapeLibraryDesc = `Get draw.io shape/icon library documentation with style syntax and shape names.

Available libraries:
- Cloud: aws4, azure2, gcp2, alibaba_cloud, openstack, salesforce
- Networking: cisco19, network, kubernetes, vvd, rack
- Business: bpmn, lean_mapping
- General: flowchart, basic, arrows2, infographic, sitemap
- UI/Mockups: android, material_design
- Enterprise: citrix, sap, mscae, atlassian
- Engineering: fluidpower, electrical, pid, cabinets, floorplan
- Icons: webicons

Call this tool to get shape names and usage syntax for a specific library.`
