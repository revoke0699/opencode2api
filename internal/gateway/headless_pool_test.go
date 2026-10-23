package gateway

import (
	"testing"
	"time"

	"opencode2api/internal/config"
)

func TestHeadlessPoolSkipsCoolingIdentity(t *testing.T) {
	pool := newHeadlessPool()
	instances := []config.HeadlessInstance{
		{BaseURL: "http://127.0.0.1:4097"},
		{BaseURL: "http://127.0.0.1:4098"},
		{BaseURL: "http://127.0.0.1:4099"},
	}
	now := time.Now()
	pool.succeed(instances[2].BaseURL)
	first := pool.order(instances, now)
	if len(first) != 3 || first[0].BaseURL != instances[2].BaseURL {
		t.Fatalf("sticky order = %+v", first)
	}
	pool.cool(instances[2].BaseURL, now.Add(time.Minute))
	second := pool.order(instances, now)
	if len(second) != 2 || second[0].BaseURL == instances[2].BaseURL {
		t.Fatalf("cooled order = %+v", second)
	}
	if got := pool.order(instances, now.Add(2*time.Minute)); len(got) != 3 || got[0].BaseURL != instances[2].BaseURL {
		t.Fatalf("recovered order = %+v", got)
	}
}
