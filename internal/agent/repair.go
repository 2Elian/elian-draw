package agent

import (
	"strings"
)

// RepairJSON attempts to fix common JSON issues from LLM output truncation
// Ported from route.ts:508-574
func RepairJSON(input string) string {
	s := input

	// Fix := instead of : (LLM sometimes generates this)
	s = strings.ReplaceAll(s, ":=", ": ")

	// Fix = " instead of : " (LLM sometimes generates this)
	s = replaceAllLiteral(s, `= "`, `: "`)

	// Fix inconsistent quote escaping in XML attributes within JSON strings
	// Pattern: attribute="value\" where opening quote is unescaped but closing is escaped
	// e.g., y="-20\" should be y=\"-20\"
	s = fixAttrEscaping(s)

	// Try basic bracket/brace matching
	s = fixBrackets(s)

	return s
}

func replaceAllLiteral(s, old, new string) string {
	return strings.ReplaceAll(s, old, new)
}

func fixAttrEscaping(s string) string {
	// Simple fix: replace pattern attribute="value\" with attribute=\"value\"
	// This is a heuristic - not perfect but covers common cases
	result := s
	// Match word="...\" pattern and fix it to word=\"...\"
	result = strings.ReplaceAll(result, `=\"`, `=ESCAPED_QUOTE`)
	result = strings.ReplaceAll(result, `"`, `\"`)
	result = strings.ReplaceAll(result, `=ESCAPED_QUOTE`, `=\"`)
	return result
}

func fixBrackets(s string) string {
	// Count unbalanced brackets
	openBraces := strings.Count(s, "{") - strings.Count(s, "\\{")
	closeBraces := strings.Count(s, "}") - strings.Count(s, "\\}")
	openBrackets := strings.Count(s, "[") - strings.Count(s, "\\[")
	closeBrackets := strings.Count(s, "]") - strings.Count(s, "\\]")

	// Add missing closing brackets
	for i := 0; i < openBraces-closeBraces; i++ {
		s += "}"
	}
	for i := 0; i < openBrackets-closeBrackets; i++ {
		s += "]"
	}

	return s
}
