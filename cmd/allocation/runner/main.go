package main

import (
	"math/rand"
	"os"

	"github.com/gocarina/gocsv"
	"github.com/mroth/weightedrand"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	soccer "github.com/stein-f/oink-soccer-common"
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

	eligibleAssets, err := appCtx.EligibleAssetRepository.GetAllEligibleAssets()
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	log.Info().Msgf("found %d eligible assets", len(eligibleAssets))

	assets, err := assignPlayersToEligibleAssets(appCtx.Rand, playersLookup, eligibleAssets)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	log.Info().Msgf("assigned %d players to assets", len(assets))

	if err := savePlayerAttributes(assets); err != nil {
		log.Fatal().Err(err).Send()
	}

	log.Info().Msgf("saved eligibility for %d assets. Run `grep Salah cmd/allocation/s3/out/assigned_players.csv` to search a player", len(assets))
}

func assignPlayersToEligibleAssets(r *rand.Rand, lookup *allocation.PlayersLookup, assets []allocation.EligibleAsset) ([]allocation.PlayerProfile, error) {
	assetProfiles := []allocation.PlayerProfile{}
	for _, asset := range assets {
		position, err := GetRandomPosition(r)
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

func GetRandomPosition(randSource *rand.Rand) (soccer.PlayerPosition, error) {
	playerChoices := []weightedrand.Choice{
		{Item: soccer.PlayerPositionGoalkeeper, Weight: 15},
		{Item: soccer.PlayerPositionDefense, Weight: 20},
		{Item: soccer.PlayerPositionMidfield, Weight: 20},
		{Item: soccer.PlayerPositionAttack, Weight: 20},
	}
	chooser, err := weightedrand.NewChooser(playerChoices...)
	if err != nil {
		return "", errors.Wrap(err, "failed to get player position")
	}
	return chooser.PickSource(randSource).(soccer.PlayerPosition), nil
}

func savePlayerAttributes(profiles []allocation.PlayerProfile) error {
	var csvRows []outputRecordRow
	for _, profile := range profiles {
		csvRows = append(csvRows, outputRecordRow{
			AssetID:    profile.Asset.AssetID,
			Name:       profile.Asset.Name,
			PlayerID:   profile.FifaPlayer.PlayerID,
			PlayerName: profile.FifaPlayer.PlayerAttributes.BasedOnPlayer,
		})
	}

	out, err := gocsv.MarshalBytes(&csvRows)
	if err != nil {
		return err
	}
	return os.WriteFile("cmd/allocation/s3/out/assigned_players.csv", out, 0644)
}

type outputRecordRow struct {
	AssetID    uint64 `csv:"asset_id"`
	Name       string `csv:"asset_name"`
	PlayerID   string `csv:"player_id"`
	PlayerName string `csv:"player_name"`
}
