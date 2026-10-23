package httpx

import (
	"bytes"
	"net/http"
	"strings"
	"testing"
)

func requestHeadFor(t *testing.T, leading bool) []string {
	t.Helper()
	body := []byte(`{"model":"m"}`)
	req, err := http.NewRequest(http.MethodPost, "https://example.test/zen/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer public")
	req.Header.Set("User-Agent", V1ModelUserAgent(V1ChatSDK))
	head, _, err := bunRequestHead(req, body, leading)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSuffix(string(head), "\r\n\r\n"), "\r\n")
}

func headerIndex(lines []string, prefix string) int {
	for i, line := range lines {
		if strings.HasPrefix(line, prefix) {
			return i
		}
	}
	return -1
}

// The 1.x client's fetch leaves Content-Length after Accept-Encoding; the 2.x
// runtime writes it in the common-header block. A shared transport must place
// it per lane.
func TestBunRequestHeadContentLengthPlacement(t *testing.T) {
	trail := requestHeadFor(t, false)
	if got := headerIndex(trail, "Content-Length:"); got != len(trail)-1 {
		t.Errorf("v1 Content-Length line = %d of %d, want last\n%s", got, len(trail), strings.Join(trail, "\n"))
	}
	if encoding, length := headerIndex(trail, "Accept-Encoding:"), headerIndex(trail, "Content-Length:"); encoding < 0 || encoding > length {
		t.Errorf("v1 Accept-Encoding line = %d, Content-Length = %d; want Accept-Encoding first", encoding, length)
	}

	lead := requestHeadFor(t, true)
	length, contentType := headerIndex(lead, "Content-Length:"), headerIndex(lead, "Content-Type:")
	if length < 0 || contentType < 0 || length > contentType {
		t.Errorf("v2 Content-Length line = %d, Content-Type = %d; want Content-Length in the common block", length, contentType)
	}
	if authorization := headerIndex(lead, "Authorization:"); authorization < 0 || authorization > length {
		t.Errorf("v2 Authorization line = %d, Content-Length = %d; want Authorization first", authorization, length)
	}
}
