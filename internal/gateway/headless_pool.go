package gateway

import (
	"sync"
	"time"

	"opencode2api/internal/config"
)

// headlessTransportCool is how long a serve process that could not be reached
// stays out of the cursor. A blocked identity uses the free-tier block TTL
// instead, which is longer.
const headlessTransportCool = 30 * time.Second

// headlessAttemptTimeout bounds one identity. A serve process that hangs on a
// dead egress must not consume the whole client deadline before the next
// identity is tried.
const headlessAttemptTimeout = 45 * time.Second

// headlessPool rotates official serve processes. It stays on the last
// identity that succeeded, and skips a cooled base URL until its deadline so
// a banned fingerprint is not retried on every request.
type headlessPool struct {
	mu     sync.Mutex
	next   int
	sticky string
	until  map[string]time.Time
}

func newHeadlessPool() *headlessPool {
	return &headlessPool{until: map[string]time.Time{}}
}

// order returns the instances to try for one request, starting at the
// rotating cursor and skipping identities that are still cooling.
func (p *headlessPool) order(instances []config.HeadlessInstance, now time.Time) []config.HeadlessInstance {
	if p == nil || len(instances) == 0 {
		return nil
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	start := p.next % len(instances)
	if p.sticky != "" {
		for i, inst := range instances {
			if inst.BaseURL != p.sticky {
				continue
			}
			if until, ok := p.until[inst.BaseURL]; !ok || !now.Before(until) {
				start = i
			}
			break
		}
	}
	p.next = (start + 1) % len(instances)
	out := make([]config.HeadlessInstance, 0, len(instances))
	for i := 0; i < len(instances); i++ {
		inst := instances[(start+i)%len(instances)]
		if until, ok := p.until[inst.BaseURL]; ok && now.Before(until) {
			continue
		}
		out = append(out, inst)
	}
	return out
}

func (p *headlessPool) succeed(baseURL string) {
	if p == nil || baseURL == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.sticky = baseURL
	delete(p.until, baseURL)
}

func (p *headlessPool) cool(baseURL string, until time.Time) {
	if p == nil || baseURL == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.until == nil {
		p.until = map[string]time.Time{}
	}
	p.until[baseURL] = until
}
