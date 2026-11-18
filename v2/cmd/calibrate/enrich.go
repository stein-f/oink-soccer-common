package main

import (
	"math/rand"
	"regexp"

	soccer "github.com/stein-f/oink-soccer-common/v2"
	"github.com/stein-f/oink-soccer-common/v2/allocation/fifacsv"
)

// Line-ups archived before season 19 carry none of the player-on-player
// attributes, so the step model would play them on fallbacks. Every archived
// player records the dataset player he is based on; enrich looks that player
// up in a dataset and gives the line-up what he really has.
//
// Two ways, by mode:
//   - "steps": only the player-on-player attributes are filled in. The squad
//     is the one that played, seen in full.
//   - "all": every attribute is replaced by the dataset's. With the FC 27
//     file this is the same squad one dataset on, where the player is still
//     in it; a player the dataset no longer has is left as he was.

var sofifaID = regexp.MustCompile(`/player/(\d+)/`)

type enricher struct {
	byID map[string]soccer.PlayerAttributes
	mode string

	matched, unmatched int
}

func newEnricher(path, mode string) (*enricher, error) {
	// The seed only decides the 5-30 filler for ratings the dataset leaves
	// blank; fixed so runs compare.
	candidates, err := fifacsv.LoadCandidates(path, rand.New(rand.NewSource(1)))
	if err != nil {
		return nil, err
	}
	e := &enricher{byID: make(map[string]soccer.PlayerAttributes, len(candidates)), mode: mode}
	for _, c := range candidates {
		e.byID[c.ID] = c.Attributes
	}
	return e, nil
}

func (e *enricher) lineup(l soccer.GameLineup) soccer.GameLineup {
	players := make([]soccer.SelectedPlayer, len(l.Players))
	copy(players, l.Players)
	for i := range players {
		players[i].Attributes = e.player(players[i].Attributes)
	}
	l.Players = players
	return l
}

func (e *enricher) player(a soccer.PlayerAttributes) soccer.PlayerAttributes {
	m := sofifaID.FindStringSubmatch(a.BasedOnPlayerURL)
	if m == nil {
		e.unmatched++
		return a
	}
	src, ok := e.byID[m[1]]
	if !ok {
		e.unmatched++
		return a
	}
	e.matched++
	if e.mode == "all" {
		return src
	}
	a.Dribbling, a.Passing, a.Crossing, a.Pace = src.Dribbling, src.Passing, src.Crossing, src.Pace
	a.Strength, a.Interceptions, a.Positioning, a.Marking = src.Strength, src.Interceptions, src.Positioning, src.Marking
	a.ShotPower, a.Reflexes, a.Handling, a.OneOnOne = src.ShotPower, src.Reflexes, src.Handling, src.OneOnOne
	return a
}
