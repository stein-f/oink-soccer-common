package soccer_test

import (
	"math/rand"
	"testing"

	soccer "github.com/stein-f/oink-soccer-common/v2"
)

// Skill-gap probe. Pits uniform-rating squads against each other across a
// matrix of overall ratings to measure how much rating differential moves
// win rates. Used to decide whether v2 needs to bring back v1's non-linear
// scaling curve.
//
// The test is informational: it always prints a table, never asserts on
// absolute win-rate targets (those are a design call to be made after
// looking at the numbers). Skipped under -short because it runs thousands
// of games.

const skillGapTrials = 2000

func TestSkillGap_WinRates(t *testing.T) {
	if testing.Short() {
		t.Skip("skill-gap probe is heavy; rerun without -short")
	}

	// Each row is a matchup we care about. Ratings are applied uniformly
	// to every attribute on every player in the squad (5 players, Diamond
	// formation both sides).
	type pair struct {
		label      string
		home, away int
	}
	pairs := []pair{
		// Equal teams — sanity check, expect ~50/50 with draws.
		{"equal 90 vs 90", 90, 90},
		{"equal 85 vs 85", 85, 85},
		{"equal 80 vs 80", 80, 80},
		{"equal 70 vs 70", 70, 70},

		// Small gap — the v1 worry was that 80 vs 84 felt like a coin flip.
		{"narrow 84 vs 80", 84, 80},
		{"narrow 82 vs 78", 82, 78},

		// Mid gap — England vs Wales territory.
		{"England-ish 87 vs Wales-ish 78", 87, 78},
		{"85 vs 75", 85, 75},
		{"80 vs 70", 80, 70},

		// Wide gap — should be a near-walkover.
		{"90 vs 70", 90, 70},
		{"85 vs 65", 85, 65},
		{"top vs minnow 85 vs 55", 85, 55},
	}

	type result struct {
		label                      string
		home, away                 int
		homeWin, draw, awayWin     float64
		avgHomeGoals, avgAwayGoals float64
	}
	var results []result

	for _, p := range pairs {
		home := uniformTeam("home", p.home, soccer.FormationTypeDiamond)
		away := uniformTeam("away", p.away, soccer.FormationTypeDiamond)

		var homeWins, awayWins, draws int
		var homeGoals, awayGoals int
		for i := 0; i < skillGapTrials; i++ {
			events, _, err := soccer.RunGameWithSeed(
				rand.New(rand.NewSource(int64(i))),
				home, away,
			)
			if err != nil {
				t.Fatalf("run game: %v", err)
			}
			s := soccer.CreateGameStats(events)
			homeGoals += s.HomeTeamStats.Goals
			awayGoals += s.AwayTeamStats.Goals
			switch {
			case s.HomeTeamStats.Goals > s.AwayTeamStats.Goals:
				homeWins++
			case s.AwayTeamStats.Goals > s.HomeTeamStats.Goals:
				awayWins++
			default:
				draws++
			}
		}
		n := float64(skillGapTrials)
		results = append(results, result{
			label:        p.label,
			home:         p.home,
			away:         p.away,
			homeWin:      float64(homeWins) / n,
			draw:         float64(draws) / n,
			awayWin:      float64(awayWins) / n,
			avgHomeGoals: float64(homeGoals) / n,
			avgAwayGoals: float64(awayGoals) / n,
		})
	}

	t.Log("\nskill-gap win rates (home / draw / away, avg goals):")
	t.Log("-------------------------------------------------------------------------")
	t.Logf("  %-34s   home    draw    away   goals (h-a)", "matchup")
	for _, r := range results {
		t.Logf("  %-34s  %5.1f%%  %5.1f%%  %5.1f%%   %.2f - %.2f",
			r.label, r.homeWin*100, r.draw*100, r.awayWin*100,
			r.avgHomeGoals, r.avgAwayGoals)
	}
}

// uniformTeam builds a 5-player Diamond squad where every relevant attribute
// on every player equals `rating`. This isolates the rating differential as
// the only variable in skill-gap experiments.
func uniformTeam(id string, rating int, formation soccer.FormationType) soccer.GameLineup {
	positions := []soccer.PlayerPosition{
		soccer.PlayerPositionGoalkeeper,
		soccer.PlayerPositionDefense,
		soccer.PlayerPositionMidfield,
		soccer.PlayerPositionMidfield,
		soccer.PlayerPositionAttack,
	}
	players := make([]soccer.SelectedPlayer, len(positions))
	for i, pos := range positions {
		players[i] = soccer.SelectedPlayer{
			ID:   id + "-" + string(rune('0'+i+1)),
			Name: id + "-" + string(rune('0'+i+1)),
			Attributes: soccer.PlayerAttributes{
				GoalkeeperRating: rating,
				DefenseRating:    rating,
				ControlRating:    rating,
				AttackRating:     rating,
				SpeedRating:      rating,
				WorkRate:         rating,
				OverallRating:    rating,
				PrimaryPosition:  pos,
				Positions:        []soccer.PlayerPosition{pos},
			},
			SelectedPosition: pos,
		}
	}
	return soccer.GameLineup{
		Team:    soccer.Team{ID: id, Formation: formation},
		Players: players,
	}
}
