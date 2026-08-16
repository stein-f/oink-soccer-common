// Command analyse_specialists is the Tier Specialist calibration study.
//
// It runs one realistic mixed-season allocation (s16 tier counts plus 3000
// schnoz at Tier Specialist), samples 5-a-side teams per tier fielding every
// player at their natural position, and plays them against each other with
// the real v2 engine under neutral tactics (Diamond formation, no roles, no
// boosts). Everything is seeded — same flags, same output.
//
// The calibration gate: an all-Specialist XI must sit between Tier B and
// Tier A team strength. The chance-type tables show the specialist
// fingerprint (where schnoz goals come from vs a quality-adjacent tier).
//
// Run from the v2 module directory:
//
//	cd v2 && go run ./cmd/analyse_specialists
package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"sort"

	soccer "github.com/stein-f/oink-soccer-common/v2"
	"github.com/stein-f/oink-soccer-common/v2/allocation"
	"github.com/stein-f/oink-soccer-common/v2/allocation/fifacsv"
)

// Season composition mirrors s16's real eligible-asset counts plus the 3000
// schnoz joining at Tier Specialist.
var seasonComposition = []struct {
	tier  allocation.AssetTier
	count int
}{
	{allocation.AssetTierS, 454},
	{allocation.AssetTierA, 841},
	{allocation.AssetTierB, 851},
	{allocation.AssetTierC, 10248},
	{allocation.AssetTierAggressive, 999},
	{allocation.AssetTierSpecialist, 3000},
}

func main() {
	root := flag.String("root", "", "repo root containing cmd/allocation (auto-detected when empty)")
	games := flag.Int("games", 4000, "games per matchup")
	seed := flag.Int64("seed", 20260816, "master seed for the whole study")
	specialistMin := flag.Int("specialist-min", 0, "override the specialist filter minimum for every position (0 = rules default)")
	flag.Parse()

	dataRoot, err := resolveDataRoot(*root)
	if err != nil {
		log.Fatal(err)
	}

	rng := rand.New(rand.NewSource(*seed))

	candidates, err := fifacsv.LoadCandidates(filepath.Join(dataRoot, "cmd/allocation/fifa_players_22.csv"), rng)
	if err != nil {
		log.Fatalf("load candidates: %v", err)
	}
	log.Printf("loaded %d candidates", len(candidates))

	rules := allocation.DefaultRules()
	if *specialistMin > 0 {
		for pos, f := range rules.SpecialistFilters {
			f.Minimum = *specialistMin
			rules.SpecialistFilters[pos] = f
		}
		log.Printf("specialist filter minimum overridden to %d", *specialistMin)
	}
	pool := allocation.NewPool(candidates, rules)

	var assets []allocation.Asset
	for _, c := range seasonComposition {
		for i := 0; i < c.count; i++ {
			assets = append(assets, allocation.Asset{
				ID:   fmt.Sprintf("%s-%d", c.tier, i),
				Name: fmt.Sprintf("%s-%d", c.tier, i),
				Tier: c.tier,
			})
		}
	}
	assignments, err := allocation.Allocate(rng, pool, assets)
	if err != nil {
		log.Fatalf("allocate: %v", err)
	}
	log.Printf("season allocated: %d assets", len(assignments))

	rosters := buildRosters(assignments)
	for _, tier := range []allocation.AssetTier{
		allocation.AssetTierS, allocation.AssetTierA, allocation.AssetTierB,
		allocation.AssetTierC, allocation.AssetTierAggressive, allocation.AssetTierSpecialist,
	} {
		r := rosters[tier]
		log.Printf("%-16s GK %-4d DEF %-4d MID %-4d ATK %-4d", tier,
			len(r[soccer.PlayerPositionGoalkeeper]), len(r[soccer.PlayerPositionDefense]),
			len(r[soccer.PlayerPositionMidfield]), len(r[soccer.PlayerPositionAttack]))
	}
	// The aggressive pool contains no keepers (aggression ≥ 80 keepers don't
	// exist in the dataset), so schizo XIs borrow Tier C keepers — the
	// realistic squad-building outcome for schizo-heavy teams.
	rosters[allocation.AssetTierAggressive][soccer.PlayerPositionGoalkeeper] =
		rosters[allocation.AssetTierC][soccer.PlayerPositionGoalkeeper]

	matchups := []struct {
		a, b allocation.AssetTier
	}{
		{allocation.AssetTierSpecialist, allocation.AssetTierS},
		{allocation.AssetTierSpecialist, allocation.AssetTierA},
		{allocation.AssetTierSpecialist, allocation.AssetTierB},
		{allocation.AssetTierSpecialist, allocation.AssetTierC},
		{allocation.AssetTierSpecialist, allocation.AssetTierAggressive},
		// Ladder controls: where do the established tiers sit vs each other?
		{allocation.AssetTierS, allocation.AssetTierA},
		{allocation.AssetTierA, allocation.AssetTierB},
		{allocation.AssetTierB, allocation.AssetTierC},
		{allocation.AssetTierA, allocation.AssetTierC},
	}

	fmt.Printf("\n== matchups (%d games each, Diamond, neutral tactics, players at natural positions) ==\n", *games)
	fmt.Printf("%-34s %8s %8s %8s %10s %10s\n", "matchup", "winA%", "draw%", "winB%", "goalsA/g", "goalsB/g")
	results := map[allocation.AssetTier]map[allocation.AssetTier]float64{}
	fingerprints := map[allocation.AssetTier]*chanceStats{}
	for _, m := range matchups {
		res := playMatchup(rng, rosters, m.a, m.b, *games, fingerprints)
		if results[m.a] == nil {
			results[m.a] = map[allocation.AssetTier]float64{}
		}
		results[m.a][m.b] = res.winA
		fmt.Printf("%-34s %7.1f%% %7.1f%% %7.1f%% %10.2f %10.2f\n",
			fmt.Sprintf("%s vs %s", m.a, m.b),
			100*res.winA, 100*res.draw, 100*res.winB, res.goalsA, res.goalsB)
	}

	fmt.Printf("\n== calibration gate: Tier B ≤ Specialist ≤ Tier A ==\n")
	specVsC := results[allocation.AssetTierSpecialist][allocation.AssetTierC]
	aVsC := results[allocation.AssetTierA][allocation.AssetTierC]
	bVsC := results[allocation.AssetTierB][allocation.AssetTierC]
	fmt.Printf("win rate vs Tier C:  Tier B %.1f%%  <  Specialist %.1f%%  <  Tier A %.1f%%   gate: %s\n",
		100*bVsC, 100*specVsC, 100*aVsC, gateVerdict(bVsC <= specVsC && specVsC <= aVsC))
	specVsA := results[allocation.AssetTierSpecialist][allocation.AssetTierA]
	specVsB := results[allocation.AssetTierSpecialist][allocation.AssetTierB]
	fmt.Printf("head to head:        vs Tier A %.1f%% (want <50)   vs Tier B %.1f%% (want >50)   gate: %s\n",
		100*specVsA, 100*specVsB, gateVerdict(specVsA < 0.5 && specVsB > 0.5))

	fmt.Printf("\n== goal fingerprint by chance type (share of team goals | conversion) ==\n")
	printFingerprint("Specialist", fingerprints[allocation.AssetTierSpecialist])
	printFingerprint("Tier A", fingerprints[allocation.AssetTierA])
	printFingerprint("Tier B", fingerprints[allocation.AssetTierB])

	fmt.Printf("\n== marginal value: swap ONE player of a team for a Specialist (win%% of hybrid vs pure) ==\n")
	for _, h := range []struct {
		base allocation.AssetTier
		pos  soccer.PlayerPosition
	}{
		{allocation.AssetTierA, soccer.PlayerPositionMidfield},
		{allocation.AssetTierA, soccer.PlayerPositionAttack},
		{allocation.AssetTierA, soccer.PlayerPositionDefense},
		{allocation.AssetTierB, soccer.PlayerPositionMidfield},
		{allocation.AssetTierS, soccer.PlayerPositionMidfield},
	} {
		win, draw := playHybrid(rng, rosters, h.base, h.pos, *games)
		fmt.Printf("%s + Specialist %-10s vs pure %-8s   win %5.1f%%  draw %5.1f%%  loss %5.1f%%\n",
			h.base, h.pos, h.base, 100*win, 100*draw, 100*(1-win-draw))
	}
}

// playHybrid pits a base-tier team with one position swapped for a
// Specialist against a pure base-tier team. A win rate above the tier's
// mirror-match baseline (~= (1-draw)/2) means the schnoz slot is an
// upgrade managers will want.
func playHybrid(rng *rand.Rand, rosters map[allocation.AssetTier]rosterIndex, base allocation.AssetTier, swapPos soccer.PlayerPosition, games int) (winRate, drawRate float64) {
	var wins, draws int
	for g := 0; g < games; g++ {
		hybrid := sampleTeam(rng, rosters[base], "H")
		pure := sampleTeam(rng, rosters[base], "P")
		// Replace the first player at swapPos with a Specialist of the same
		// position (distinct from the rest by asset id construction).
		spec := rosters[allocation.AssetTierSpecialist][swapPos]
		pick := spec[rng.Intn(len(spec))]
		for i, p := range hybrid.Players {
			if p.SelectedPosition == swapPos {
				hybrid.Players[i] = soccer.SelectedPlayer{
					ID:               pick.Asset.ID,
					Name:             pick.Player.Name,
					Attributes:       pick.Player.Attributes,
					SelectedPosition: swapPos,
				}
				break
			}
		}
		home, away := hybrid, pure
		hybridIsHome := g%2 == 0
		if !hybridIsHome {
			home, away = pure, hybrid
		}
		gameRng := rand.New(rand.NewSource(rng.Int63()))
		events, _, err := soccer.RunGameWithSeed(gameRng, home, away)
		if err != nil {
			log.Fatalf("hybrid game failed: %v", err)
		}
		var homeGoals, awayGoals int
		for _, e := range events {
			if !e.IsGoal() {
				continue
			}
			if e.GetGoalEvent().TeamType == soccer.TeamTypeHome {
				homeGoals++
			} else {
				awayGoals++
			}
		}
		hybridGoals, pureGoals := homeGoals, awayGoals
		if !hybridIsHome {
			hybridGoals, pureGoals = awayGoals, homeGoals
		}
		switch {
		case hybridGoals > pureGoals:
			wins++
		case hybridGoals == pureGoals:
			draws++
		}
	}
	return float64(wins) / float64(games), float64(draws) / float64(games)
}

func gateVerdict(ok bool) string {
	if ok {
		return "PASS"
	}
	return "FAIL"
}

// rosterIndex maps position → the tier's allocated players available there.
type rosterIndex map[soccer.PlayerPosition][]allocation.Assignment

func buildRosters(assignments []allocation.Assignment) map[allocation.AssetTier]rosterIndex {
	out := map[allocation.AssetTier]rosterIndex{}
	for _, a := range assignments {
		tier := a.Asset.Tier
		if out[tier] == nil {
			out[tier] = rosterIndex{}
		}
		// Field players at their natural position (what managers do), not the
		// rolled asset position — matters for the position-blind aggressive tier.
		pos := a.Player.Attributes.PrimaryPosition
		out[tier][pos] = append(out[tier][pos], a)
	}
	return out
}

// sampleTeam builds a Diamond lineup (GK, DEF, MID, MID, ATK) from a tier's
// roster with distinct assets.
func sampleTeam(rng *rand.Rand, roster rosterIndex, teamID string) soccer.GameLineup {
	slots := []soccer.PlayerPosition{
		soccer.PlayerPositionGoalkeeper,
		soccer.PlayerPositionDefense,
		soccer.PlayerPositionMidfield,
		soccer.PlayerPositionMidfield,
		soccer.PlayerPositionAttack,
	}
	used := map[string]bool{}
	players := make([]soccer.SelectedPlayer, 0, len(slots))
	for _, pos := range slots {
		bucket := roster[pos]
		var pick allocation.Assignment
		for {
			pick = bucket[rng.Intn(len(bucket))]
			if !used[pick.Asset.ID] {
				break
			}
		}
		used[pick.Asset.ID] = true
		players = append(players, soccer.SelectedPlayer{
			ID:               pick.Asset.ID,
			Name:             pick.Player.Name,
			Attributes:       pick.Player.Attributes,
			SelectedPosition: pos,
		})
	}
	return soccer.GameLineup{
		Team:    soccer.Team{ID: teamID, Formation: soccer.FormationTypeDiamond},
		Players: players,
	}
}

type matchupResult struct {
	winA, draw, winB float64
	goalsA, goalsB   float64
}

// chanceStats accumulates per-chance-type shots and goals for one tier
// across every matchup it plays in.
type chanceStats struct {
	shots map[soccer.ChanceType]int
	goals map[soccer.ChanceType]int
}

func newChanceStats() *chanceStats {
	return &chanceStats{shots: map[soccer.ChanceType]int{}, goals: map[soccer.ChanceType]int{}}
}

func playMatchup(rng *rand.Rand, rosters map[allocation.AssetTier]rosterIndex, tierA, tierB allocation.AssetTier, games int, fingerprints map[allocation.AssetTier]*chanceStats) matchupResult {
	if fingerprints[tierA] == nil {
		fingerprints[tierA] = newChanceStats()
	}
	if fingerprints[tierB] == nil {
		fingerprints[tierB] = newChanceStats()
	}
	var res matchupResult
	var goalsA, goalsB int
	for g := 0; g < games; g++ {
		teamA := sampleTeam(rng, rosters[tierA], "A")
		teamB := sampleTeam(rng, rosters[tierB], "B")
		// Alternate sides so any structural home/away asymmetry cancels.
		home, away := teamA, teamB
		aIsHome := g%2 == 0
		if !aIsHome {
			home, away = teamB, teamA
		}
		gameRng := rand.New(rand.NewSource(rng.Int63()))
		events, _, err := soccer.RunGameWithSeed(gameRng, home, away)
		if err != nil {
			log.Fatalf("game failed: %v", err)
		}
		var homeGoals, awayGoals int
		for _, e := range events {
			isHomeEvent := teamTypeOf(e) == soccer.TeamTypeHome
			eventTier := tierA
			if isHomeEvent != aIsHome {
				eventTier = tierB
			}
			fp := fingerprints[eventTier]
			fp.shots[e.ChanceType]++
			if e.IsGoal() {
				fp.goals[e.ChanceType]++
				if isHomeEvent {
					homeGoals++
				} else {
					awayGoals++
				}
			}
		}
		aGoals, bGoals := homeGoals, awayGoals
		if !aIsHome {
			aGoals, bGoals = awayGoals, homeGoals
		}
		goalsA += aGoals
		goalsB += bGoals
		switch {
		case aGoals > bGoals:
			res.winA++
		case bGoals > aGoals:
			res.winB++
		default:
			res.draw++
		}
	}
	n := float64(games)
	res.winA /= n
	res.draw /= n
	res.winB /= n
	res.goalsA = float64(goalsA) / n
	res.goalsB = float64(goalsB) / n
	return res
}

func teamTypeOf(e soccer.GameEvent) soccer.TeamType {
	if e.IsGoal() {
		return e.GetGoalEvent().TeamType
	}
	return e.GetMissEvent().TeamType
}

func printFingerprint(label string, cs *chanceStats) {
	if cs == nil {
		return
	}
	var totalGoals int
	types := make([]soccer.ChanceType, 0, len(cs.shots))
	for t := range cs.shots {
		types = append(types, t)
		totalGoals += cs.goals[t]
	}
	sort.Slice(types, func(i, j int) bool { return types[i] < types[j] })
	fmt.Printf("%s (total goals %d):\n", label, totalGoals)
	for _, t := range types {
		share := 0.0
		if totalGoals > 0 {
			share = 100 * float64(cs.goals[t]) / float64(totalGoals)
		}
		conv := 0.0
		if cs.shots[t] > 0 {
			conv = 100 * float64(cs.goals[t]) / float64(cs.shots[t])
		}
		fmt.Printf("  %-16s share %5.1f%%   conversion %5.1f%%  (%d/%d)\n", t, share, conv, cs.goals[t], cs.shots[t])
	}
}

// resolveDataRoot mirrors cmd/allocation: explicit -root, else walk up to the
// checkout containing the shared dataset.
func resolveDataRoot(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "cmd/allocation/config.json")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not locate cmd/allocation/config.json above %s", cwd)
		}
		dir = parent
	}
}
