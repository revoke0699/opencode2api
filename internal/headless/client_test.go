package headless

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCompleteReadsAssistantText(t *testing.T) {
	var sawAuth string
	var deleted bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		if r.Method == http.MethodDelete && r.URL.Path == "/session/ses_test" {
			deleted = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		switch {
		case r.URL.Path == "/session" && r.Method == http.MethodPost:
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"id":"ses_test"}`)
		case strings.HasPrefix(r.URL.Path, "/session/ses_test/message"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"info":{"id":"msg_test","finish":"stop","tokens":{"input":3,"output":1,"total":4}},"parts":[{"type":"text","text":"OK"}]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	result, err := (Client{BaseURL: server.URL, Password: "secret"}).Complete(context.Background(), "opencode", "glm-5.3-flash-free", "reply OK")
	if err != nil {
		t.Fatal(err)
	}
	if result.Text != "OK" || result.ID != "msg_test" || result.TotalTokens != 4 {
		t.Fatalf("result = %+v", result)
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("opencode:secret"))
	if sawAuth != want {
		t.Fatalf("authorization = %q", sawAuth)
	}
	if !deleted {
		t.Fatal("session was not deleted")
	}
	provider, model := ModelRef("opencode/glm-5.3-flash-free")
	if provider != "opencode" || model != "glm-5.3-flash-free" {
		t.Fatalf("model ref = %s %s", provider, model)
	}
	if !strings.Contains(string(ChatJSON(result)), `"content":"OK"`) {
		t.Fatal("json missing content")
	}
	sse := string(ChatSSE(result))
	if !strings.Contains(sse, `"content":"OK"`) || !strings.HasSuffix(strings.TrimSpace(sse), "data: [DONE]") {
		t.Fatalf("sse = %s", sse)
	}
}

func TestConversationText(t *testing.T) {
	body := []byte(`{"system":"be brief","messages":[{"role":"user","content":[{"type":"text","text":"reply OK"}]}]}`)
	got := ConversationText(body)
	if !strings.Contains(got, "be brief") || !strings.Contains(got, "user: reply OK") {
		t.Fatalf("text = %q", got)
	}
}
