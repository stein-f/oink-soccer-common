// Package algorand fetches block hashes from an Algorand indexer and turns
// them into deterministic random sources for the engine. v1 mixed HTTP I/O
// into the engine package (and panicked on failure); in v2 it lives here so
// the engine itself stays a pure function of (seed, lineups).
package algorand

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
)

// DefaultIndexerURL is the public Algonode endpoint v1 used.
const DefaultIndexerURL = "https://mainnet-api.algonode.cloud"

// Client fetches block hashes from an Algorand indexer.
type Client struct {
	HTTP    *http.Client
	BaseURL string
}

// NewClient returns a Client that talks to DefaultIndexerURL with the
// supplied http.Client. Pass nil to use http.DefaultClient.
func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{HTTP: httpClient, BaseURL: DefaultIndexerURL}
}

// BlockSeed pairs a fetched block hash with a deterministic *rand.Rand
// derived from it. Surfacing the hash alongside the source means callers
// can store it for audit / verification without re-fetching.
type BlockSeed struct {
	Round     uint64
	BlockHash string
	Source    *rand.Rand
}

// FetchBlockSeed fetches the block hash for round and derives a
// deterministic *rand.Rand from it. Returns an error rather than panicking
// (v1 panicked, which made it impossible to retry or fall back gracefully).
func (c *Client) FetchBlockSeed(ctx context.Context, round uint64) (BlockSeed, error) {
	url := fmt.Sprintf("%s/v2/blocks/%d/hash", c.BaseURL, round)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return BlockSeed{}, fmt.Errorf("algorand: build request: %w", err)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return BlockSeed{}, fmt.Errorf("algorand: fetch round %d: %w", round, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return BlockSeed{}, fmt.Errorf("algorand: round %d returned status %d", round, resp.StatusCode)
	}

	var body struct {
		BlockHash string `json:"blockHash"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return BlockSeed{}, fmt.Errorf("algorand: decode response: %w", err)
	}
	if body.BlockHash == "" {
		return BlockSeed{}, fmt.Errorf("algorand: empty block hash for round %d", round)
	}

	return BlockSeed{
		Round:     round,
		BlockHash: body.BlockHash,
		Source:    SeedFromBlockHash(body.BlockHash),
	}, nil
}

// SeedFromBlockHash deterministically derives a *rand.Rand from a block hash.
// The same string always produces the same source — this is what lets a
// match outcome be re-verified later from just the round number.
//
// v1 ran the result through math.Abs which has undefined behavior at
// math.MinInt64. Negative seeds are valid for rand.NewSource so v2 passes
// the value through directly. (This means v1 and v2 produce different
// streams from the same block hash; lost-pigs doesn't use this function so
// the change is safe.)
func SeedFromBlockHash(blockHash string) *rand.Rand {
	sum := sha256.Sum256([]byte(blockHash))
	seed := int64(binary.BigEndian.Uint64(sum[:8]))
	return rand.New(rand.NewSource(seed))
}
