package main

import (
	"os"
	"path/filepath"

	"github.com/gocarina/gocsv"
	"github.com/stein-f/oink-soccer-common/v2/allocation"
)

// assetRow is one row of a season's eligible_assets.csv.
type assetRow struct {
	Name     string `csv:"name"`
	PlayerID string `csv:"player_id"`
	Tier     string `csv:"tier"`
}

// loadAssets reads the season's eligible assets. assetsPath comes straight from
// config.json (already relative to the repo root, e.g. cmd/allocation/s15/...).
func loadAssets(dataRoot, assetsPath string) ([]allocation.Asset, error) {
	raw, err := os.ReadFile(filepath.Join(dataRoot, assetsPath))
	if err != nil {
		return nil, err
	}
	var rows []assetRow
	if err := gocsv.UnmarshalBytes(raw, &rows); err != nil {
		return nil, err
	}
	assets := make([]allocation.Asset, 0, len(rows))
	for _, row := range rows {
		assets = append(assets, allocation.Asset{
			ID:   row.PlayerID,
			Name: row.Name,
			Tier: allocation.AssetTier(row.Tier),
		})
	}
	return assets, nil
}
