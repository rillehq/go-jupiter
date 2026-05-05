# go-jupiter

A typed Go client for the [Jupiter v6 swap aggregator API](https://station.jup.ag/docs/apis/swap-api) on Solana.

Sister package to [`go-kora`](https://github.com/rillehq/go-kora). Use them together to build gasless stablecoin swaps where the user never holds SOL.

## Installation

```bash
go get github.com/rillehq/go-jupiter
```

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"log"

	jupiter "github.com/rillehq/go-jupiter"
)

const (
	USDT = "Es9vMFrzaCERmJfrF4H2FYD4KCoNkY11McCe8BenwNYB"
	USDC = "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v"
)

func main() {
	c := jupiter.NewClient(jupiter.Config{})
	ctx := context.Background()

	// 1. Get a quote (1 USDT → USDC).
	quote, err := c.Quote(ctx, jupiter.QuoteRequest{
		InputMint:   USDT,
		OutputMint:  USDC,
		Amount:      1_000_000, // 1 USDT (6 decimals)
		SlippageBps: 10,        // 0.10%
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Out:", quote.OutAmount, "Price impact:", quote.PriceImpactPct)

	// 2. Build a signable transaction for the user.
	swap, err := c.Swap(ctx, jupiter.SwapRequest{
		QuoteResponse: quote,
		UserPublicKey: "<user-wallet-address>",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Sign and send this base64 tx:", swap.SwapTransaction)
}
```

## Composing with go-kora (gasless swap)

When the user has no SOL, fetch raw instructions instead of a full transaction
and inject a Kora fee-payment instruction before signing.

```go
ix, err := c.SwapInstructions(ctx, jupiter.SwapRequest{
	QuoteResponse: quote,
	UserPublicKey: "<user-wallet>",
})
// ix.ComputeBudgetInstructions, ix.SetupInstructions, ix.SwapInstruction,
// ix.CleanupInstruction, ix.AddressLookupTableAddresses are all returned
// in the shape Solana SDKs expect. Combine with kora.GetPaymentInstruction
// in the same transaction.
```

## Error handling

```go
import "errors"

_, err := c.Quote(ctx, req)
if err != nil {
	var je *jupiter.JupiterError
	if errors.As(err, &je) {
		fmt.Printf("Jupiter %d: %s\n", je.StatusCode, je.Body)
	} else {
		fmt.Println("Transport error:", err)
	}
}
```

## Configuration

| Field            | Description                                       | Default                       |
|------------------|---------------------------------------------------|-------------------------------|
| `Endpoint`       | Jupiter v6 base URL                               | `https://quote-api.jup.ag/v6` |
| `TokensEndpoint` | Token list base URL                               | `https://tokens.jup.ag`       |
| `APIKey`         | Optional `x-api-key` for paid tiers               | —                             |
| `HTTPClient`     | Custom `*http.Client`                             | 15s timeout                   |
| `MaxRetries`     | Retries on 429 / 5xx / network errors             | 3                             |

## License

MIT
