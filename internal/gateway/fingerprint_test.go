package gateway

import (
	"testing"
	"time"

	"opencode2api/internal/config"
)

func fingerprintForTest(mode, rotate string) config.FingerprintConfig {
	fingerprint := config.FingerprintConfig{
		Mode:     mode,
		Rotate:   rotate,
		Profiles: []string{"bun", "chrome", "firefox"},
	}
	if err := fingerprint.Normalize(); err != nil {
		panic(err)
	}
	return fingerprint
}

// Proxy rotation binds one profile per exit, so a session keeps its
// fingerprint until the gateway moves it to another proxy.
func TestTransportPoolBindsProfilePerProxy(t *testing.T) {
	pool, err := transportPool(
		[]string{"direct", "http://127.0.0.1:1", "socks5://127.0.0.1:2"},
		config.PerformanceConfig{},
		fingerprintForTest(config.FingerprintRandom, config.FingerprintRotateProxy),
		time.Second,
	)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"bun", "chrome", "firefox"}
	for i, proxy := range pool.items {
		if proxy.fingerprint != want[i] {
			t.Errorf("proxy %d fingerprint = %q, want %q", i, proxy.fingerprint, want[i])
		}
	}
}

func TestTransportPoolRandomizesPerConnection(t *testing.T) {
	pool, err := transportPool(
		[]string{"direct", "http://127.0.0.1:1"},
		config.PerformanceConfig{},
		fingerprintForTest(config.FingerprintRandom, config.FingerprintRotateConnection),
		time.Second,
	)
	if err != nil {
		t.Fatal(err)
	}
	for i, proxy := range pool.items {
		if proxy.fingerprint != "random" {
			t.Errorf("proxy %d fingerprint = %q, want random", i, proxy.fingerprint)
		}
	}
}
