package soccer_test

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"testing"

	soccer "github.com/stein-f/oink-soccer-common/v2"
	"github.com/stein-f/oink-soccer-common/v2/testdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoldenSnapshots replays every committed v2 snapshot through the
// engine and asserts the output matches. If a code change is intentional,
// regenerate the snapshots:
//
//	go run ./cmd/snapshot
//
// and explain the diff in the PR description (see
// testdata/golden/README.md).
//
// This is the primary regression guard — any unintended change to engine
// math will break it.
func TestGoldenSnapshots(t *testing.T) {
	files, err := filepath.Glob("testdata/golden/v2/*.json")
	require.NoError(t, err)
	require.NotEmpty(t, files, "no v2 golden snapshots — run: go run ./cmd/snapshot")

	for _, f := range files {
		t.Run(filepath.Base(f), func(t *testing.T) {
			body, err := os.ReadFile(f)
			require.NoError(t, err)

			var want goldenSnapshot
			require.NoError(t, json.Unmarshal(body, &want))

			home, away := lineupsForSnapshot(t, want.Input)
			events, injuries, err := soccer.RunGameWithSeed(rand.New(rand.NewSource(want.Input.Seed)), home, away)
			require.NoError(t, err)

			got := goldenSnapshot{
				Input:    want.Input,
				Events:   eventsToGolden(events),
				Injuries: injuriesToGolden(injuries),
				Stats:    soccer.CreateGameStats(events),
			}

			assert.Equal(t, want, got, "snapshot %s drifted — regenerate with `go run ./cmd/snapshot` if intentional", filepath.Base(f))
		})
	}
}

func lineupsForSnapshot(t *testing.T, in goldenInput) (soccer.GameLineup, soccer.GameLineup) {
	t.Helper()
	home := lineupFor(t, in.HomeTag, in.HomeFormation)
	away := lineupFor(t, in.AwayTag, in.AwayFormation)
	return home, away
}

func lineupFor(t *testing.T, tag string, f soccer.FormationType) soccer.GameLineup {
	t.Helper()
	switch tag {
	case "strong":
		return testdata.StrongTeam(f)
	case "weak":
		return testdata.WeakTeam(f)
	default:
		t.Fatalf("unknown lineup tag %q", tag)
		return soccer.GameLineup{}
	}
}

type goldenSnapshot struct {
	Input    goldenInput      `json:"input"`
	Events   []goldenEvent    `json:"events"`
	Injuries goldenInjuries   `json:"injuries"`
	Stats    soccer.GameStats `json:"stats"`
}

type goldenInput struct {
	Seed          int64                `json:"seed"`
	HomeFormation soccer.FormationType `json:"home_formation"`
	AwayFormation soccer.FormationType `json:"away_formation"`
	HomeTag       string               `json:"home_tag"`
	AwayTag       string               `json:"away_tag"`
}

type goldenEvent struct {
	Type       soccer.GameEventType `json:"type"`
	ChanceType soccer.ChanceType    `json:"chance_type"`
	Minute     int                  `json:"minute"`
	PlayerID   string               `json:"player_id"`
	TeamType   soccer.TeamType      `json:"team_type"`
}

type goldenInjuries struct {
	Home []goldenInjury `json:"home"`
	Away []goldenInjury `json:"away"`
}

type goldenInjury struct {
	PlayerID     string                `json:"player_id"`
	Severity     soccer.InjurySeverity `json:"severity"`
	Name         string                `json:"name"`
	DurationDays int                   `json:"duration_days"`
}

func eventsToGolden(events []soccer.GameEvent) []goldenEvent {
	if len(events) == 0 {
		return nil
	}
	out := make([]goldenEvent, 0, len(events))
	for _, e := range events {
		ev := goldenEvent{Type: e.Type, ChanceType: e.ChanceType, Minute: e.Minute}
		switch e.Type {
		case soccer.GameEventTypeGoal:
			g := e.GetGoalEvent()
			ev.PlayerID, ev.TeamType = g.PlayerID, g.TeamType
		case soccer.GameEventTypeMiss:
			m := e.GetMissEvent()
			ev.PlayerID, ev.TeamType = m.PlayerID, m.TeamType
		}
		out = append(out, ev)
	}
	return out
}

func injuriesToGolden(in soccer.Injuries) goldenInjuries {
	return goldenInjuries{
		Home: injuryListToGolden(in.HomeTeamInjuries),
		Away: injuryListToGolden(in.AwayTeamInjuries),
	}
}

func injuryListToGolden(events []soccer.InjuryEvent) []goldenInjury {
	if len(events) == 0 {
		return nil
	}
	out := make([]goldenInjury, 0, len(events))
	for _, e := range events {
		out = append(out, goldenInjury{
			PlayerID:     e.PlayerID,
			Severity:     e.Injury.Severity,
			Name:         e.Injury.Name,
			DurationDays: e.DurationDays,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PlayerID < out[j].PlayerID })
	return out
}
