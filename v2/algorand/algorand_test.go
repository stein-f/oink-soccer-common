package algorand_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stein-f/oink-soccer-common/v2/algorand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedFromBlockHashIsDeterministic(t *testing.T) {
	const hash = "TMUTUFAKGCDT4VHG2QJCIRR26ATBIWDOKDGDLDEQTGLAY34ZKDTA"

	a := algorand.SeedFromBlockHash(hash)
	b := algorand.SeedFromBlockHash(hash)

	// Same input ⇒ same first N draws. This is the *only* property the
	// engine depends on; the absolute values aren't important so we don't
	// pin them.
	for i := 0; i < 50; i++ {
		assert.Equal(t, a.Int63(), b.Int63(), "draw %d differs", i)
	}
}

func TestSeedFromBlockHashDiffersByInput(t *testing.T) {
	a := algorand.SeedFromBlockHash("hash-one")
	b := algorand.SeedFromBlockHash("hash-two")
	// Vanishingly unlikely collision, but worth pinning.
	assert.NotEqual(t, a.Int63(), b.Int63())
}

func TestFetchBlockSeed(t *testing.T) {
	const wantHash = "TMUTUFAKGCDT4VHG2QJCIRR26ATBIWDOKDGDLDEQTGLAY34ZKDTA"
	const wantRound uint64 = 35323350

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v2/blocks/35323350/hash", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]string{"blockHash": wantHash})
	}))
	defer srv.Close()

	client := &algorand.Client{HTTP: srv.Client(), BaseURL: srv.URL}
	got, err := client.FetchBlockSeed(context.Background(), wantRound)

	require.NoError(t, err)
	assert.Equal(t, wantRound, got.Round)
	assert.Equal(t, wantHash, got.BlockHash)
	require.NotNil(t, got.Source)

	// The returned source must match SeedFromBlockHash on the same hash.
	expected := algorand.SeedFromBlockHash(wantHash)
	assert.Equal(t, expected.Int63(), got.Source.Int63())
}

func TestFetchBlockSeedHandlesHTTPErrors(t *testing.T) {
	t.Run("non-200 status", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()
		client := &algorand.Client{HTTP: srv.Client(), BaseURL: srv.URL}
		_, err := client.FetchBlockSeed(context.Background(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "status 500")
	})

	t.Run("empty hash", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]string{"blockHash": ""})
		}))
		defer srv.Close()
		client := &algorand.Client{HTTP: srv.Client(), BaseURL: srv.URL}
		_, err := client.FetchBlockSeed(context.Background(), 1)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty block hash")
	})

	t.Run("malformed json", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("not json"))
		}))
		defer srv.Close()
		client := &algorand.Client{HTTP: srv.Client(), BaseURL: srv.URL}
		_, err := client.FetchBlockSeed(context.Background(), 1)
		require.Error(t, err)
	})
}

func TestNewClientUsesDefaults(t *testing.T) {
	c := algorand.NewClient(nil)
	assert.NotNil(t, c.HTTP)
	assert.Equal(t, algorand.DefaultIndexerURL, c.BaseURL)
}
