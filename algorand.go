package soccer

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"

	"github.com/rs/zerolog/log"
)

func CreateRandomSourceFromAlgorandBlockHash(httpCli *http.Client, round uint64) *rand.Rand {
	resp, err := httpCli.Get(fmt.Sprintf("https://mainnet-api.algonode.cloud/v2/blocks/%d/hash", round))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var resBody map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&resBody); err != nil {
		panic(err)
	}

	blockHash := resBody["blockHash"]

	// Compute SHA256 hash of the block
	hash := sha256.Sum256([]byte(blockHash))
	log.Debug().Msgf("Hash: %x", hash)

	// Convert the first 8 bytes of the hash to an int64 for seeding
	seed := int64(binary.BigEndian.Uint64(hash[:8]))

	// Seed the random number generator
	return rand.New(rand.NewSource(int64(math.Abs(float64(seed)))))
}
