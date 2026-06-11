package allocation

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// makeAssets builds a synthetic, deterministic set of eligible assets across every
// tier. It is independent of the per-season CSVs (which change each season) so the
// invariants below stay meaningful over time. Tier S+A is sized comfortably larger
// than the legend pool so every legend has a host.
func makeAssets() []EligibleAsset {
	plan := []struct {
		tier  EligibleAssetTier
		count int
	}{
		{EligibleAssetTierS, 80},
		{EligibleAssetTierA, 80},
		{EligibleAssetTierB, 60},
		{EligibleAssetTierC, 300},
		{EligibleAssetTierAggressive, 40},
	}
	var assets []EligibleAsset
	for _, p := range plan {
		for i := 0; i < p.count; i++ {
			id := fmt.Sprintf("%s-%d", strings.ReplaceAll(string(p.tier), " ", ""), i)
			assets = append(assets, EligibleAsset{
				PlayerID:          id,
				Name:              id,
				EligibleAssetTier: p.tier,
			})
		}
	}
	return assets
}

// TestEachLegendAllocatedExactlyOnce locks in the allocation contract: across any
// seed, every curated legend card is allocated exactly once, only to Tier S/A
// assets, with no duplicates and no empty assignments.
func TestEachLegendAllocatedExactlyOnce(t *testing.T) {
	t.Chdir("../..") // repo root: FifaPlayersRepository reads cmd/allocation/fifa_players_22.csv

	assets := makeAssets()
	tierOf := make(map[string]EligibleAssetTier, len(assets))
	for _, a := range assets {
		tierOf[a.PlayerID] = a.EligibleAssetTier
	}

	for _, seed := range []int64{1, 42, 20260609, 777, 999999} {
		t.Run(fmt.Sprintf("seed-%d", seed), func(t *testing.T) {
			r := rand.New(rand.NewSource(seed))
			lookup, err := BuildPlayersLookup(r, FifaPlayersRepository{RandSource: r})
			if err != nil {
				t.Fatal(err)
			}

			legends := lookup.LegendCards()
			if len(legends) == 0 {
				t.Fatal("no legend cards in pool")
			}

			profiles, err := AssignPlayers(r, lookup, assets)
			if err != nil {
				t.Fatal(err)
			}
			if len(profiles) != len(assets) {
				t.Fatalf("assigned %d profiles for %d assets", len(profiles), len(assets))
			}

			counts := make(map[string]int)
			var empties int
			for _, p := range profiles {
				id := p.FifaPlayer.PlayerID
				if id == "" {
					empties++
					continue
				}
				if !strings.HasPrefix(id, LegendCardPrefix) {
					continue
				}
				counts[id]++
				if tr := tierOf[p.Asset.PlayerID]; tr != EligibleAssetTierS && tr != EligibleAssetTierA {
					t.Errorf("legend %s placed on %s asset, want Tier S/A", id, tr)
				}
			}

			if empties != 0 {
				t.Errorf("got %d empty assignments, want 0", empties)
			}
			if len(counts) != len(legends) {
				t.Errorf("allocated %d distinct legends, want all %d", len(counts), len(legends))
			}
			for _, l := range legends {
				if c := counts[l.PlayerID]; c != 1 {
					t.Errorf("legend %s allocated %d times, want exactly 1", l.PlayerID, c)
				}
			}
		})
	}
}

func TestFindPlayerLevel(t *testing.T) {
	tests := []struct {
		name          string
		playerID      string
		overallRating int
		want          string
	}{
		{name: "curated legend card is legendary regardless of rating", playerID: "L0001", overallRating: 80, want: "Legendary"},
		{name: "real player in legendary band is capped at world class", playerID: "158023", overallRating: 93, want: "World class"},
		{name: "world class band", playerID: "188545", overallRating: 84, want: "World class"},
		{name: "professional band", playerID: "200000", overallRating: 75, want: "Professional"},
		{name: "semi professional band", playerID: "200001", overallRating: 60, want: "Semi Professional"},
		{name: "amateur band", playerID: "200002", overallRating: 40, want: "Amateur"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findPlayerLevel(tt.playerID, tt.overallRating); string(got) != tt.want {
				t.Errorf("findPlayerLevel(%s, %d) = %s, want %s", tt.playerID, tt.overallRating, got, tt.want)
			}
		})
	}
}
