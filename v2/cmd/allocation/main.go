// Command allocation runs the v2 season-start NFT allocation. It loads the
// shared config + CSV datasets, derives a deterministic seed from the season's
// Algorand block, runs the pure allocation core in v2/allocation, and writes
// the assignment CSV that lost-pigs imports.
//
// Run it from the v2 module directory:
//
//	cd v2 && go run ./cmd/allocation
//
// The season is taken from cmd/allocation/config.json (current_season). Pass
// -root to point at a repo checkout other than the auto-detected one.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/gocarina/gocsv"
	"github.com/stein-f/oink-soccer-common/v2/algorand"
	"github.com/stein-f/oink-soccer-common/v2/allocation"
)

func main() {
	root := flag.String("root", "", "repo root containing cmd/allocation (auto-detected when empty)")
	flag.Parse()

	dataRoot, err := resolveDataRoot(*root)
	if err != nil {
		log.Fatal(err)
	}

	season, err := loadCurrentSeason(dataRoot)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	log.Printf("allocating season %d (%s) from round %d", season.Season, season.Collab, season.Round)

	bs, err := algorand.NewClient(nil).FetchBlockSeed(context.Background(), season.Round)
	if err != nil {
		log.Fatalf("fetch block seed: %v", err)
	}
	log.Printf("seed derived from block %d hash %s", bs.Round, bs.BlockHash)

	candidates, err := loadCandidates(dataRoot, bs.Source)
	if err != nil {
		log.Fatalf("load candidates: %v", err)
	}
	log.Printf("loaded %d candidate players", len(candidates))

	assets, err := loadAssets(dataRoot, season.AssetsPath)
	if err != nil {
		log.Fatalf("load eligible assets: %v", err)
	}
	log.Printf("found %d eligible assets", len(assets))

	pool := allocation.NewPool(candidates, allocation.DefaultRules())
	assignments, err := allocation.Allocate(bs.Source, pool, assets)
	if err != nil {
		log.Fatalf("allocate: %v", err)
	}
	log.Printf("assigned %d players to assets", len(assignments))

	outPath := filepath.Join(dataRoot, fmt.Sprintf("cmd/allocation/s%d/out/assigned_players.csv", season.Season))
	if err := writeAssignments(outPath, assignments); err != nil {
		log.Fatalf("write output: %v", err)
	}
	log.Printf("wrote %s", outPath)
	log.Printf("spot-check a player with: grep Salah %s", outPath)
}

// resolveDataRoot returns the explicit -root if given, otherwise walks up from
// the current directory to find the repo checkout.
func resolveDataRoot(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return findDataRoot(cwd)
}

// outputRow matches the CSV shape consumed by lost-pigs'
// scripts/soccer/allocate_players_to_assets_v2.
type outputRow struct {
	PlayerID     string `csv:"player_id"`
	Name         string `csv:"asset_name"`
	FifaPlayerID string `csv:"fifa_player_id"`
	PlayerName   string `csv:"player_name"`
}

func writeAssignments(path string, assignments []allocation.Assignment) error {
	rows := make([]outputRow, 0, len(assignments))
	for _, a := range assignments {
		rows = append(rows, outputRow{
			PlayerID:     a.Asset.ID,
			Name:         a.Asset.Name,
			FifaPlayerID: a.Player.ID,
			PlayerName:   a.Player.Attributes.BasedOnPlayer,
		})
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	out, err := gocsv.MarshalBytes(&rows)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}
