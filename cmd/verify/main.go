package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"

	retryablehttp "github.com/hashicorp/go-retryablehttp"

	soccer "github.com/stein-f/oink-soccer-common"
)

// gameKey is the unique id for a game. It's the last path segment in the highlights.
// https://www.thelostpigs.com/oink-soccer/player/TC0yLTYtMS05
const gameKey = "TC0yLTctMS0x"

type GameEventArchiveRecord struct {
	HomeTeamLineup soccer.GameLineup `json:"home_team_lineup"`
	AwayTeamLineup soccer.GameLineup `json:"away_team_lineup"`
	RoundInfo      LatestRoundInfo   `json:"round_info"`
}

type LatestRoundInfo struct {
	Round uint64 `json:"round"`
	Hash  string `json:"hash"`
}

// this script provides an example of verifying a game against a blockchain round hash
// running the game repeatedly with the same players and round should produce the same result
func main() {
	httpCli := retryablehttp.NewClient().StandardClient()

	event, err := fetchGameEventArchiveRecord(httpCli, gameKey)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Round: %d, Block hash: %s\n", event.RoundInfo.Round, event.RoundInfo.Hash)

	randSource := fetchSeedFromBlockHash(httpCli, event.RoundInfo.Round)

	runGame(randSource, event.HomeTeamLineup, event.AwayTeamLineup)
}

func fetchGameEventArchiveRecord(client *http.Client, gameKey string) (GameEventArchiveRecord, error) {
	resp, err := client.Get("https://api.thelostpigs.com/soccer/game/" + gameKey)
	if err != nil {
		return GameEventArchiveRecord{}, fmt.Errorf("failed to get game event archive record %w", err)
	}
	defer resp.Body.Close()

	var archiveRecord GameEventArchiveRecord
	if err := json.NewDecoder(resp.Body).Decode(&archiveRecord); err != nil {
		return GameEventArchiveRecord{}, fmt.Errorf("failed to decode game event archive record %w", err)
	}
	return archiveRecord, nil
}

func fetchSeedFromBlockHash(httpCli *http.Client, round uint64) *rand.Rand {
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
	fmt.Printf("Hash: %x\n", hash)

	// Convert the first 8 bytes of the hash to an int64 for seeding
	seed := int64(binary.BigEndian.Uint64(hash[:8]))

	// Seed the random number generator
	return rand.New(rand.NewSource(seed))
}

func runGame(r *rand.Rand, homeLineup, awayLineup soccer.GameLineup) {
	gameEvents, err := soccer.RunGameWithSeed(r, homeLineup, awayLineup)
	if err != nil {
		panic(err)
	}

	gameStats := soccer.CreateGameStats(gameEvents)

	fmt.Printf("%s %d - %d %s\n",
		homeLineup.Team.CustomName,
		gameStats.HomeTeamStats.Goals,
		gameStats.AwayTeamStats.Goals,
		awayLineup.Team.CustomName,
	)
}
