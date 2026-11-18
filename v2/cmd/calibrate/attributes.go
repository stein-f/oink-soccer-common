package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"sort"
	"sync"

	soccer "github.com/stein-f/oink-soccer-common/v2"
	"github.com/stein-f/oink-soccer-common/v2/testdata"
)

// The attribute report answers "what is five points of this attribute worth,
// at this position?" in points per game. It is how a dominant attribute is
// caught before a season is played on it, and what the manager guide's
// "what each attribute does" is written from.
//
// A typical squad (testdata.TypicalTeam) plays itself, home and away, with
// one attribute raised by five for every player in one position. Formations
// are averaged so each position is seen in the shapes that field it.

type attributeSetter struct {
	name string
	add  func(a *soccer.PlayerAttributes, d int)
}

// attributeSetters is every attribute on its own, led by all of them at once
// (a player who is simply five points better).
var attributeSetters = append([]attributeSetter{{"Every attribute", func(a *soccer.PlayerAttributes, d int) {
	for _, s := range singleAttributes {
		s.add(a, d)
	}
}}}, singleAttributes...)

var singleAttributes = []attributeSetter{
	// The number on the card. The step model reads it for possession (see
	// tuning.StepsPossessionExponent); the classic model doesn't read it.
	{"OverallRating", func(a *soccer.PlayerAttributes, d int) { a.OverallRating += d }},
	{"GoalkeeperRating", func(a *soccer.PlayerAttributes, d int) { a.GoalkeeperRating += d }},
	{"DefenseRating", func(a *soccer.PlayerAttributes, d int) { a.DefenseRating += d }},
	{"ControlRating", func(a *soccer.PlayerAttributes, d int) { a.ControlRating += d }},
	{"AttackRating", func(a *soccer.PlayerAttributes, d int) { a.AttackRating += d }},
	{"SpeedRating", func(a *soccer.PlayerAttributes, d int) { a.SpeedRating += d }},
	{"AggressionRating", func(a *soccer.PlayerAttributes, d int) { a.AggressionRating += d }},
	{"WorkRate", func(a *soccer.PlayerAttributes, d int) { a.WorkRate += d }},
	{"Finishing", func(a *soccer.PlayerAttributes, d int) { a.Finishing += d }},
	{"Heading", func(a *soccer.PlayerAttributes, d int) { a.Heading += d }},
	{"Technique", func(a *soccer.PlayerAttributes, d int) { a.Technique += d }},
	{"Composure", func(a *soccer.PlayerAttributes, d int) { a.Composure += d }},
	{"Tackling", func(a *soccer.PlayerAttributes, d int) { a.Tackling += d }},
	{"Dribbling", func(a *soccer.PlayerAttributes, d int) { a.Dribbling += d }},
	{"Passing", func(a *soccer.PlayerAttributes, d int) { a.Passing += d }},
	{"Crossing", func(a *soccer.PlayerAttributes, d int) { a.Crossing += d }},
	{"Pace", func(a *soccer.PlayerAttributes, d int) { a.Pace += d }},
	{"Strength", func(a *soccer.PlayerAttributes, d int) { a.Strength += d }},
	{"Interceptions", func(a *soccer.PlayerAttributes, d int) { a.Interceptions += d }},
	{"Positioning", func(a *soccer.PlayerAttributes, d int) { a.Positioning += d }},
	{"Marking", func(a *soccer.PlayerAttributes, d int) { a.Marking += d }},
	{"ShotPower", func(a *soccer.PlayerAttributes, d int) { a.ShotPower += d }},
	{"Reflexes", func(a *soccer.PlayerAttributes, d int) { a.Reflexes += d }},
	{"Handling", func(a *soccer.PlayerAttributes, d int) { a.Handling += d }},
	{"OneOnOne", func(a *soccer.PlayerAttributes, d int) { a.OneOnOne += d }},
}

var reportPositions = []soccer.PlayerPosition{
	soccer.PlayerPositionGoalkeeper, soccer.PlayerPositionDefense, soccer.PlayerPositionMidfield, soccer.PlayerPositionAttack,
}

var reportFormations = []soccer.FormationType{
	soccer.FormationTypePyramid, soccer.FormationTypeDiamond, soccer.FormationTypeY, soccer.FormationTypeBox,
}

// pointsPerGame plays team against base, home and away, on the same seeds.
func pointsPerGame(team, base soccer.GameLineup, seeds int, model soccer.Model) float64 {
	points := 0
	for s := 0; s < seeds; s++ {
		for leg := 0; leg < 2; leg++ {
			home, away := team, base
			if leg == 1 {
				home, away = base, team
			}
			out, err := soccer.RunGameWithModel(rand.New(rand.NewSource(int64(s))), home, away, model)
			if err != nil {
				panic(err)
			}
			stats := soccer.CreateGameStats(out.Events)
			mine, theirs := stats.HomeTeamStats.Goals, stats.AwayTeamStats.Goals
			if leg == 1 {
				mine, theirs = theirs, mine
			}
			switch {
			case mine > theirs:
				points += 3
			case mine == theirs:
				points++
			}
		}
	}
	return float64(points) / float64(2*seeds)
}

func attributeReport(seeds int, model soccer.Model, delta int) {
	type cell struct {
		attr, pos int
		value     float64
		n         int
	}
	cells := make([]cell, 0, len(attributeSetters)*len(reportPositions))
	for a := range attributeSetters {
		for p := range reportPositions {
			cells = append(cells, cell{attr: a, pos: p})
		}
	}

	baseline := map[soccer.FormationType]float64{}
	for _, f := range reportFormations {
		baseline[f] = pointsPerGame(testdata.TypicalTeam(f), testdata.TypicalTeam(f), seeds, model)
	}

	var wg sync.WaitGroup
	work := make(chan int)
	for w := 0; w < runtime.NumCPU(); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range work {
				c := &cells[i]
				for _, f := range reportFormations {
					team := testdata.TypicalTeam(f)
					fielded := false
					for j := range team.Players {
						if team.Players[j].SelectedPosition == reportPositions[c.pos] {
							attributeSetters[c.attr].add(&team.Players[j].Attributes, delta)
							fielded = true
						}
					}
					if !fielded {
						continue
					}
					c.value += pointsPerGame(team, testdata.TypicalTeam(f), seeds, model) - baseline[f]
					c.n++
				}
			}
		}()
	}
	for i := range cells {
		work <- i
	}
	close(work)
	wg.Wait()

	value := func(a, p int) float64 {
		c := cells[a*len(reportPositions)+p]
		if c.n == 0 {
			return 0
		}
		return c.value / float64(c.n)
	}
	fmt.Printf("\n== what +%d of an attribute is worth, in points per game (typical squad v itself, %d seeds x 2 legs x 4 formations)\n\n", delta, seeds)
	fmt.Printf("%-28s %10s %10s %10s %10s %10s\n", "attribute", "keeper", "defence", "midfield", "attack", "all four")
	order := make([]int, len(attributeSetters))
	for i := range order {
		order[i] = i
	}
	total := func(a int) float64 {
		var t float64
		for p := range reportPositions {
			t += value(a, p)
		}
		return t
	}
	sort.SliceStable(order[1:], func(i, j int) bool { return total(order[1+i]) > total(order[1+j]) })
	for _, a := range order {
		fmt.Printf("%-28s", attributeSetters[a].name)
		for p := range reportPositions {
			fmt.Printf(" %+10.3f", value(a, p))
		}
		fmt.Printf(" %+10.3f\n", total(a))
	}
	fmt.Printf("\nstandard error of one cell is about %.3f; read small numbers as zero.\n", 1.3/sqrtf(float64(2*seeds*len(reportFormations))))
}

func sqrtf(x float64) float64 {
	z := x / 2
	for i := 0; i < 30 && z > 0; i++ {
		z = (z + x/z) / 2
	}
	return z
}
