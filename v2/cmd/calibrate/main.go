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

// gapBuckets are the lower bounds of the strength-gap bands the favourite's
// results are reported in (points of mean overall rating).
var gapBuckets = [...]float64{0, 1, 3, 5, 8}

func gapBucket(gap float64) int {
	b := 0
	for i, lo := range gapBuckets {
		if gap >= lo {
			b = i
		}
	}
	return b
}

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

	// How the stronger side (by mean overall) did, by how much stronger.
	favourite [len(gapBuckets)]struct{ matches, wins, draws int }

	// likelier is, summed over fixtures, how often the result each fixture
	// most often produced came up: a direct measure of how much of a match
	// is decided before kick-off. 100% means the same fixture always ends
	// the same way; a coin flip between three results is 33%.
	likelierWins, likelierMatches int
	// For the correlation between a fixture's rating gap (home minus away,
	// mean overall) and its average points difference: how much of who wins
	// the ratings on a player's card explain.
	gapN, gapX, gapY, gapXX, gapYY, gapXY float64

	// Step model only.
	stops, assisted, headed int
	missDetail              map[string]int
	// Goals worth a second look, by kind, and the chances that could
	// have been: shots at the end of a solo run, from a rebound (by what
	// it came off) and by a goalkeeper.
	goalDetail, special map[string]int
	totals              soccer.TeamTotals
	withTotals          int
}

func main() {
	dir := flag.String("archives", "", "directory of lost-pigs match archive JSON files")
	seeds := flag.Int("seeds", 20, "replays per fixture")
	model := flag.String("model", "", `engine model: "" (classic), "contest" or "steps"`)
	limit := flag.Int("limit", 0, "use at most this many fixtures (0 = all)")
	dataset := flag.String("enrich", "", "player dataset CSV to fill in what archived players lack (see enrich.go)")
	enrichMode := flag.String("enrich-mode", "steps", `"steps": add the player-on-player attributes; "all": replace every attribute`)
	attributes := flag.Int("attributes", 0, "instead of replaying fixtures, report what each attribute is worth, over this many seeds")
	attributeDelta := flag.Int("attribute-delta", 5, "attribute report: how many points to raise each attribute by")
	formationsDir := flag.String("formations", "", "instead of replaying fixtures, run the formations gate on this directory of team files (see formations.go)")
	dealAssets := flag.String("deal", "", "formations report: eligible assets CSV; deal every squad afresh from -deal-dataset (see deal.go)")
	dealData := flag.String("deal-dataset", "", "formations report: the dataset to deal from")
	dealSeed := flag.Int64("deal-seed", 1, "formations report: seed for the deal")
	lineupsPath := flag.String("lineups", "", "formations report: cache file for every team's best line-up per formation (written if missing, read if present)")
	seasons := flag.Int("seasons", 1000, "formations report: seasons to play for the strongest team's title odds (0 to skip)")
	levers := flag.Int("levers", 0, "instead of replaying fixtures, report what boosts, tactics and roles are worth, over this many seeds")
	flag.Parse()
	if *attributes > 0 {
		attributeReport(*attributes, soccer.Model(*model), *attributeDelta)
		return
	}
	if *levers > 0 {
		leversReport(*levers, soccer.Model(*model))
		return
	}
	if *formationsDir != "" {
		var e *enricher
		if *dataset != "" {
			var err error
			if e, err = newEnricher(*dataset, *enrichMode); err != nil {
				log.Fatal(err)
			}
		}
		if err := formationsReport(formationsOptions{
			dir: *formationsDir, model: soccer.Model(*model), enrich: e,
			dealAssets: *dealAssets, dealData: *dealData, dealSeed: *dealSeed,
			finalSeeds: *seeds * 10, seasons: *seasons, lineupsPath: *lineupsPath,
		}); err != nil {
			log.Fatal(err)
		}
		return
	}
	if *dir == "" {
		log.Fatal("-archives is required")
	}

	fixtures, err := loadFixtures(*dir, *limit)
	if err != nil {
		log.Fatal(err)
	}
	if *dataset != "" {
		e, err := newEnricher(*dataset, *enrichMode)
		if err != nil {
			log.Fatal(err)
		}
		for i := range fixtures {
			fixtures[i].Home = e.lineup(fixtures[i].Home)
			fixtures[i].Away = e.lineup(fixtures[i].Away)
		}
		fmt.Printf("enriched from %s (%s): %d players matched, %d left as archived\n", filepath.Base(*dataset), *enrichMode, e.matched, e.unmatched)
	}

	newTally := func() *tally {
		return &tally{
			shotsByType:     map[soccer.ChanceType]int{},
			goalsByType:     map[soccer.ChanceType]int{},
			goalsByPosition: map[soccer.PlayerPosition]int{},
			missDetail:      map[string]int{},
			goalDetail:      map[string]int{},
			special:         map[string]int{},
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
		bucket := gapBucket(gap)
		var homeWins, awayWins int
		for s := 0; s < *seeds; s++ {
			src := rand.New(rand.NewSource(seedFor(f.Name, s)))
			outcome, err := soccer.RunGameWithModel(src, f.Home, f.Away, soccer.Model(*model))
			if err != nil {
				log.Fatalf("%s: %v", f.Name, err)
			}
			stats := soccer.CreateGameStats(outcome.Events)
			switch {
			case stats.HomeTeamStats.Goals > stats.AwayTeamStats.Goals:
				homeWins++
			case stats.HomeTeamStats.Goals < stats.AwayTeamStats.Goals:
				awayWins++
			}
			for _, t := range buckets {
				t.add(outcome, positions, strongerHome, bucket)
			}
		}
		signedGap := lineupStrength(f.Home) - lineupStrength(f.Away)
		pointsDiff := 3 * float64(homeWins-awayWins) / float64(*seeds)
		for _, t := range buckets {
			t.likelierWins += max(homeWins, awayWins)
			t.likelierMatches += *seeds
			t.gapN++
			t.gapX += signedGap
			t.gapY += pointsDiff
			t.gapXX += signedGap * signedGap
			t.gapYY += pointsDiff * pointsDiff
			t.gapXY += signedGap * pointsDiff
		}
	}
	report(*all, len(fixtures), *seeds, *model, "all fixtures")
	report(*close, closeFixtures, *seeds, *model, "close fixtures (mean overall within 5)")
	report(*mismatch, mismatchFixtures, *seeds, *model, "mismatches (mean overall 8+ apart)")
}

func (t *tally) add(outcome soccer.MatchOutcome, positions map[string]soccer.PlayerPosition, strongerHome bool, bucket int) {
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
					if d := e.GetGoalEvent().Detail; d != "" {
						t.goalDetail[d]++
					}
					if e.ChanceType == soccer.ChanceTypeCross || e.ChanceType == soccer.ChanceTypeCorner {
						t.headed++
					}
				}
				t.shots++
				t.shotsByType[e.ChanceType]++
				if len(e.Steps) > 0 {
					switch first := e.Steps[0]; {
					case first.Kind == soccer.StepKindRebound:
						t.special["rebound off "+first.Detail]++
					case first.Kind == soccer.StepKindTakeOn && first.Detail == "solo":
						t.special["solo"]++
					case first.Kind == soccer.StepKindPass && first.Detail == "high":
						t.special["long ball"]++
						if e.Type == soccer.GameEventTypeGoal {
							t.special["long ball goal"]++
						}
					}
					for _, s := range e.Steps {
						if s.Kind == soccer.StepKindTakeOn && s.Detail == "cut_in" {
							t.special["cut in"]++
							break
						}
					}
					if last := e.Steps[len(e.Steps)-1]; positions[last.ActorID] == soccer.PlayerPositionGoalkeeper {
						t.special["keeper"]++
					}
				}
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
			t.favourite[bucket].matches++
			switch {
			case home == away:
				t.favourite[bucket].draws++
			case (home > away) == strongerHome:
				t.favourite[bucket].wins++
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

// gapCorrelation is Pearson's r between fixtures' rating gaps and their
// average points difference.
func (t tally) gapCorrelation() float64 {
	n := t.gapN
	if n == 0 {
		return 0
	}
	cov := t.gapXY/n - (t.gapX/n)*(t.gapY/n)
	vx := t.gapXX/n - (t.gapX/n)*(t.gapX/n)
	vy := t.gapYY/n - (t.gapY/n)*(t.gapY/n)
	if vx <= 0 || vy <= 0 {
		return 0
	}
	return cov / math.Sqrt(vx*vy)
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
		{"Likelier winner wins", fmt.Sprintf("%.1f%%", 100*float64(t.likelierWins)/float64(max(t.likelierMatches, 1))), "not below the live model"},
		{"Rating gap v result (correlation)", fmt.Sprintf("%.3f", t.gapCorrelation()), "not below the live model"},
	}
	fmt.Printf("%-36s %10s   %s\n", "metric", "value", "target")
	for _, r := range rows {
		fmt.Printf("%-36s %10s   %s\n", r[0], r[1], r[2])
	}

	fmt.Printf("\nthe stronger side, by how much stronger (mean overall):\n%-12s %8s %8s %8s %8s %10s\n", "gap", "matches", "wins", "draws", "loses", "points/g")
	var fm, fw, fd int
	for i, lo := range gapBuckets {
		b := t.favourite[i]
		if b.matches == 0 {
			continue
		}
		name := fmt.Sprintf("%.0f+", lo)
		if i+1 < len(gapBuckets) {
			name = fmt.Sprintf("%.0f to %.0f", lo, gapBuckets[i+1])
		}
		n := float64(b.matches)
		fmt.Printf("%-12s %8d %7.1f%% %7.1f%% %7.1f%% %10.2f\n", name, b.matches, 100*float64(b.wins)/n, 100*float64(b.draws)/n,
			100*float64(b.matches-b.wins-b.draws)/n, float64(3*b.wins+b.draws)/n)
		fm, fw, fd = fm+b.matches, fw+b.wins, fd+b.draws
	}
	fmt.Printf("%-12s %8d %7.1f%% %7.1f%% %7.1f%% %10.2f\n", "all", fm, 100*float64(fw)/float64(fm), 100*float64(fd)/float64(fm),
		100*float64(fm-fw-fd)/float64(fm), float64(3*fw+fd)/float64(fm))

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
		per100 := func(v int) string { return fmt.Sprintf("%.1f", 100*float64(v)/n) }
		fmt.Printf("per 100 matches, goals: worldie %s  solo %s  rebound %s  keeper %s\n",
			per100(t.goalDetail["worldie"]), per100(t.goalDetail["solo"]), per100(t.goalDetail["rebound"]), per100(t.goalDetail["keeper"]))
		fmt.Printf("per 100 matches, shots: after a solo run %s  a wide player cutting inside %s  rebound off a save %s, the post %s, the bar %s  by a keeper %s\n",
			per100(t.special["solo"]), per100(t.special["cut in"]), per100(t.special["rebound off save"]), per100(t.special["rebound off post"]),
			per100(t.special["rebound off bar"]), per100(t.special["keeper"]))
		fmt.Printf("per 100 matches, long balls a forward won that ended in a shot %s, and in a goal %s\n",
			per100(t.special["long ball"]), per100(t.special["long ball goal"]))
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
