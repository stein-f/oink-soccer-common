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

	// Define what "aggressive" means for this comparison.
	const aggressionThreshold = 80

	var (
		totalOverallAllPlayers int
		totalOverallAggressive int
		aggressivePlayers      []allocation.FifaPlayer
	)

	for _, player := range players {
		totalOverallAllPlayers += player.PlayerAttributes.OverallRating
		if player.PlayerAttributes.AggressionRating >= aggressionThreshold {
			aggressivePlayers = append(aggressivePlayers, player)
			totalOverallAggressive += player.PlayerAttributes.OverallRating
		}
	}

	log.Info().Msgf("found %d aggressive players (threshold: %d)", len(aggressivePlayers), aggressionThreshold)

	var (
		avgOverallAll float64
		avgOverallAgg float64
	)

	if len(players) > 0 {
		avgOverallAll = float64(totalOverallAllPlayers) / float64(len(players))
	}
	if len(aggressivePlayers) > 0 {
		avgOverallAgg = float64(totalOverallAggressive) / float64(len(aggressivePlayers))
	}

	log.Info().Msgf("average overall rating - aggressive players: %.3f", avgOverallAgg)
	log.Info().Msgf("average overall rating - all players: %.3f", avgOverallAll)
	log.Info().Msgf("difference (aggressive - all): %.3f", avgOverallAgg-avgOverallAll)

	// Sort and show top 25 aggressive players by overall rating
	sort.Slice(aggressivePlayers, func(i, j int) bool {
		return aggressivePlayers[i].PlayerAttributes.OverallRating > aggressivePlayers[j].PlayerAttributes.OverallRating
	})
	for i, player := range aggressivePlayers {
		if i >= 25 {
			break
		}
		log.Info().Msgf("%s (%d) - Aggression %d",
			player.PlayerAttributes.BasedOnPlayer,
			player.PlayerAttributes.OverallRating,
			player.PlayerAttributes.AggressionRating,
		)
	}
}
