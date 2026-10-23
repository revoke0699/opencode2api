package gateway

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"opencode2api/internal/config"
	"opencode2api/internal/headless"
	"opencode2api/internal/identity"
	"opencode2api/internal/models"
	wire "opencode2api/internal/protocol"
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
	prompt := headless.ConversationText(headlessBody(bodies))
	if prompt == "" {
		resp := headlessErrorResponse(http.StatusBadRequest, "headless request has no text to send")
		g.recordUpstreamAttempt(ctx, route, ids, 1, "headless", "headless", true, nil, resp, nil, 0)
		return resp, nil
	}
	instances := g.headless.order(g.cfg.Headless.Endpoints(), time.Now())
	if len(instances) == 0 {
		return nil, fmt.Errorf("every headless identity is cooling")
	}
	provider, modelID := headless.ModelRef(route.ID)
	var lastErr error
	for attempt, inst := range instances {
		if ctx.Err() != nil {
			if lastErr == nil {
				lastErr = ctx.Err()
			}
			break
		}
		label := headlessLabel(inst.BaseURL)
		started := time.Now()
		setRequestCredential(ctx, config.TierZen, label, "headless", false, nil)
		password := inst.Password
		if password == "" {
			password = g.cfg.Headless.Password
		}
		attemptCtx, cancel := context.WithTimeout(ctx, headlessAttemptTimeout)
		result, err := (headless.Client{
			BaseURL:  inst.BaseURL,
			Password: password,
		}).Complete(attemptCtx, provider, modelID, prompt)
		cancel()
		if err != nil {
			lastErr = err
			g.recordUpstreamAttempt(ctx, route, ids, attempt+1, label, "headless", false, nil, nil, err, time.Since(started))
			cool := headlessTransportCool
			if headless.IdentityBlocked(err) {
				cool = g.freeTierBlockTTL()
				g.logger.Warn("headless identity blocked; rotating", "component", "upstream", "event", "headless_identity_blocked", "request_id", ids.Request, "identity", label, "block_for", cool.String())
			}
			g.headless.cool(inst.BaseURL, time.Now().Add(cool))
			continue
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
		g.recordUpstreamAttempt(ctx, route, ids, attempt+1, label, "headless", false, nil, resp, nil, time.Since(started))
		g.headless.succeed(inst.BaseURL)
		return resp, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("every headless identity is cooling")
	}
	return nil, lastErr
}

func headlessLabel(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "headless"
	}
	return u.Host
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
