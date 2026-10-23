package headless

import (
	"encoding/json"
	"strings"
)

// ConversationText renders a gateway request body as the text parts opencode
// serve accepts. It understands Chat messages, Responses input, and Anthropic
// message blocks. System text is included ahead of the turns.
func ConversationText(body []byte) string {
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	var lines []string
	if system := textOf(payload["system"]); system != "" {
		lines = append(lines, system)
	}
	if instructions := textOf(payload["instructions"]); instructions != "" {
		lines = append(lines, instructions)
	}
	if messages, ok := payload["messages"].([]any); ok {
		for _, raw := range messages {
			if line := messageLine(raw); line != "" {
				lines = append(lines, line)
			}
		}
	}
	switch input := payload["input"].(type) {
	case string:
		if strings.TrimSpace(input) != "" {
			lines = append(lines, input)
		}
	case []any:
		for _, raw := range input {
			if line := messageLine(raw); line != "" {
				lines = append(lines, line)
			}
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func messageLine(raw any) string {
	item, ok := raw.(map[string]any)
	if !ok {
		return textOf(raw)
	}
	role, _ := item["role"].(string)
	content := textOf(item["content"])
	if content == "" {
		content = textOf(item["text"])
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	if role == "" {
		return content
	}
	return role + ": " + content
}

func textOf(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case []any:
		var parts []string
		for _, raw := range typed {
			switch part := raw.(type) {
			case string:
				if strings.TrimSpace(part) != "" {
					parts = append(parts, part)
				}
			case map[string]any:
				if text, _ := part["text"].(string); strings.TrimSpace(text) != "" {
					parts = append(parts, text)
					continue
				}
				if text := textOf(part["content"]); text != "" {
					parts = append(parts, text)
				}
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	default:
		return ""
	}
}
