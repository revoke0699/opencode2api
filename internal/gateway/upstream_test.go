package gateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"opencode2api/internal/identity"
	wire "opencode2api/internal/protocol"
)

// The shapes below are the fields an OpenCode 1.19.4 agent request carries on
// the v1 lane: the client's key order, max_tokens 32000 and the streaming
// option that keeps usage reporting alive. A free-tier request from a plain
// OpenAI client must come out looking like that, not like an alphabetically
// ordered Go map.

const testReadTool = `{"type":"function","function":{"name":"read","description":"Read a file.","parameters":{"type":"object","properties":{}}}}`

func topLevelKeys(t *testing.T, body []byte) []string {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(body))
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		t.Fatalf("body is not a JSON object: %s", body)
	}
	var keys []string
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, token.(string))
		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			t.Fatal(err)
		}
	}
	return keys
}

func decodeBody(t *testing.T, body []byte) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return payload
}

func TestPrepareAnonymousBodyV1ChatUsesClientOrderAndLimit(t *testing.T) {
	body := []byte(`{"model":"m","messages":[{"role":"user","content":"hi"}],"tools":[` + testReadTool + `],"stream":false,"topP":0.9}`)

	shaped, alias := prepareAnonymousBody(body, wire.Chat, true)
	if alias != "" {
		t.Fatalf("alias = %q, want empty", alias)
	}
	wantKeys := []string{"model", "max_tokens", "topP", "messages", "tools", "tool_choice", "stream", "stream_options"}
	if got := topLevelKeys(t, shaped); !reflect.DeepEqual(got, wantKeys) {
		t.Fatalf("key order = %v, want %v\nbody: %s", got, wantKeys, shaped)
	}
	payload := decodeBody(t, shaped)
	if payload["max_tokens"] != float64(clientCompletionLimit) {
		t.Errorf("max_tokens = %v, want %d", payload["max_tokens"], clientCompletionLimit)
	}
	if payload["stream"] != true {
		t.Errorf("stream = %v, want true", payload["stream"])
	}
}

func TestPrepareAnonymousBodyV1ChatKeepsExplicitLimit(t *testing.T) {
	body := []byte(`{"model":"m","max_tokens":128,"messages":[{"role":"user","content":"hi"}],"tools":[` + testReadTool + `],"stream":true}`)

	shaped, _ := prepareAnonymousBody(body, wire.Chat, true)
	if got := topLevelKeys(t, shaped); got[0] != "model" || got[1] != "max_tokens" {
		t.Fatalf("key order = %v, want model first and max_tokens second", got)
	}
	if limit := decodeBody(t, shaped)["max_tokens"]; limit != float64(128) {
		t.Fatalf("max_tokens = %v, want the caller's 128", limit)
	}
}

func TestPrepareAnonymousBodyV1ResponsesAddsOutputLimit(t *testing.T) {
	body := []byte(`{"model":"m","input":"hi","stream":false}`)

	shaped, _ := prepareAnonymousBody(body, wire.Responses, true)
	wantKeys := []string{"model", "input", "max_output_tokens", "stream", "tools"}
	if got := topLevelKeys(t, shaped); !reflect.DeepEqual(got, wantKeys) {
		t.Fatalf("key order = %v, want %v\nbody: %s", got, wantKeys, shaped)
	}
	if limit := decodeBody(t, shaped)["max_output_tokens"]; limit != float64(clientCompletionLimit) {
		t.Fatalf("max_output_tokens = %v, want %d", limit, clientCompletionLimit)
	}
}

func TestSetV1HeadersUsesChildSessionForSubagent(t *testing.T) {
	root := "ses_e978f8caefferuKJhUSXgHWQlq"
	child := "ses_e978f8757ffe1QzSFdmT8X4oco"
	ids := identity.RequestIDs{
		Session:       child,
		ChildSession:  child,
		ParentSession: root,
		Message:       "msg_1687078b00010k93d7DIWPx7Hb",
		Project:       "global",
	}
	header := http.Header{}
	setV1Headers(header, wire.Chat, nil, ids)

	if got := header.Get("x-opencode-session"); got != child {
		t.Errorf("x-opencode-session = %q, want child %q", got, child)
	}
	if got := header.Get("x-opencode-session-id"); got != child {
		t.Errorf("x-opencode-session-id = %q, want child %q", got, child)
	}
	if got := header.Get("x-opencode-parent-session-id"); got != root {
		t.Errorf("x-opencode-parent-session-id = %q, want %q", got, root)
	}
	if got := header.Get("x-parent-session-id"); got != root {
		t.Errorf("x-parent-session-id = %q, want %q", got, root)
	}
}

func TestSetV1HeadersRootSessionHasNoParent(t *testing.T) {
	root := "ses_e978f8caefferuKJhUSXgHWQlq"
	ids := identity.RequestIDs{Session: root, ChildSession: root, Message: "msg_1686d1e8a001gpUXJ7RIkcoGGD", Project: "global"}
	header := http.Header{}
	setV1Headers(header, wire.Chat, nil, ids)

	if got := header.Get("x-opencode-session"); got != root {
		t.Errorf("x-opencode-session = %q, want %q", got, root)
	}
	if got := header.Get("x-opencode-session-id"); got != root {
		t.Errorf("x-opencode-session-id = %q, want %q", got, root)
	}
	if got := header.Get("x-opencode-parent-session-id"); got != "" {
		t.Errorf("x-opencode-parent-session-id = %q, want empty", got)
	}
}
