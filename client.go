package jupiter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// DefaultEndpoint is the public Jupiter v6 quote/swap endpoint.
const DefaultEndpoint = "https://quote-api.jup.ag/v6"

// DefaultTokensEndpoint is the public Jupiter token list endpoint.
const DefaultTokensEndpoint = "https://tokens.jup.ag"

// Config holds the configuration for a Jupiter client.
type Config struct {
	// Endpoint is the Jupiter v6 base URL. Defaults to DefaultEndpoint.
	Endpoint string
	// TokensEndpoint is the token list base URL. Defaults to DefaultTokensEndpoint.
	TokensEndpoint string
	// APIKey is an optional API key for paid Jupiter tiers, sent as
	// the x-api-key header.
	APIKey string
	// HTTPClient is an optional *http.Client. Defaults to a 15s timeout client.
	HTTPClient *http.Client
	// MaxRetries is the maximum number of retry attempts for transient errors.
	// Defaults to 3 if zero.
	MaxRetries int
}

// Client is a thread-safe Jupiter API client.
type Client struct {
	endpoint       string
	tokensEndpoint string
	apiKey         string
	httpClient     *http.Client
	maxRetries     int
}

// NewClient creates a Jupiter client from the given configuration.
func NewClient(cfg Config) *Client {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	tokens := cfg.TokensEndpoint
	if tokens == "" {
		tokens = DefaultTokensEndpoint
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	retries := cfg.MaxRetries
	if retries <= 0 {
		retries = 3
	}
	return &Client{
		endpoint:       endpoint,
		tokensEndpoint: tokens,
		apiKey:         cfg.APIKey,
		httpClient:     hc,
		maxRetries:     retries,
	}
}

// doJSON performs an HTTP request with retries on transient errors and
// unmarshals the response body into out.
func (c *Client) doJSON(ctx context.Context, method, urlStr string, body any, out any) error {
	var payload []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("jupiter: marshal request: %w", err)
		}
		payload = b
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			if err := sleepCtx(ctx, backoff(attempt-1)); err != nil {
				return fmt.Errorf("jupiter: %w", err)
			}
		}

		respBody, err := c.do(ctx, method, urlStr, payload)
		if err != nil {
			lastErr = err
			if isRetryable(err) {
				continue
			}
			return err
		}

		if out != nil {
			if err := json.Unmarshal(respBody, out); err != nil {
				return fmt.Errorf("jupiter: unmarshal response: %w", err)
			}
		}
		return nil
	}

	return fmt.Errorf("jupiter: retries exhausted: %w", lastErr)
}

func (c *Client) do(ctx context.Context, method, urlStr string, payload []byte) ([]byte, error) {
	var reqBody io.Reader
	if payload != nil {
		reqBody = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, urlStr, reqBody)
	if err != nil {
		return nil, fmt.Errorf("jupiter: create request: %w", err)
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("x-api-key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &transportError{err: err, retryable: true}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("jupiter: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &transportError{
			err: &JupiterError{
				StatusCode: resp.StatusCode,
				Body:       string(body),
			},
			retryable: retryable(resp.StatusCode),
		}
	}

	return body, nil
}

// transportError is an internal wrapper carrying retry information.
type transportError struct {
	err       error
	retryable bool
}

func (e *transportError) Error() string { return e.err.Error() }
func (e *transportError) Unwrap() error { return e.err }

func isRetryable(err error) bool {
	te, ok := err.(*transportError)
	return ok && te.retryable
}

// buildQuoteURL builds the GET /quote URL with query parameters.
func (c *Client) buildQuoteURL(req QuoteRequest) string {
	q := url.Values{}
	q.Set("inputMint", req.InputMint)
	q.Set("outputMint", req.OutputMint)
	q.Set("amount", strconv.FormatUint(req.Amount, 10))
	if req.SlippageBps > 0 && !req.DynamicSlippage {
		q.Set("slippageBps", strconv.Itoa(req.SlippageBps))
	}
	if req.SwapMode != "" {
		q.Set("swapMode", string(req.SwapMode))
	}
	if req.OnlyDirectRoutes {
		q.Set("onlyDirectRoutes", "true")
	}
	if req.AsLegacyTransaction {
		q.Set("asLegacyTransaction", "true")
	}
	if req.PlatformFeeBps > 0 {
		q.Set("platformFeeBps", strconv.Itoa(req.PlatformFeeBps))
	}
	if req.MaxAccounts > 0 {
		q.Set("maxAccounts", strconv.Itoa(req.MaxAccounts))
	}
	if req.DynamicSlippage {
		q.Set("dynamicSlippage", "true")
	}
	return c.endpoint + "/quote?" + q.Encode()
}
