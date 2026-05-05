package jupiter

import (
	"context"
	"net/http"
)

// Quote fetches a swap quote from Jupiter for the given input/output mints
// and amount. The returned QuoteResponse must be passed verbatim to Swap.
func (c *Client) Quote(ctx context.Context, req QuoteRequest) (QuoteResponse, error) {
	var out QuoteResponse
	err := c.doJSON(ctx, http.MethodGet, c.buildQuoteURL(req), nil, &out)
	return out, err
}

// Swap builds a signed-by-user-only versioned transaction that performs the
// swap described by the quote. The returned base64 transaction still needs
// to be signed by the user's wallet before being submitted.
func (c *Client) Swap(ctx context.Context, req SwapRequest) (SwapResponse, error) {
	var out SwapResponse
	err := c.doJSON(ctx, http.MethodPost, c.endpoint+"/swap", req, &out)
	return out, err
}

// SwapInstructions returns the raw instructions, ALT addresses, and compute
// budget for the swap. Use this when you need to compose Jupiter's swap
// inside a larger transaction (e.g. with a Kora fee-payment instruction).
func (c *Client) SwapInstructions(ctx context.Context, req SwapRequest) (SwapInstructionsResponse, error) {
	var out SwapInstructionsResponse
	err := c.doJSON(ctx, http.MethodPost, c.endpoint+"/swap-instructions", req, &out)
	return out, err
}

// Tokens fetches the strict (verified) token list from tokens.jup.ag.
func (c *Client) Tokens(ctx context.Context) ([]Token, error) {
	var out []Token
	err := c.doJSON(ctx, http.MethodGet, c.tokensEndpoint+"/tokens?tags=verified", nil, &out)
	return out, err
}
