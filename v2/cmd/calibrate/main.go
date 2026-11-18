// Command calibrate replays real fixtures through the engine many times and
// reports how the results compare with football: goals, conversion, draws,
// 0-0s, blowouts, penalties and home advantage. It is how the contest model's
// constants are tuned; see docs/realism-review.md.
//
// Fixtures are lost-pigs match archives (one JSON file per match, as served by
// /soccer/game/{key}); only their line-ups are used, item boosts and the home
// advantage boost included, so the replay sees exactly what the real match did.
//
//	go run ./cmd/calibrate -archives <dir> [-seeds 20] [-model contest]
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"hash/fnv"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"strings"

	soccer "github.com/stein-f/oink-soccer-common/v2"
)

type fixture struct {
	Name string
	Home soccer.GameLineup `json:"home_team_lineup"`
	Away soccer.GameLineup `json:"away_team_lineup"`
}

type tally struct {
	matches, goals, shots         int
	homeWins, draws, awayWins     int
	nilNil, blowouts              int
	penalties                     int
	shotsByType, goalsByType      map[soccer.ChanceType]int
	goalsByPosition               map[soccer.PlayerPosition]int
	strongerWins, strongerMatches int

	// Step model only.
	stops, assisted, headed int
	missDetail              map[string]int
	totals                  soccer.TeamTotals
	withTotals              int
}

func main() {
	dir := flag.String("archives", "", "directory of lost-pigs match archive JSON files")
	seeds := flag.Int("seeds", 20, "replays per fixture")
	model := flag.String("model", "", `engine model: "" (classic), "contest" or "steps"`)
	limit := flag.Int("limit", 0, "use at most this many fixtures (0 = all)")
	flag.Parse()
	if *dir == "" {
		log.Fatal("-archives is required")
	}

	fixtures, err := loadFixtures(*dir, *limit)
	if err != nil {
		log.Fatal(err)
	}

	newTally := func() *tally {
		return &tally{
			shotsByType:     map[soccer.ChanceType]int{},
			goalsByType:     map[soccer.ChanceType]int{},
			goalsByPosition: map[soccer.PlayerPosition]int{},
			missDetail:      map[string]int{},
		}
	}
	// S18 fixtures range from evenly matched to a full side against a
	// team of placeholders, so the realism targets are read on close
	// fixtures and mismatches are reported on their own.
	all, close, mismatch := newTally(), newTally(), newTally()
	var closeFixtures, mismatchFixtures int
	for _, f := range fixtures {
		gap := math.Abs(lineupStrength(f.Home) - lineupStrength(f.Away))
		buckets := []*tally{all}
		switch {
		case gap < 5:
			buckets = append(buckets, close)
			closeFixtures++
		case gap >= 8:
			buckets = append(buckets, mismatch)
			mismatchFixtures++
		}
		positions := map[string]soccer.PlayerPosition{}
		for _, l := range []soccer.GameLineup{f.Home, f.Away} {
			for _, p := range l.Players {
				positions[p.ID] = p.SelectedPosition
			}
		}
		strongerHome := lineupStrength(f.Home) > lineupStrength(f.Away)
		for s := 0; s < *seeds; s++ {
			src := rand.New(rand.NewSource(seedFor(f.Name, s)))
			outcome, err := soccer.RunGameWithModel(src, f.Home, f.Away, soccer.Model(*model))
			if err != nil {
				log.Fatalf("%s: %v", f.Name, err)
			}
			for _, t := range buckets {
				t.add(outcome, positions, strongerHome)
			}
		}
	}
	report(*all, len(fixtures), *seeds, *model, "all fixtures")
	report(*close, closeFixtures, *seeds, *model, "close fixtures (mean overall within 5)")
	report(*mismatch, mismatchFixtures, *seeds, *model, "mismatches (mean overall 8+ apart)")
}

func (t *tally) add(outcome soccer.MatchOutcome, positions map[string]soccer.PlayerPosition, strongerHome bool) {
	events := outcome.Events
	if tt := outcome.Totals; tt != nil {
		t.withTotals++
		for _, side := range []soccer.TeamTotals{tt.Home, tt.Away} {
			t.totals.Possessions += side.Possessions
			t.totals.Passes += side.Passes
			t.totals.PassesCompleted += side.PassesCompleted
			t.totals.TakeOns += side.TakeOns
			t.totals.TakeOnsWon += side.TakeOnsWon
			t.totals.Crosses += side.Crosses
			t.totals.CrossesCompleted += side.CrossesCompleted
			t.totals.AerialsWon += side.AerialsWon
			t.totals.ShotsOnTarget += side.ShotsOnTarget
			t.totals.Woodwork += side.Woodwork
			t.totals.Tackles += side.Tackles
			t.totals.Interceptions += side.Interceptions
			t.totals.Clearances += side.Clearances
			t.totals.Blocks += side.Blocks
			t.totals.Saves += side.Saves
			t.totals.Fouls += side.Fouls
		}
	}
	{
		{
			home, away := 0, 0
			for _, e := range events {
				switch e.Type {
				case soccer.GameEventTypeStop:
					t.stops++
					continue
				case soccer.GameEventTypeMiss:
					if d := e.GetMissEvent().Detail; d != "" {
						t.missDetail[d]++
					}
				case soccer.GameEventTypeGoal:
					if e.GetGoalEvent().AssistID != "" {
						t.assisted++
					}
					if e.ChanceType == soccer.ChanceTypeCross || e.ChanceType == soccer.ChanceTypeCorner {
						t.headed++
					}
				}
				t.shots++
				t.shotsByType[e.ChanceType]++
				if e.ChanceType == soccer.ChanceTypePenalty {
					t.penalties++
				}
				if !e.IsGoal() {
					continue
				}
				g := e.GetGoalEvent()
				t.goals++
				t.goalsByType[e.ChanceType]++
				t.goalsByPosition[positions[g.PlayerID]]++
				if g.TeamType == soccer.TeamTypeHome {
					home++
				} else {
					away++
				}
			}
			t.matches++
			switch {
			case home > away:
				t.homeWins++
			case home < away:
				t.awayWins++
			default:
				t.draws++
			}
			if home+away == 0 {
				t.nilNil++
			}
			if abs(home-away) >= 5 {
				t.blowouts++
			}
			if home != away {
				t.strongerMatches++
				if (home > away) == strongerHome {
					t.strongerWins++
				}
			}
		}
	}
}

func loadFixtures(dir string, limit int) ([]fixture, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	if limit > 0 && len(paths) > limit {
		paths = paths[:limit]
	}
	var out []fixture
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		var f fixture
		if err := json.Unmarshal(b, &f); err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		if len(f.Home.Players) == 0 || len(f.Away.Players) == 0 {
			continue
		}
		f.Name = strings.TrimSuffix(filepath.Base(p), ".json")
		out = append(out, f)
	}
	return out, nil
}

// seedFor gives every (fixture, replay) its own fixed seed, so runs are
// repeatable and comparable between models.
func seedFor(name string, i int) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(name))
	return int64(h.Sum64()>>1) + int64(i)
}

// lineupStrength is a crude "who should win" measure: mean overall rating.
func lineupStrength(l soccer.GameLineup) float64 {
	var sum float64
	for _, p := range l.Players {
		sum += float64(p.Attributes.OverallRating)
	}
	return sum / float64(len(l.Players))
}

func report(t tally, fixtures, seeds int, model, label string) {
	if model == "" {
		model = "classic"
	}
	if t.matches == 0 {
		return
	}
	m := float64(t.matches)
	pct := func(n int) string { return fmt.Sprintf("%.1f%%", 100*float64(n)/m) }
	fmt.Printf("\n== %s — model %s: %d fixtures x %d seeds = %d matches\n\n", label, model, fixtures, seeds, t.matches)
	rows := [][3]string{
		{"Goals / match", fmt.Sprintf("%.2f", float64(t.goals)/m), "3.0-3.5"},
		{"Shots / match", fmt.Sprintf("%.2f", float64(t.shots)/m), "10-14"},
		{"Conversion", fmt.Sprintf("%.1f%%", 100*float64(t.goals)/float64(max(t.shots, 1))), "25-30%"},
		{"Home wins", pct(t.homeWins), ""},
		{"Draws", pct(t.draws), "18-22%"},
		{"Away wins", pct(t.awayWins), ""},
		{"Home - away (points)", fmt.Sprintf("%+.1f", 100*float64(t.homeWins-t.awayWins)/m), ">= +8"},
		{"0-0", pct(t.nilNil), "3-5%"},
		{"Wins by 5+", pct(t.blowouts), "< 3%"},
		{"Penalties / match", fmt.Sprintf("%.2f", float64(t.penalties)/m), "0.2-0.3"},
		{"Stronger side wins (decided games)", fmt.Sprintf("%.1f%%", 100*float64(t.strongerWins)/float64(max(t.strongerMatches, 1))), "clearly > 50%"},
	}
	fmt.Printf("%-36s %10s   %s\n", "metric", "value", "target")
	for _, r := range rows {
		fmt.Printf("%-36s %10s   %s\n", r[0], r[1], r[2])
	}

	fmt.Printf("\n%-16s %8s %8s\n", "chance type", "share", "conv")
	types := make([]soccer.ChanceType, 0, len(t.shotsByType))
	for ct := range t.shotsByType {
		types = append(types, ct)
	}
	sort.Slice(types, func(i, j int) bool { return t.shotsByType[types[i]] > t.shotsByType[types[j]] })
	for _, ct := range types {
		n := t.shotsByType[ct]
		fmt.Printf("%-16s %7.1f%% %7.1f%%\n", ct, 100*float64(n)/float64(t.shots), 100*float64(t.goalsByType[ct])/float64(n))
	}

	if t.withTotals > 0 {
		n := float64(t.withTotals)
		per := func(v int) string { return fmt.Sprintf("%.1f", float64(v)/n) }
		share := func(a, b int) string { return fmt.Sprintf("%.0f%%", 100*float64(a)/float64(max(b, 1))) }
		fmt.Printf("\nper match: possessions %s  passes %s (%s arrive)  take-ons %s (%s won)  crosses %s (%s in)\n",
			per(t.totals.Possessions), per(t.totals.Passes), share(t.totals.PassesCompleted, t.totals.Passes),
			per(t.totals.TakeOns), share(t.totals.TakeOnsWon, t.totals.TakeOns),
			per(t.totals.Crosses), share(t.totals.CrossesCompleted, t.totals.Crosses))
		fmt.Printf("           on target %s  saves %s  blocks %s  woodwork %s  tackles %s  interceptions %s  clearances %s  fouls %s  stops shown %s\n",
			per(t.totals.ShotsOnTarget), per(t.totals.Saves), per(t.totals.Blocks), per(t.totals.Woodwork),
			per(t.totals.Tackles), per(t.totals.Interceptions), per(t.totals.Clearances), per(t.totals.Fouls), per(t.stops))
		fmt.Printf("goals: assisted %s (target 60-75%%)  headed %s (target 10-20%%)\n", share(t.assisted, t.goals), share(t.headed, t.goals))
		fmt.Printf("misses:")
		for _, d := range []string{"blocked", "wide", "over", "post", "bar", "held", "parried"} {
			fmt.Printf("  %s %s", d, share(t.missDetail[d], t.shots-t.goals))
		}
		fmt.Println()
	}

	fmt.Printf("\ngoals by scorer position:")
	for _, pos := range []soccer.PlayerPosition{soccer.PlayerPositionAttack, soccer.PlayerPositionMidfield, soccer.PlayerPositionDefense, soccer.PlayerPositionGoalkeeper} {
		fmt.Printf("  %s %.0f%%", pos, 100*float64(t.goalsByPosition[pos])/float64(max(t.goals, 1)))
	}
	fmt.Println()
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
