package gateway

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"opencode2api/internal/config"
	"opencode2api/internal/headless"
	"opencode2api/internal/identity"
	"opencode2api/internal/models"
	wire "opencode2api/internal/protocol"
	"opencode2api/internal/telemetry"
)

// clientStreamKey carries the caller's stream flag. Anonymous bodies are
// rewritten to stream upstream, so the prepared body cannot answer this.
type clientStreamKey struct{}

func withClientStream(ctx context.Context, stream bool) context.Context {
	return context.WithValue(ctx, clientStreamKey{}, stream)
}

func clientRequestedStream(ctx context.Context) bool {
	stream, _ := ctx.Value(clientStreamKey{}).(bool)
	return stream
}

// serveHeadless satisfies one anonymous request through the official
// opencode serve process. The returned response is a Chat Completions
// document, or a chat SSE stream when the caller asked for one, so the
// gateway's existing translator can still adapt it to the client protocol.
// A non-nil response is the answer even when the status is not 2xx.
func (g *Gateway) serveHeadless(ctx context.Context, route models.Route, bodies map[config.Tier][]byte, ids identity.RequestIDs) (*http.Response, error) {
	started := time.Now()
	setRequestCredential(ctx, config.TierZen, "headless", "headless", true, nil)
	prompt := headless.ConversationText(headlessBody(bodies))
	if prompt == "" {
		resp := headlessErrorResponse(http.StatusBadRequest, "headless request has no text to send")
		g.recordUpstreamAttempt(ctx, route, ids, 1, "headless", "headless", true, nil, resp, nil, time.Since(started))
		return resp, nil
	}
	provider, modelID := headless.ModelRef(route.ID)
	result, err := (headless.Client{
		BaseURL:  g.cfg.Headless.BaseURL,
		Password: g.cfg.Headless.Password,
	}).Complete(ctx, provider, modelID, prompt)
	if err != nil {
		g.recordUpstreamAttempt(ctx, route, ids, 1, "headless", "headless", true, nil, nil, err, time.Since(started))
		if meta := telemetry.MetaFromContext(ctx); meta != nil {
			meta.Legs = append(meta.Legs, "headless:error:"+fmt.Sprint(time.Since(started).Milliseconds())+"ms")
		}
		return nil, err
	}
	var body []byte
	contentType := "application/json"
	if clientRequestedStream(ctx) {
		body = headless.ChatSSE(result)
		contentType = "text/event-stream"
	} else {
		body = headless.ChatJSON(result)
	}
	resp := &http.Response{
		StatusCode:    http.StatusOK,
		Status:        "200 OK",
		Header:        make(http.Header),
		Body:          io.NopCloser(bytes.NewReader(body)),
		ContentLength: int64(len(body)),
	}
	resp.Header.Set("Content-Type", contentType)
	g.recordUpstreamAttempt(ctx, route, ids, 1, "headless", "headless", true, nil, resp, nil, time.Since(started))
	return resp, nil
}

func headlessBody(bodies map[config.Tier][]byte) []byte {
	if body := bodies[config.TierZen]; len(body) > 0 {
		return body
	}
	for _, body := range bodies {
		if len(body) > 0 {
			return body
		}
	}
	return nil
}

func headlessErrorResponse(status int, message string) *http.Response {
	raw := []byte(`{"error":{"message":"` + message + `","type":"invalid_request_error"}}`)
	resp := &http.Response{
		StatusCode:    status,
		Status:        fmt.Sprintf("%d %s", status, http.StatusText(status)),
		Header:        make(http.Header),
		Body:          io.NopCloser(bytes.NewReader(raw)),
		ContentLength: int64(len(raw)),
	}
	resp.Header.Set("Content-Type", "application/json")
	return resp
}

// headlessRoute is the route the translator should see for a successful
// headless answer: a normal Chat body, not a forced anonymous SSE stream.
func headlessRoute(route models.Route) models.Route {
	route.Anonymous = false
	route.Protocol = wire.Chat
	if route.Protocols == nil {
		route.Protocols = map[config.Tier]wire.Protocol{}
	}
	route.Protocols[config.TierZen] = wire.Chat
	return route
}
