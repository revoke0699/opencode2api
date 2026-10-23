package httpx

import "testing"

// These literals are the User-Agent lines captured from an official OpenCode
// 1.19.4 request. They pin the claimed client and runtime so a version bump
// updates them together.
func TestClientUserAgentsMatchCapturedRelease(t *testing.T) {
	if got, want := V1ModelUserAgent(V1ChatSDK), "opencode/1.19.4 ai-sdk/provider-utils/4.0.23 runtime/bun/1.4.4"; got != want {
		t.Errorf("chat User-Agent = %q, want %q", got, want)
	}
	if got, want := V1ModelUserAgent(V1ResponsesSDK), "opencode/1.19.4 ai-sdk/provider-utils/4.0.40 runtime/bun/1.4.4"; got != want {
		t.Errorf("responses User-Agent = %q, want %q", got, want)
	}
	if got, want := V1ModelUserAgent(V1AnthropicSDK), "opencode/1.19.4 ai-sdk/provider-utils/4.0.46 runtime/bun/1.4.4"; got != want {
		t.Errorf("anthropic User-Agent = %q, want %q", got, want)
	}
	if got, want := V2UserAgent(), "opencode/latest/2.1.1/cli"; got != want {
		t.Errorf("v2 User-Agent = %q, want %q", got, want)
	}
}
