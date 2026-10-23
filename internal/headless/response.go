package headless

import (
	"encoding/json"
	"strings"
	"time"
)

// ChatJSON is one non-streaming Chat Completions document.
func ChatJSON(result Result) []byte {
	raw, _ := json.Marshal(map[string]any{
		"id":      result.ID,
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   result.Model,
		"choices": []any{map[string]any{
			"index":         0,
			"finish_reason": result.Finish,
			"message": map[string]any{
				"role":    "assistant",
				"content": result.Text,
			},
		}},
		"usage": usage(result),
	})
	return raw
}

// ChatSSE is a Chat Completions stream: one content chunk, a finish chunk
// carrying usage, and the [DONE] terminator the gateway's chat parser expects.
func ChatSSE(result Result) []byte {
	var b strings.Builder
	write := func(payload map[string]any) {
		raw, _ := json.Marshal(payload)
		b.WriteString("data: ")
		b.Write(raw)
		b.WriteString("\n\n")
	}
	write(map[string]any{
		"id": result.ID, "object": "chat.completion.chunk", "model": result.Model,
		"choices": []any{map[string]any{
			"index": 0,
			"delta": map[string]any{"role": "assistant", "content": result.Text},
		}},
	})
	write(map[string]any{
		"id": result.ID, "object": "chat.completion.chunk", "model": result.Model,
		"choices": []any{map[string]any{"index": 0, "delta": map[string]any{}, "finish_reason": result.Finish}},
		"usage":   usage(result),
	})
	b.WriteString("data: [DONE]\n\n")
	return []byte(b.String())
}

func usage(result Result) map[string]int {
	return map[string]int{
		"prompt_tokens":     result.PromptTokens,
		"completion_tokens": result.OutputTokens,
		"total_tokens":      result.TotalTokens,
	}
}
