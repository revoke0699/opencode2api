package httpx

import (
	"net/url"
	"testing"

	utls "github.com/refraction-networking/utls"
)

func TestClientHelloProfilesOfferHTTP11Only(t *testing.T) {
	for _, profile := range []string{"bun", "chrome", "firefox", "safari", "edge", "randomized"} {
		spec, err := clientHelloFor(profile)
		if err != nil {
			t.Fatalf("%s: %v", profile, err)
		}
		var alpns []string
		for _, extension := range spec.Extensions {
			if alpn, ok := extension.(*utls.ALPNExtension); ok {
				alpns = append(alpns, alpn.AlpnProtocols...)
			}
		}
		if len(alpns) == 0 {
			t.Errorf("%s: no ALPN extension", profile)
			continue
		}
		for _, alpn := range alpns {
			if alpn != "http/1.1" {
				t.Errorf("%s: ALPN %q, want http/1.1 only", profile, alpn)
			}
		}
	}
}

func TestClientHelloRejectsUnknownProfile(t *testing.T) {
	if _, err := clientHelloFor("netscape"); err == nil {
		t.Fatal("unknown profile built a ClientHello")
	}
	if _, err := NewBunTransport(BunOptions{Profiles: []string{"netscape"}}); err == nil {
		t.Fatal("transport accepted an unknown profile")
	}
}

func TestBunTransportAcceptsProfilePool(t *testing.T) {
	proxy, _ := url.Parse("http://127.0.0.1:1")
	transport, err := NewBunTransport(BunOptions{Proxy: proxy, Profiles: []string{"bun", "chrome"}, Rotate: true})
	if err != nil {
		t.Fatal(err)
	}
	if !transport.rotate || len(transport.profiles) != 2 {
		t.Fatalf("transport profiles = %v, rotate = %v", transport.profiles, transport.rotate)
	}
}
