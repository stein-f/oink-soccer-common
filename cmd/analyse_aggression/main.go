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
	var totalAggressionRating int
	var aggressivePlayers []allocation.FifaPlayer

	for _, player := range players {
		totalAggressionRating += player.PlayerAttributes.AggressionRating
		if player.PlayerAttributes.AggressionRating >= 80 {
			aggressivePlayers = append(aggressivePlayers, player)
			totalOverallRating += player.PlayerAttributes.OverallRating
		}
	}

	log.Info().Msgf("found %d aggressive players", len(aggressivePlayers))
	averageRatingOfAggressivePlayers := float64(totalOverallRating) / float64(len(aggressivePlayers))
	log.Info().Msgf("average overall rating of aggressive players: %f", averageRatingOfAggressivePlayers)
	log.Info().Msgf("average aggression rating of all players: %f", float64(totalAggressionRating)/float64(len(players)))

	sort.Slice(aggressivePlayers, func(i, j int) bool {
		return aggressivePlayers[i].PlayerAttributes.OverallRating > aggressivePlayers[j].PlayerAttributes.OverallRating
	})

	// log top 25 aggressive players
	for i, player := range aggressivePlayers {
		if i >= 25 {
			break
		}
		log.Info().Msgf("%s (%d) - %d", player.PlayerAttributes.BasedOnPlayer, player.PlayerAttributes.OverallRating, player.PlayerAttributes.AggressionRating)
	}

	// found 1046 aggressive players
	// average overall rating of aggressive players: 69.913002
}
