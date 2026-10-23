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
	first := pool.order(instances, now)
	if len(first) != 3 || first[0].BaseURL != instances[0].BaseURL {
		t.Fatalf("first order = %+v", first)
	}
	pool.cool(instances[0].BaseURL, now.Add(time.Minute))
	second := pool.order(instances, now)
	if len(second) != 2 || second[0].BaseURL != instances[1].BaseURL {
		t.Fatalf("second order = %+v", second)
	}
	third := pool.order(instances, now)
	if len(third) != 2 || third[0].BaseURL != instances[2].BaseURL {
		t.Fatalf("third order = %+v", third)
	}
	if got := pool.order(instances, now.Add(2*time.Minute)); len(got) != 3 {
		t.Fatalf("recovered order = %+v", got)
	}
}
