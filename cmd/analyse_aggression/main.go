package main

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stein-f/oink-soccer-common/cmd/allocation"
	"os"
	"sort"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	appCtx, err := allocation.GetContext()
	if err != nil {
		panic(err)
	}

	players, err := appCtx.FifaPlayerRepository.GetAllPlayers()
	if err != nil {
		panic(err)
	}

	var totalOverallRating int
	var totalPhysicalRating int

	for _, player := range players {
		totalOverallRating += player.PlayerAttributes.OverallRating
	}

	log.Info().Msgf("average physical rating of all players: %f", float64(totalPhysicalRating)/float64(len(players)))

	sort.Slice(players, func(i, j int) bool {
		return players[i].PlayerAttributes.OverallRating > players[j].PlayerAttributes.OverallRating
	})
}
