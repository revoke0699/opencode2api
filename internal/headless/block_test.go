package headless

import (
	"errors"
	"testing"
)

func TestIdentityBlocked(t *testing.T) {
	blocked := errors.New("opencode serve /session/x/message: HTTP 403: abusive traffic from this IP address")
	if !IdentityBlocked(blocked) {
		t.Fatal("abuse body should rotate")
	}
	quota := errors.New("Free usage limit reached. Subscribe to OpenCode Go")
	if !IdentityBlocked(quota) {
		t.Fatal("quota body should rotate")
	}
	shape := errors.New("HTTP 400: context_length_exceeded")
	if IdentityBlocked(shape) {
		t.Fatal("request rejection is not an identity block")
	}
}
