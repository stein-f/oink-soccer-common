package main

import (
	"fmt"
	"runtime"
	"sync"

	soccer "github.com/stein-f/oink-soccer-common/v2"
	"github.com/stein-f/oink-soccer-common/v2/testdata"
)

// The tactics report answers "is each tactic a real choice?". A setting is a
// real choice when it is worth about nothing to a typical squad against a
// typical squad, and worth something, one way or the other, depending on who
// is playing: a side that can run all day should press, a side that can't
// shouldn't; quick dribblers want the game fast, passers and a target man
// want it slow; you sit off a side that runs at you and press one that
// wants time on the ball.
//
// Every row is one kind of squad against one kind of opponent, in each of
// the four shapes, home and away. The columns are what changing one setting
// is worth to the row's squad in points per game, against the same squad on
// the default settings. Each kind of squad has the same attribute points as
// the typical one: what it gains somewhere it gives up somewhere else.

type squadKind struct {
	name  string
	apply func(p *soccer.SelectedPlayer)
}

func outfield(p *soccer.SelectedPlayer) bool {
	return p.SelectedPosition != soccer.PlayerPositionGoalkeeper
}

var squadKinds = []squadKind{
	{"typical", func(*soccer.SelectedPlayer) {}},
	{"runs all day", func(p *soccer.SelectedPlayer) {
		if outfield(p) {
			p.Attributes.WorkRate += 11
		}
	}},
	{"tires", func(p *soccer.SelectedPlayer) {
		if outfield(p) {
			p.Attributes.WorkRate -= 11
		}
	}},
	{"quick dribblers", func(p *soccer.SelectedPlayer) {
		if p.SelectedPosition == soccer.PlayerPositionMidfield || p.SelectedPosition == soccer.PlayerPositionAttack {
			a := &p.Attributes
			a.Pace, a.Dribbling = a.Pace+10, a.Dribbling+10
			a.Passing, a.Crossing, a.Heading, a.Strength = a.Passing-5, a.Crossing-5, a.Heading-5, a.Strength-5
		}
	}},
	{"passers and a target man", func(p *soccer.SelectedPlayer) {
		a := &p.Attributes
		switch p.SelectedPosition {
		case soccer.PlayerPositionMidfield:
			a.Passing, a.Crossing = a.Passing+10, a.Crossing+10
			a.Pace, a.Dribbling = a.Pace-10, a.Dribbling-10
		case soccer.PlayerPositionAttack:
			a.Heading, a.Strength = a.Heading+10, a.Strength+10
			a.Pace, a.Dribbling = a.Pace-10, a.Dribbling-10
		}
	}},
}

type tacticChange struct {
	name  string
	apply func(t *soccer.Tactics)
}

var tacticChanges = []tacticChange{
	{"press low", func(t *soccer.Tactics) { t.Press = soccer.PressLevelLow }},
	{"press high", func(t *soccer.Tactics) { t.Press = soccer.PressLevelHigh }},
	{"tempo slow", func(t *soccer.Tactics) { t.Tempo = soccer.TempoLevelSlow }},
	{"tempo fast", func(t *soccer.Tactics) { t.Tempo = soccer.TempoLevelFast }},
	{"line deep", func(t *soccer.Tactics) { t.LineHeight = soccer.LineHeightDeep }},
	{"line high", func(t *soccer.Tactics) { t.LineHeight = soccer.LineHeightHigh }},
}

func squadOf(kind squadKind, f soccer.FormationType) soccer.GameLineup {
	l := testdata.TypicalTeam(f)
	for i := range l.Players {
		kind.apply(&l.Players[i])
	}
	return l
}

func tacticsReport(seeds int, model soccer.Model) {
	// Which opponents each kind of squad is shown against: everyone meets a
	// typical side, and the typical side meets everyone, since how to defend
	// depends on who is attacking.
	type row struct{ me, them int }
	var rows []row
	for i := range squadKinds {
		rows = append(rows, row{i, 0})
	}
	for j := 1; j < len(squadKinds); j++ {
		rows = append(rows, row{0, j})
	}
	values := make([][]float64, len(rows))
	type job struct{ r, c int }
	jobs := make(chan job)
	var wg sync.WaitGroup
	var mu sync.Mutex
	for w := 0; w < runtime.NumCPU(); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				var sum float64
				for _, f := range reportFormations {
					me, them := squadOf(squadKinds[rows[j.r].me], f), squadOf(squadKinds[rows[j.r].them], f)
					them.Team.ID = "them"
					base := pointsPerGame(me, them, seeds, model)
					tacticChanges[j.c].apply(&me.Team.Tactics)
					sum += pointsPerGame(me, them, seeds, model) - base
				}
				mu.Lock()
				values[j.r][j.c] = sum / float64(len(reportFormations))
				mu.Unlock()
			}
		}()
	}
	for r := range rows {
		values[r] = make([]float64, len(tacticChanges))
		for c := range tacticChanges {
			jobs <- job{r, c}
		}
	}
	close(jobs)
	wg.Wait()

	label := string(model)
	if label == "" {
		label = "classic"
	}
	fmt.Printf("\n== is each tactic a real choice? model %s (points per game for changing one setting; %d seeds x 2 legs x 4 shapes)\n\n", label, seeds)
	fmt.Printf("%-26s %-26s", "squad", "against")
	for _, c := range tacticChanges {
		fmt.Printf(" %11s", c.name)
	}
	fmt.Println()
	for r, rw := range rows {
		fmt.Printf("%-26s %-26s", squadKinds[rw.me].name, squadKinds[rw.them].name)
		for c := range tacticChanges {
			fmt.Printf(" %+11.3f", values[r][c])
		}
		fmt.Println()
	}
	fmt.Printf("\nstandard error of a cell is about %.3f.\n", 1.3*1.41/sqrtf(float64(2*seeds*len(reportFormations))))
}
