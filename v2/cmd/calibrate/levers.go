package main

import (
	"fmt"

	soccer "github.com/stein-f/oink-soccer-common/v2"
	"github.com/stein-f/oink-soccer-common/v2/testdata"
)

// The levers report answers "what is a manager's choice worth?": an item
// boost, each tactic setting, a role. A typical squad plays itself, home and
// away, with one thing changed on one side. A new model must leave these
// worth at least what they are worth on the live one, or managers who paid
// for an item or worked out their tactics lose what they had.
func leversReport(seeds int, model soccer.Model) {
	base := func() soccer.GameLineup { return testdata.TypicalTeam(soccer.FormationTypeDiamond) }
	baseline := pointsPerGame(base(), base(), seeds, model)

	type lever struct {
		name  string
		apply func(l *soccer.GameLineup)
	}
	boost := func(m float64) func(*soccer.GameLineup) {
		return func(l *soccer.GameLineup) {
			l.ItemBoosts = append(l.ItemBoosts, soccer.Boost{BoostType: soccer.BoostTypeTeam, MinBoost: m, MaxBoost: m})
		}
	}
	role := func(pos soccer.PlayerPosition, r soccer.PlayerRole) func(*soccer.GameLineup) {
		return func(l *soccer.GameLineup) {
			for i := range l.Players {
				if l.Players[i].SelectedPosition == pos {
					l.Players[i].Role = r
					return
				}
			}
		}
	}
	levers := []lever{
		{"team boost +3%", boost(1.03)},
		{"team boost +5%", boost(1.05)},
		{"team boost +10%", boost(1.10)},
		{"press low", func(l *soccer.GameLineup) { l.Team.Tactics.Press = soccer.PressLevelLow }},
		{"press high", func(l *soccer.GameLineup) { l.Team.Tactics.Press = soccer.PressLevelHigh }},
		{"tempo slow", func(l *soccer.GameLineup) { l.Team.Tactics.Tempo = soccer.TempoLevelSlow }},
		{"tempo fast", func(l *soccer.GameLineup) { l.Team.Tactics.Tempo = soccer.TempoLevelFast }},
		{"line deep", func(l *soccer.GameLineup) { l.Team.Tactics.LineHeight = soccer.LineHeightDeep }},
		{"line high", func(l *soccer.GameLineup) { l.Team.Tactics.LineHeight = soccer.LineHeightHigh }},
		{"captain (a midfielder)", role(soccer.PlayerPositionMidfield, soccer.PlayerRoleCaptain)},
		{"playmaker", role(soccer.PlayerPositionMidfield, soccer.PlayerRolePlaymaker)},
		{"ball winner", role(soccer.PlayerPositionDefense, soccer.PlayerRoleBallWinner)},
		{"target man", role(soccer.PlayerPositionAttack, soccer.PlayerRoleTargetMan)},
	}
	label := string(model)
	if label == "" {
		label = "classic"
	}
	fmt.Printf("\n== what a manager's choice is worth, model %s (typical squad v itself in a Diamond, %d seeds x 2 legs; baseline %.2f points per game)\n\n", label, seeds, baseline)
	fmt.Printf("%-26s %12s\n", "one side changes", "points/game")
	for _, lv := range levers {
		team := base()
		lv.apply(&team)
		fmt.Printf("%-26s %+12.3f\n", lv.name, pointsPerGame(team, base(), seeds, model)-baseline)
	}
	fmt.Printf("\nstandard error of a row is about %.3f.\n", 1.3/sqrtf(float64(2*seeds)))
}
