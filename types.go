// Package jupiter provides a typed Go client for the Jupiter v6 swap aggregator
// API on Solana (https://station.jup.ag/docs/apis/swap-api).
package jupiter

import "fmt"

// SwapMode controls whether the amount is the exact input or exact output.
type SwapMode string

const (
	SwapModeExactIn  SwapMode = "ExactIn"
	SwapModeExactOut SwapMode = "ExactOut"
)

// JupiterError represents a non-2xx response from the Jupiter API.
type JupiterError struct {
	// StatusCode is the HTTP status code returned by Jupiter.
	StatusCode int
	// Body is the raw response body as returned by Jupiter.
	Body string
}

// Error implements the error interface.
func (e *JupiterError) Error() string {
	return fmt.Sprintf("jupiter api error: HTTP %d: %s", e.StatusCode, e.Body)
}

// --- Quote ---

// QuoteRequest is the parameter set for the GET /quote endpoint.
type QuoteRequest struct {
	// InputMint is the mint address of the input token (required).
	InputMint string
	// OutputMint is the mint address of the output token (required).
	OutputMint string
	// Amount is the amount in the input token's smallest unit when SwapMode is
	// ExactIn (default), or the output token's smallest unit when ExactOut.
	Amount uint64
	// SlippageBps is the maximum slippage tolerated, in basis points
	// (e.g. 50 = 0.50%). Ignored when DynamicSlippage is true.
	SlippageBps int
	// SwapMode is ExactIn (default) or ExactOut.
	SwapMode SwapMode
	// OnlyDirectRoutes restricts routing to single-hop swaps.
	OnlyDirectRoutes bool
	// AsLegacyTransaction returns a legacy (non-versioned) transaction shape.
	AsLegacyTransaction bool
	// PlatformFeeBps charges a fee on the swap, in basis points. Requires the
	// swap request to specify a feeAccount.
	PlatformFeeBps int
	// MaxAccounts caps the number of accounts in the resulting transaction.
	MaxAccounts int
	// DynamicSlippage enables Jupiter's RPC-aware slippage estimator.
	DynamicSlippage bool
}

// QuoteResponse is the body returned by GET /quote. It is the exact JSON
// shape Jupiter returns and must be passed verbatim to Swap.
type QuoteResponse struct {
	InputMint            string         `json:"inputMint"`
	InAmount             string         `json:"inAmount"`
	OutputMint           string         `json:"outputMint"`
	OutAmount            string         `json:"outAmount"`
	OtherAmountThreshold string         `json:"otherAmountThreshold"`
	SwapMode             string         `json:"swapMode"`
	SlippageBps          int            `json:"slippageBps"`
	PlatformFee          *PlatformFee   `json:"platformFee,omitempty"`
	PriceImpactPct       string         `json:"priceImpactPct"`
	RoutePlan            []RoutePlanHop `json:"routePlan"`
	ContextSlot          uint64         `json:"contextSlot,omitempty"`
	TimeTaken            float64        `json:"timeTaken,omitempty"`
}

// PlatformFee describes a partner fee charged on the swap.
type PlatformFee struct {
	Amount string `json:"amount"`
	FeeBps int    `json:"feeBps"`
}

// RoutePlanHop is one hop in the swap route.
type RoutePlanHop struct {
	SwapInfo SwapInfo `json:"swapInfo"`
	Percent  int      `json:"percent"`
}

// SwapInfo describes a single AMM/pool used in a hop.
type SwapInfo struct {
	AmmKey     string `json:"ammKey"`
	Label      string `json:"label,omitempty"`
	InputMint  string `json:"inputMint"`
	OutputMint string `json:"outputMint"`
	InAmount   string `json:"inAmount"`
	OutAmount  string `json:"outAmount"`
	FeeAmount  string `json:"feeAmount"`
	FeeMint    string `json:"feeMint"`
}

// --- Swap ---

// SwapRequest is the body for POST /swap and POST /swap-instructions.
type SwapRequest struct {
	// QuoteResponse is the unmodified response from a previous Quote call.
	QuoteResponse QuoteResponse `json:"quoteResponse"`
	// UserPublicKey is the wallet that will sign and pay for the swap.
	UserPublicKey string `json:"userPublicKey"`
	// WrapAndUnwrapSol auto-wraps SOL → wSOL and unwraps on the way back.
	// Defaults to true on Jupiter's side.
	WrapAndUnwrapSol *bool `json:"wrapAndUnwrapSol,omitempty"`
	// UseSharedAccounts uses Jupiter's shared accounts to reduce tx size.
	UseSharedAccounts *bool `json:"useSharedAccounts,omitempty"`
	// FeeAccount is the token account that receives the platformFeeBps fee.
	FeeAccount string `json:"feeAccount,omitempty"`
	// ComputeUnitPriceMicroLamports sets a static priority fee.
	ComputeUnitPriceMicroLamports *uint64 `json:"computeUnitPriceMicroLamports,omitempty"`
	// PrioritizationFeeLamports lets Jupiter auto-price the priority fee.
	// Pass "auto" or a numeric string.
	PrioritizationFeeLamports interface{} `json:"prioritizationFeeLamports,omitempty"`
	// AsLegacyTransaction returns a legacy (non-versioned) transaction.
	AsLegacyTransaction *bool `json:"asLegacyTransaction,omitempty"`
	// DestinationTokenAccount overrides the default ATA for the output token.
	DestinationTokenAccount string `json:"destinationTokenAccount,omitempty"`
	// DynamicComputeUnitLimit sizes the CU budget from a simulated swap.
	DynamicComputeUnitLimit *bool `json:"dynamicComputeUnitLimit,omitempty"`
	// SkipUserAccountsRpcCalls skips Jupiter's pre-flight account checks.
	SkipUserAccountsRpcCalls *bool `json:"skipUserAccountsRpcCalls,omitempty"`
}

// SwapResponse is the body returned by POST /swap.
type SwapResponse struct {
	// SwapTransaction is the base64-encoded versioned transaction ready to sign.
	SwapTransaction string `json:"swapTransaction"`
	// LastValidBlockHeight is the block height after which the tx will fail.
	LastValidBlockHeight uint64 `json:"lastValidBlockHeight"`
	// PrioritizationFeeLamports is the priority fee chosen by Jupiter.
	PrioritizationFeeLamports uint64 `json:"prioritizationFeeLamports,omitempty"`
}

// --- Swap instructions ---

// SwapInstructionsResponse is the body returned by POST /swap-instructions.
// Use this when you need to compose Jupiter's swap inside a larger transaction
// (for example with a Kora fee payment instruction).
type SwapInstructionsResponse struct {
	TokenLedgerInstruction      *Instruction  `json:"tokenLedgerInstruction,omitempty"`
	ComputeBudgetInstructions   []Instruction `json:"computeBudgetInstructions"`
	SetupInstructions           []Instruction `json:"setupInstructions"`
	SwapInstruction             Instruction   `json:"swapInstruction"`
	CleanupInstruction          *Instruction  `json:"cleanupInstruction,omitempty"`
	OtherInstructions           []Instruction `json:"otherInstructions,omitempty"`
	AddressLookupTableAddresses []string      `json:"addressLookupTableAddresses"`
	PrioritizationFeeLamports   uint64        `json:"prioritizationFeeLamports,omitempty"`
}

// Instruction is a Solana instruction in the shape Jupiter returns.
type Instruction struct {
	ProgramID string             `json:"programId"`
	Accounts  []InstructionAccount `json:"accounts"`
	// Data is base64-encoded instruction data.
	Data string `json:"data"`
}

// InstructionAccount is one account meta on a Jupiter-returned instruction.
type InstructionAccount struct {
	Pubkey     string `json:"pubkey"`
	IsSigner   bool   `json:"isSigner"`
	IsWritable bool   `json:"isWritable"`
}

// --- Tokens ---

// Token is one entry in the Jupiter strict/all token list.
type Token struct {
	Address  string   `json:"address"`
	Symbol   string   `json:"symbol"`
	Name     string   `json:"name"`
	Decimals int      `json:"decimals"`
	LogoURI  string   `json:"logoURI,omitempty"`
	Tags     []string `json:"tags,omitempty"`
}
