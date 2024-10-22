package main

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/gocarina/gocsv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stein-f/oink-soccer-common/cmd/allocation"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	appCtx, err := allocation.GetContext()
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	playersLookup, err := allocation.BuildPlayersLookup(appCtx.Rand, appCtx.FifaPlayerRepository)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	eligibleAssets, err := appCtx.EligibleAssetRepository.GetAllEligibleAssets(appCtx.Config.Season)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	log.Info().Msgf("found %d eligible assets", len(eligibleAssets))

	assets, err := assignPlayersToEligibleAssets(appCtx.Rand, playersLookup, eligibleAssets)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	log.Info().Msgf("assigned %d players to assets", len(assets))

	if err := savePlayerAttributes(assets, appCtx.Config.Season); err != nil {
		log.Fatal().Err(err).Send()
	}

	log.Info().Msgf("saved eligibility for %d assets. Run `grep Salah cmd/allocation/s%d/out/assigned_players.csv` to search a player", len(assets), appCtx.Config.Season)
}

func assignPlayersToEligibleAssets(r *rand.Rand, lookup *allocation.PlayersLookup, assets []allocation.EligibleAsset) ([]allocation.PlayerProfile, error) {
	assetProfiles := []allocation.PlayerProfile{}
	for _, asset := range assets {
		position, err := allocation.GetRandomPosition(r)
		if err != nil {
			return nil, err
		}
		player, err := lookup.GetRandomPlayer(position, asset)
		if err != nil {
			return nil, err
		}
		assetProfiles = append(assetProfiles, allocation.PlayerProfile{
			Asset:      asset,
			FifaPlayer: player,
		})
	}
	return assetProfiles, nil
}

func savePlayerAttributes(profiles []allocation.PlayerProfile, season int) error {
	var csvRows []outputRecordRow
	for _, profile := range profiles {
		csvRows = append(csvRows, outputRecordRow{
			PlayerID:     profile.Asset.PlayerID,
			Name:         profile.Asset.Name,
			FifaPlayerID: profile.FifaPlayer.PlayerID,
			PlayerName:   profile.FifaPlayer.PlayerAttributes.BasedOnPlayer,
		})
	}

	out, err := gocsv.MarshalBytes(&csvRows)
	if err != nil {
		return err
	}
	return os.WriteFile(fmt.Sprintf("cmd/allocation/s%d/out/assigned_players.csv", season), out, 0644)
}

type outputRecordRow struct {
	PlayerID     string `csv:"player_id"`
	Name         string `csv:"asset_name"`
	FifaPlayerID string `csv:"fifa_player_id"`
	PlayerName   string `csv:"player_name"`
}
