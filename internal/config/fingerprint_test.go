package config

import (
	"reflect"
	"testing"
)

func TestFingerprintDefaultsToFixedBun(t *testing.T) {
	var fingerprint FingerprintConfig
	if err := fingerprint.Normalize(); err != nil {
		t.Fatal(err)
	}
	if fingerprint.Mode != FingerprintFixed {
		t.Errorf("mode = %q, want %q", fingerprint.Mode, FingerprintFixed)
	}
	if fingerprint.Rotate != FingerprintRotateProxy {
		t.Errorf("rotate = %q, want %q", fingerprint.Rotate, FingerprintRotateProxy)
	}
	if want := []string{"bun"}; !reflect.DeepEqual(fingerprint.Profiles, want) {
		t.Errorf("profiles = %v, want %v", fingerprint.Profiles, want)
	}
}

func TestFingerprintRandomFillsDefaultProfiles(t *testing.T) {
	fingerprint := FingerprintConfig{Mode: "RANDOM", Rotate: "CONNECTION"}
	if err := fingerprint.Normalize(); err != nil {
		t.Fatal(err)
	}
	if fingerprint.Mode != FingerprintRandom || fingerprint.Rotate != FingerprintRotateConnection {
		t.Fatalf("normalized mode/rotate = %q/%q", fingerprint.Mode, fingerprint.Rotate)
	}
	if want := DefaultFingerprintProfiles; !reflect.DeepEqual(fingerprint.Profiles, want) {
		t.Errorf("profiles = %v, want %v", fingerprint.Profiles, want)
	}
}

func TestFingerprintRejectsUnknownValues(t *testing.T) {
	if err := (&FingerprintConfig{Mode: "sometimes"}).Normalize(); err == nil {
		t.Error("unknown mode normalized without an error")
	}
	if err := (&FingerprintConfig{Mode: FingerprintRandom, Rotate: "session"}).Normalize(); err == nil {
		t.Error("unknown rotate normalized without an error")
	}
}

func TestFingerprintFixedDiscardsProfiles(t *testing.T) {
	fingerprint := FingerprintConfig{Mode: FingerprintFixed, Profiles: []string{"chrome"}}
	if err := fingerprint.Normalize(); err != nil {
		t.Fatal(err)
	}
	if want := []string{"bun"}; !reflect.DeepEqual(fingerprint.Profiles, want) {
		t.Errorf("profiles = %v, want %v", fingerprint.Profiles, want)
	}
}
