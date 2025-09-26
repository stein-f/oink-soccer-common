package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	soccer "github.com/stein-f/oink-soccer-common"
)

// gameKey is the unique id for a game. It's the last path segment in the highlights.
// https://www.thelostpigs.com/oink-soccer/player/TC0yLTYtMS05
const gameKey = "TC0xMi0yMC0xLTY="

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
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	retryableHttpCli := retryablehttp.NewClient()
	retryableHttpCli.Logger = nil
	httpCli := retryableHttpCli.StandardClient()

	event, err := fetchGameEventArchiveRecord(httpCli, gameKey)
	if err != nil {
		panic(err)
	}
	log.Debug().Msgf("Round: %d, Block hash: %s", event.RoundInfo.Round, event.RoundInfo.Hash)

	randSource := soccer.CreateRandomSourceFromAlgorandBlockHash(httpCli, event.RoundInfo.Round)

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

func runGame(r *rand.Rand, homeLineup, awayLineup soccer.GameLineup) {
	gameEvents, _, err := soccer.RunGameWithSeed(r, homeLineup, awayLineup)
	if err != nil {
		panic(err)
	}

	gameStats := soccer.CreateGameStats(gameEvents)

	log.Info().Msgf("%s %d - %d %s",
		homeLineup.Team.CustomName,
		gameStats.HomeTeamStats.Goals,
		gameStats.AwayTeamStats.Goals,
		awayLineup.Team.CustomName,
	)
}
