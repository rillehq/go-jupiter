package jupiter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const (
	usdcMint = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
	usdtMint = "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB"
)

func newTestClient(handler http.HandlerFunc) (*Client, *httptest.Server) {
	srv := httptest.NewServer(handler)
	c := NewClient(Config{
		Endpoint:       srv.URL,
		TokensEndpoint: srv.URL,
		HTTPClient:     &http.Client{Timeout: 2 * time.Second},
		MaxRetries:     1,
	})
	return c, srv
}

func TestQuote_Success(t *testing.T) {
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/quote" {
			t.Fatalf("expected /quote, got %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("inputMint") != usdtMint || q.Get("outputMint") != usdcMint {
			t.Fatalf("missing mint params: %v", q)
		}
		if q.Get("amount") != "1000000" {
			t.Fatalf("expected amount=1000000, got %s", q.Get("amount"))
		}
		if q.Get("slippageBps") != "10" {
			t.Fatalf("expected slippageBps=10, got %s", q.Get("slippageBps"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"inputMint": "`+usdtMint+`",
			"inAmount": "1000000",
			"outputMint": "`+usdcMint+`",
			"outAmount": "999500",
			"otherAmountThreshold": "998500",
			"swapMode": "ExactIn",
			"slippageBps": 10,
			"priceImpactPct": "0.0001",
			"routePlan": []
		}`)
	})
	defer srv.Close()

	got, err := c.Quote(context.Background(), QuoteRequest{
		InputMint:   usdtMint,
		OutputMint:  usdcMint,
		Amount:      1_000_000,
		SlippageBps: 10,
	})
	if err != nil {
		t.Fatalf("Quote: %v", err)
	}
	if got.OutAmount != "999500" {
		t.Fatalf("expected outAmount=999500, got %s", got.OutAmount)
	}
}

func TestQuote_Error(t *testing.T) {
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"error":"bad mint"}`)
	})
	defer srv.Close()

	_, err := c.Quote(context.Background(), QuoteRequest{
		InputMint:  usdtMint,
		OutputMint: usdcMint,
		Amount:     100,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	je, ok := err.(*JupiterError)
	if !ok {
		// May be wrapped — check unwrap chain.
		t.Logf("err type: %T", err)
		if !strings.Contains(err.Error(), "HTTP 400") {
			t.Fatalf("expected 400 in error, got: %v", err)
		}
		return
	}
	if je.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", je.StatusCode)
	}
}

func TestSwap_Success(t *testing.T) {
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/swap" {
			t.Fatalf("expected /swap, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		var req SwapRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if req.UserPublicKey != "user-pub" {
			t.Fatalf("expected user-pub, got %s", req.UserPublicKey)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"swapTransaction":"BASE64TX","lastValidBlockHeight":12345}`)
	})
	defer srv.Close()

	got, err := c.Swap(context.Background(), SwapRequest{
		QuoteResponse: QuoteResponse{InputMint: usdtMint, OutputMint: usdcMint},
		UserPublicKey: "user-pub",
	})
	if err != nil {
		t.Fatalf("Swap: %v", err)
	}
	if got.SwapTransaction != "BASE64TX" {
		t.Fatalf("expected BASE64TX, got %s", got.SwapTransaction)
	}
	if got.LastValidBlockHeight != 12345 {
		t.Fatalf("expected 12345, got %d", got.LastValidBlockHeight)
	}
}

func TestSwapInstructions_Success(t *testing.T) {
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/swap-instructions" {
			t.Fatalf("expected /swap-instructions, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{
			"computeBudgetInstructions": [],
			"setupInstructions": [],
			"swapInstruction": {"programId":"P","accounts":[],"data":"D"},
			"addressLookupTableAddresses": ["alt1"]
		}`)
	})
	defer srv.Close()

	got, err := c.SwapInstructions(context.Background(), SwapRequest{UserPublicKey: "u"})
	if err != nil {
		t.Fatalf("SwapInstructions: %v", err)
	}
	if got.SwapInstruction.ProgramID != "P" {
		t.Fatalf("expected programId=P, got %s", got.SwapInstruction.ProgramID)
	}
	if len(got.AddressLookupTableAddresses) != 1 {
		t.Fatalf("expected 1 ALT, got %d", len(got.AddressLookupTableAddresses))
	}
}

func TestRetry_OnServerError(t *testing.T) {
	calls := 0
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"inputMint":"a","outputMint":"b","inAmount":"1","outAmount":"1","otherAmountThreshold":"1","swapMode":"ExactIn","slippageBps":0,"priceImpactPct":"0","routePlan":[]}`)
	})
	defer srv.Close()

	_, err := c.Quote(context.Background(), QuoteRequest{InputMint: "a", OutputMint: "b", Amount: 1})
	if err != nil {
		t.Fatalf("Quote: %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls (1 retry), got %d", calls)
	}
}

func TestQuote_DynamicSlippage_OmitsSlippageBps(t *testing.T) {
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("slippageBps") != "" {
			t.Fatalf("expected slippageBps to be omitted with DynamicSlippage, got %s", q.Get("slippageBps"))
		}
		if q.Get("dynamicSlippage") != "true" {
			t.Fatalf("expected dynamicSlippage=true, got %s", q.Get("dynamicSlippage"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"inputMint":"a","outputMint":"b","inAmount":"1","outAmount":"1","otherAmountThreshold":"1","swapMode":"ExactIn","slippageBps":0,"priceImpactPct":"0","routePlan":[]}`)
	})
	defer srv.Close()

	_, err := c.Quote(context.Background(), QuoteRequest{
		InputMint:       "a",
		OutputMint:      "b",
		Amount:          1,
		SlippageBps:     50,
		DynamicSlippage: true,
	})
	if err != nil {
		t.Fatalf("Quote: %v", err)
	}
}
