// Package forgedashboard is a client for the forge-dashboard REST API
// (github.com/alrayyes/forge-dashboard).
//
// Authenticate with a personal API token, generated from the dashboard's
// own Settings page (POST /api/tokens) -- the alternative forge-dashboard's
// spec documents to its browser session cookie, and the one this SDK
// actually uses, since a headless client has no WebAuthn ceremony to run.
// Pass the token to New or set FORGE_DASHBOARD_TOKEN.
package forgedashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/alrayyes/forge-dashboard-sdk-go/internal/genclient"
)

// TokenEnvVar is the environment variable New falls back to when no
// token is passed explicitly via WithToken.
const TokenEnvVar = "FORGE_DASHBOARD_TOKEN"

// Client is a forge-dashboard API client. Construct one with New.
type Client struct {
	*genclient.ClientWithResponses
}

type config struct {
	token      string
	httpClient genclient.HttpRequestDoer
	retry      retryConfig
}

// Option configures a Client constructed by New.
type Option func(*config)

// WithToken sets the bearer token sent on every request, overriding
// FORGE_DASHBOARD_TOKEN.
func WithToken(token string) Option {
	return func(c *config) { c.token = token }
}

// WithHTTPClient replaces the underlying HTTP client. The retry transport
// (WithRetry) wraps whatever Transport this client has set, so pass a
// client with a custom Transport already configured (proxies, mTLS, ...)
// rather than replacing RoundTrip yourself afterward.
func WithHTTPClient(hc genclient.HttpRequestDoer) Option {
	return func(c *config) { c.httpClient = hc }
}

// WithRetry overrides the default retry policy. maxRetries is the number
// of additional attempts after the first; base is the starting backoff
// before jitter and any server-sent Retry-After.
func WithRetry(maxRetries int, base time.Duration) Option {
	return func(c *config) { c.retry = retryConfig{maxRetries: maxRetries, base: base} }
}

// New builds a Client against baseURL (the forge-dashboard instance's own
// origin, e.g. "https://dashboard.example.com").
func New(baseURL string, opts ...Option) (*Client, error) {
	cfg := config{
		token: os.Getenv(TokenEnvVar),
		retry: defaultRetryConfig,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	base := cfg.httpClient
	if base == nil {
		base = &http.Client{}
	}
	httpClient, ok := base.(*http.Client)
	if !ok {
		// A caller-supplied HttpRequestDoer that isn't a *http.Client (a
		// test double, usually) skips the retry transport -- there's no
		// http.RoundTripper to wrap.
		httpClient = nil
	}

	doer := base
	if httpClient != nil {
		wrapped := *httpClient
		wrapped.Transport = &retryTransport{
			next:   transportOrDefault(httpClient.Transport),
			config: cfg.retry,
		}
		doer = &wrapped
	}

	genOpts := []genclient.ClientOption{
		genclient.WithHTTPClient(doer),
		genclient.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
			if cfg.token != "" {
				req.Header.Set("Authorization", "Bearer "+cfg.token)
			}

			return nil
		}),
	}

	gc, err := genclient.NewClientWithResponses(baseURL, genOpts...)
	if err != nil {
		return nil, fmt.Errorf("build forge-dashboard client: %w", err)
	}

	return &Client{ClientWithResponses: gc}, nil
}

func transportOrDefault(t http.RoundTripper) http.RoundTripper {
	if t != nil {
		return t
	}

	return http.DefaultTransport
}

// APIError is returned by DecodeError for any forge-dashboard response
// carrying an error body -- every 4xx/5xx in the spec shares the same
// {error} shape (components.schemas.Error).
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("forge-dashboard: %d: %s", e.StatusCode, e.Message)
}

// DecodeError builds an *APIError from a generated response's status and
// body, or returns nil when statusCode isn't an error. Every operation's
// generated response type carries these two fields under the same names
// (StatusCode via HTTPResponse, and Body), so this one helper covers all
// of them:
//
//	resp, err := client.GetDashboardWithResponse(ctx, nil)
//	if err != nil {
//		return err // transport failure, already retried
//	}
//	if apiErr := forgedashboard.DecodeError(resp.HTTPResponse.StatusCode, resp.Body); apiErr != nil {
//		return apiErr
//	}
func DecodeError(statusCode int, body []byte) *APIError {
	if statusCode < 400 {
		return nil
	}
	var parsed struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(body, &parsed)

	return &APIError{
		StatusCode: statusCode,
		Message:    parsed.Error,
	}
}
