// Package headless talks to a local official `opencode serve` process.
//
// The free tier rejects forged clients. A headless official server still
// accepts the same models, so the gateway creates one session per request,
// sends the conversation as text parts, and turns the assistant text back
// into a Chat Completions body the rest of the gateway already understands.
package headless

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Result is the assistant turn returned by opencode serve.
type Result struct {
	ID           string
	Model        string
	Text         string
	Finish       string
	PromptTokens int
	OutputTokens int
	TotalTokens  int
}

// Client calls one opencode serve base URL. Password, when set, is sent as
// HTTP basic auth with the username "opencode", which is what serve expects
// from OPENCODE_SERVER_PASSWORD.
type Client struct {
	BaseURL  string
	Password string
	HTTP     *http.Client
}

// Complete opens a session and waits for the model to finish. provider is the
// OpenCode provider id ("opencode" for the free catalog) and model is the
// model id without that prefix.
func (c Client) Complete(ctx context.Context, provider, model, prompt string) (Result, error) {
	if strings.TrimSpace(prompt) == "" {
		return Result{}, fmt.Errorf("headless prompt is empty")
	}
	if provider == "" {
		provider = "opencode"
	}
	session := struct {
		ID string `json:"id"`
	}{}
	if err := c.post(ctx, "/session", map[string]any{}, &session); err != nil {
		return Result{}, err
	}
	if session.ID == "" {
		return Result{}, fmt.Errorf("opencode serve did not return a session id")
	}
	defer c.deleteSession(ctx, session.ID)
	var message serveMessage
	err := c.post(ctx, "/session/"+session.ID+"/message", map[string]any{
		"model": map[string]string{"providerID": provider, "modelID": model},
		"parts": []map[string]string{{"type": "text", "text": prompt}},
	}, &message)
	if err != nil {
		return Result{}, err
	}
	text := message.text()
	if text == "" && serveError(message.Info.Error) != "" {
		return Result{}, fmt.Errorf("opencode serve: %s", serveError(message.Info.Error))
	}
	finish := message.Info.Finish
	if finish == "" {
		finish = "stop"
	}
	total := message.Info.Tokens.Total
	if total == 0 {
		total = message.Info.Tokens.Input + message.Info.Tokens.Output
	}
	return Result{
		ID:           message.Info.ID,
		Model:        provider + "/" + model,
		Text:         text,
		Finish:       finish,
		PromptTokens: message.Info.Tokens.Input,
		OutputTokens: message.Info.Tokens.Output,
		TotalTokens:  total,
	}, nil
}

// ModelRef splits a gateway model id into the OpenCode provider and model.
// Clients sometimes send "opencode/glm-5.3-flash-free"; serve wants those as
// two fields. Any other id is sent as the model under the opencode provider.
func ModelRef(id string) (provider, model string) {
	provider = "opencode"
	model = strings.TrimSpace(id)
	if rest, ok := strings.CutPrefix(model, "opencode/"); ok && rest != "" {
		model = rest
	}
	return provider, model
}

type serveMessage struct {
	Info struct {
		ID     string          `json:"id"`
		Finish string          `json:"finish"`
		Error  json.RawMessage `json:"error"`
		Tokens struct {
			Input  int `json:"input"`
			Output int `json:"output"`
			Total  int `json:"total"`
		} `json:"tokens"`
	} `json:"info"`
	Parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"parts"`
}

func (m serveMessage) text() string {
	var b strings.Builder
	for _, part := range m.Parts {
		if part.Type == "text" {
			b.WriteString(part.Text)
		}
	}
	return b.String()
}

func (c Client) post(ctx context.Context, path string, payload any, dest any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+path, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.Password != "" {
		token := base64.StdEncoding.EncodeToString([]byte("opencode:" + c.Password))
		req.Header.Set("Authorization", "Basic "+token)
	}
	client := c.HTTP
	if client == nil {
		// The request context carries the gateway deadline. A client-level
		// timeout would cut a long completion short of that deadline.
		client = &http.Client{}
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("opencode serve %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return fmt.Errorf("opencode serve %s: %w", path, err)
	}
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("opencode serve %s: HTTP %d: %s", path, resp.StatusCode, trimBody(body))
	}
	if dest == nil {
		return nil
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("opencode serve %s: %w", path, err)
	}
	return nil
}

func serveError(raw json.RawMessage) string {
	text := strings.TrimSpace(string(raw))
	if text == "" || text == "null" {
		return ""
	}
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return strings.TrimSpace(asString)
	}
	return trimBody(raw)
}

func (c Client) deleteSession(ctx context.Context, id string) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, strings.TrimRight(c.BaseURL, "/")+"/session/"+id, nil)
	if err != nil {
		return
	}
	if c.Password != "" {
		token := base64.StdEncoding.EncodeToString([]byte("opencode:" + c.Password))
		req.Header.Set("Authorization", "Basic "+token)
	}
	client := c.HTTP
	if client == nil {
		client = &http.Client{}
	}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	_ = resp.Body.Close()
}

func trimBody(body []byte) string {
	text := strings.TrimSpace(string(body))
	if len(text) > 300 {
		return text[:300]
	}
	return text
}
