// snapshot-v1 generates JSON baseline snapshots of v1 engine output for a
// pinned set of (seed, formation, lineup) inputs. The snapshots are
// committed to v2/testdata/golden/v1-baseline/ as a reference for how
// far v2's behavior diverges from v1 as the rebuild progresses.
//
// Run from the repo root:
//
//	go run ./cmd/snapshot-v1
package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"

	soccer "github.com/stein-f/oink-soccer-common"
	"github.com/stein-f/oink-soccer-common/testdata"
)

const outDir = "v2/testdata/golden/v1-baseline"

// Snapshot is the persisted shape. Injury.Expires is stripped because v1
// populates it with time.Now() which is non-deterministic — the injury
// identity (name, severity, days) is fully seeded.
type Snapshot struct {
	Input  Input    `json:"input"`
	Events []Event  `json:"events"`
	Injury Injuries `json:"injuries"`
}

type Input struct {
	Seed           int64                `json:"seed"`
	HomeFormation  soccer.FormationType `json:"home_formation"`
	AwayFormation  soccer.FormationType `json:"away_formation"`
	HomePlayersTag string               `json:"home_players_tag"`
	AwayPlayersTag string               `json:"away_players_tag"`
}

type Event struct {
	Type     soccer.GameEventType `json:"type"`
	Minute   int                  `json:"minute"`
	PlayerID string               `json:"player_id"`
	TeamType soccer.TeamType      `json:"team_type"`
}

type Injuries struct {
	Home []InjurySummary `json:"home"`
	Away []InjurySummary `json:"away"`
}

type InjurySummary struct {
	PlayerID string                `json:"player_id"`
	Severity soccer.InjurySeverity `json:"severity"`
	Name     string                `json:"name"`
	MinDays  int                   `json:"min_days"`
	MaxDays  int                   `json:"max_days"`
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
	// 16 snapshots: every (home, away) formation pair with StrongTeam vs StrongTeam, fixed seed.
	for _, h := range formations {
		for _, a := range formations {
			snap := generate(int64(42), h, a, "strong", "strong",
				testdata.StrongTeam(), testdata.StrongTeam())
			if err := write(snap); err != nil {
				fail("write: %v", err)
			}
			count++
		}
	}

	// 4 snapshots: StrongTeam vs WeakTeam across different seeds, Diamond v Diamond.
	for _, seed := range []int64{1, 7, 99, 12345} {
		snap := generate(seed, soccer.FormationTypeDiamond, soccer.FormationTypeDiamond,
			"strong", "weak",
			testdata.StrongTeam(), testdata.WeakTeamPlayers)
		if err := write(snap); err != nil {
			fail("write: %v", err)
		}
		count++
	}

	fmt.Printf("wrote %d snapshots to %s\n", count, outDir)
}

func generate(seed int64, homeF, awayF soccer.FormationType, homeTag, awayTag string,
	homePlayers, awayPlayers []soccer.SelectedPlayer) Snapshot {

	src := rand.New(rand.NewSource(seed))
	home := soccer.GameLineup{Team: soccer.Team{ID: "home", Formation: homeF}, Players: homePlayers}
	away := soccer.GameLineup{Team: soccer.Team{ID: "away", Formation: awayF}, Players: awayPlayers}

	events, injuries, err := soccer.RunGameWithSeed(src, home, away)
	if err != nil {
		fail("run game seed=%d: %v", seed, err)
	}

	out := Snapshot{
		Input: Input{
			Seed:           seed,
			HomeFormation:  homeF,
			AwayFormation:  awayF,
			HomePlayersTag: homeTag,
			AwayPlayersTag: awayTag,
		},
	}
	for _, e := range events {
		ev := Event{Type: e.Type, Minute: e.Minute}
		switch e.Type {
		case soccer.GameEventTypeGoal:
			g := e.GetGoalEvent()
			ev.PlayerID = g.PlayerID
			ev.TeamType = g.TeamType
		case soccer.GameEventTypeMiss:
			m := e.GetMissEvent()
			ev.PlayerID = m.PlayerID
			ev.TeamType = m.TeamType
		}
		out.Events = append(out.Events, ev)
	}
	out.Injury = Injuries{
		Home: summariseInjuries(injuries.HomeTeamInjuries),
		Away: summariseInjuries(injuries.AwayTeamInjuries),
	}
	return out
}

func summariseInjuries(events []soccer.InjuryEvent) []InjurySummary {
	if len(events) == 0 {
		return nil
	}
	out := make([]InjurySummary, 0, len(events))
	for _, e := range events {
		out = append(out, InjurySummary{
			PlayerID: e.PlayerID,
			Severity: e.Injury.Severity,
			Name:     e.Injury.Name,
			MinDays:  e.Injury.MinDays,
			MaxDays:  e.Injury.MaxDays,
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
		s.Input.HomePlayersTag,
		s.Input.AwayPlayersTag,
	)
	path := filepath.Join(outDir, name)
	body, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(body, '\n'), 0o644)
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
	fmt.Fprintf(os.Stderr, "snapshot-v1: "+format+"\n", args...)
	os.Exit(1)
}
