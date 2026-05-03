// snapshot generates JSON snapshots of v2 engine output for a pinned set
// of (seed, formation, lineup) inputs. Snapshots live in
// v2/testdata/golden/v2/ and are loaded by golden_test.go to detect any
// unintended drift in engine behavior.
//
// Run from the v2 module:
//
//	go run ./cmd/snapshot
package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"

	soccer "github.com/stein-f/oink-soccer-common/v2"
	"github.com/stein-f/oink-soccer-common/v2/testdata"
)

const outDir = "testdata/golden/v2"

type Snapshot struct {
	Input    Input            `json:"input"`
	Events   []Event          `json:"events"`
	Injuries InjuriesSummary  `json:"injuries"`
	Stats    soccer.GameStats `json:"stats"`
}

type Input struct {
	Seed          int64                `json:"seed"`
	HomeFormation soccer.FormationType `json:"home_formation"`
	AwayFormation soccer.FormationType `json:"away_formation"`
	HomeTag       string               `json:"home_tag"`
	AwayTag       string               `json:"away_tag"`
}

type Event struct {
	Type       soccer.GameEventType `json:"type"`
	ChanceType soccer.ChanceType    `json:"chance_type"`
	Minute     int                  `json:"minute"`
	PlayerID   string               `json:"player_id"`
	TeamType   soccer.TeamType      `json:"team_type"`
}

type InjuriesSummary struct {
	Home []InjurySummary `json:"home"`
	Away []InjurySummary `json:"away"`
}

type InjurySummary struct {
	PlayerID     string                `json:"player_id"`
	Severity     soccer.InjurySeverity `json:"severity"`
	Name         string                `json:"name"`
	DurationDays int                   `json:"duration_days"`
}

func main() {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fail("mkdir: %v", err)
	}

	formations := []soccer.FormationType{
		soccer.FormationTypePyramid,
		soccer.FormationTypeDiamond,
		soccer.FormationTypeY,
		soccer.FormationTypeBox,
	}

	count := 0
	// 16 snapshots: every (home, away) formation pair, StrongTeam vs StrongTeam.
	for _, h := range formations {
		for _, a := range formations {
			snap := generate(42, h, a, "strong", "strong",
				testdata.StrongTeam(h), testdata.StrongTeam(a))
			if err := write(snap); err != nil {
				fail("write: %v", err)
			}
			count++
		}
	}
	// 4 snapshots: StrongTeam vs WeakTeam, Diamond vs Diamond, varying seed.
	for _, seed := range []int64{1, 7, 99, 12345} {
		snap := generate(seed,
			soccer.FormationTypeDiamond, soccer.FormationTypeDiamond,
			"strong", "weak",
			testdata.StrongTeam(soccer.FormationTypeDiamond),
			testdata.WeakTeam(soccer.FormationTypeDiamond))
		if err := write(snap); err != nil {
			fail("write: %v", err)
		}
		count++
	}

	fmt.Printf("wrote %d snapshots to %s\n", count, outDir)
}

func generate(seed int64, homeF, awayF soccer.FormationType, homeTag, awayTag string,
	home, away soccer.GameLineup) Snapshot {

	src := rand.New(rand.NewSource(seed))
	events, injuries, err := soccer.RunGameWithSeed(src, home, away)
	if err != nil {
		fail("run game seed=%d: %v", seed, err)
	}

	out := Snapshot{
		Input: Input{
			Seed: seed, HomeFormation: homeF, AwayFormation: awayF,
			HomeTag: homeTag, AwayTag: awayTag,
		},
		Stats: soccer.CreateGameStats(events),
	}
	for _, e := range events {
		ev := Event{Type: e.Type, ChanceType: e.ChanceType, Minute: e.Minute}
		switch e.Type {
		case soccer.GameEventTypeGoal:
			g := e.GetGoalEvent()
			ev.PlayerID, ev.TeamType = g.PlayerID, g.TeamType
		case soccer.GameEventTypeMiss:
			m := e.GetMissEvent()
			ev.PlayerID, ev.TeamType = m.PlayerID, m.TeamType
		}
		out.Events = append(out.Events, ev)
	}
	out.Injuries = InjuriesSummary{
		Home: summarise(injuries.HomeTeamInjuries),
		Away: summarise(injuries.AwayTeamInjuries),
	}
	return out
}

func summarise(events []soccer.InjuryEvent) []InjurySummary {
	if len(events) == 0 {
		return nil
	}
	out := make([]InjurySummary, 0, len(events))
	for _, e := range events {
		out = append(out, InjurySummary{
			PlayerID:     e.PlayerID,
			Severity:     e.Injury.Severity,
			Name:         e.Injury.Name,
			DurationDays: e.DurationDays,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PlayerID < out[j].PlayerID })
	return out
}

func write(s Snapshot) error {
	name := fmt.Sprintf("seed-%d_home-%s_away-%s_%s-vs-%s.json",
		s.Input.Seed,
		slug(string(s.Input.HomeFormation)),
		slug(string(s.Input.AwayFormation)),
		s.Input.HomeTag, s.Input.AwayTag,
	)
	body, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, name), append(body, '\n'), 0o644)
}

func slug(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == ' ':
			out = append(out, '-')
		case c >= 'A' && c <= 'Z':
			out = append(out, c+('a'-'A'))
		default:
			out = append(out, c)
		}
	}
	return string(out)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "snapshot: "+format+"\n", args...)
	os.Exit(1)
}
