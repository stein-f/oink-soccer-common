package allocation

import (
	"os"

	"github.com/gocarina/gocsv"
)

type SoccerEligibleAssetsRepository interface {
	GetAllEligibleAssets() ([]EligibleAsset, error)
}

type EligibleAssetsRepository struct {
}

func (f EligibleAssetsRepository) GetAllEligibleAssets() ([]EligibleAsset, error) {
	file, err := os.ReadFile("cmd/allocation/s3/eligible_assets.csv")
	if err != nil {
		return nil, err
	}
	var records []csvRow
	err = gocsv.UnmarshalBytes(file, &records)
	if err != nil {
		return nil, err
	}
	var players []EligibleAsset
	for _, record := range records {
		players = append(players, EligibleAsset{
			Name:              record.Name,
			AssetID:           record.AssetID,
			EligibleAssetTier: record.Tier,
		})
	}
	return players, nil
}

type csvRow struct {
	Name    string            `csv:"name"`
	AssetID uint64            `csv:"asset_id"`
	Tier    EligibleAssetTier `csv:"tier"`
}
