package identity

import (
	"net/http/httptest"
	"testing"
)

// An official 1.x subagent sends the child session in both session headers and
// the parent in x-opencode-parent-session-id. Both ids must survive derivation
// so the v1 header writer can reproduce that layout.
func TestDeriveRequestIDsKeepsSubagentSessions(t *testing.T) {
	root := "ses_e978f8caefferuKJhUSXgHWQlq"
	child := "ses_e978f8757ffe1QzSFdmT8X4oco"
	request := httptest.NewRequest("POST", "/v1/chat/completions", nil)
	request.Header.Set("x-opencode-session", child)
	request.Header.Set("x-opencode-session-id", child)
	request.Header.Set("x-opencode-parent-session-id", root)

	ids := DeriveRequestIDs(request, map[string]any{}, "scope")
	if ids.Session != child {
		t.Errorf("Session = %q, want child %q", ids.Session, child)
	}
	if ids.ChildSession != child {
		t.Errorf("ChildSession = %q, want child %q", ids.ChildSession, child)
	}
	if ids.ParentSession != root {
		t.Errorf("ParentSession = %q, want parent %q", ids.ParentSession, root)
	}
}
