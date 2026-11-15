// Command convert_fc27 rewrites EA's FC 27 ratings export into the sofifa
// column layout the allocation and the lost-pigs importer read, carrying the
// curated legend cards over from the FIFA 22 dataset. See
// fifacsv.ConvertFC27 for the mapping.
//
// Download the export (no Kaggle account needed), then run from v2/:
//
//	curl -L -o fc27.zip https://www.kaggle.com/api/v1/datasets/download/mikedpad/ea-sports-fc27-player-ratings
//	unzip fc27.zip players.csv -d /tmp/fc27
//	go run ./cmd/convert_fc27 -in /tmp/fc27/players.csv
//
// The output (cmd/allocation/fifa_players_fc27.csv) must also be uploaded to
// s3://soccer-seasons-<env>/fifa_players_fc27.csv, byte-identical.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/stein-f/oink-soccer-common/v2/allocation/fifacsv"
)

func main() {
	in := flag.String("in", "", "FC 27 players.csv from the Kaggle export (required)")
	root := flag.String("root", "", "repo root containing cmd/allocation (auto-detected when empty)")
	flag.Parse()
	if *in == "" {
		flag.Usage()
		os.Exit(2)
	}

	dataRoot, err := findDataRoot(*root)
	if err != nil {
		log.Fatal(err)
	}

	fc27, err := os.Open(*in)
	if err != nil {
		log.Fatal(err)
	}
	defer fc27.Close()
	legacy, err := os.Open(filepath.Join(dataRoot, fifacsv.FIFA22Dataset))
	if err != nil {
		log.Fatal(err)
	}
	defer legacy.Close()

	var out bytes.Buffer
	stats, err := fifacsv.ConvertFC27(fc27, legacy, &out)
	if err != nil {
		log.Fatal(err)
	}
	outPath := filepath.Join(dataRoot, fifacsv.FC27Dataset)
	if err := os.WriteFile(outPath, out.Bytes(), 0o644); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("wrote %s: %d players (%d goalkeepers) + %d legend cards; skipped %d women's players\n",
		outPath, stats.Players, stats.Goalkeepers, stats.Legends, stats.SkippedWomen)
}

// findDataRoot returns explicit if set, otherwise walks up from the working
// directory to the checkout holding cmd/allocation/config.json.
func findDataRoot(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "cmd/allocation/config.json")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not locate cmd/allocation/config.json above the working directory")
		}
		dir = parent
	}
}
