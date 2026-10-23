package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHeadlessBaseURLDefaultsWhenEnabled(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	raw := []byte(`{
		"server_keys": ["local-key"],
		"anonymous": true,
		"headless": {"enabled": true, "base_url": ""}
	}`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Headless.Enabled || cfg.Headless.BaseURL != "http://127.0.0.1:4097" {
		t.Fatalf("headless = %+v", cfg.Headless)
	}
}

func TestHeadlessEndpointsPreferInstances(t *testing.T) {
	cfg := HeadlessConfig{
		BaseURL:  "http://127.0.0.1:4097",
		Password: "shared",
		Instances: []HeadlessInstance{
			{BaseURL: "http://127.0.0.1:4097"},
			{BaseURL: "http://127.0.0.1:4098"},
		},
	}
	got := cfg.Endpoints()
	if len(got) != 2 || got[1].BaseURL != "http://127.0.0.1:4098" {
		t.Fatalf("endpoints = %+v", got)
	}
	single := HeadlessConfig{BaseURL: "http://127.0.0.1:4097", Password: "shared"}.Endpoints()
	if len(single) != 1 || single[0].Password != "shared" {
		t.Fatalf("single = %+v", single)
	}
}

func TestHeadlessBaseURLRejectsNonHTTP(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	raw := []byte(`{
		"server_keys": ["local-key"],
		"anonymous": true,
		"headless": {"enabled": true, "base_url": "file:///tmp/serve"}
	}`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected invalid base_url to fail")
	}
}
