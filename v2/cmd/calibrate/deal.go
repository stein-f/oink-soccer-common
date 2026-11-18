package main

import (
	"fmt"
	"math/rand"
	"os"

	"github.com/gocarina/gocsv"
	soccer "github.com/stein-f/oink-soccer-common/v2"
	"github.com/stein-f/oink-soccer-common/v2/allocation"
	"github.com/stein-f/oink-soccer-common/v2/allocation/fifacsv"
)

// A season's squads are dealt, not built: every asset a team holds is given a
// new player by the allocation. So the squads to judge a new season's engine
// on are the ones that season's allocation would deal, not last season's.
//
// deal runs the real allocation (same pool rules, same code) over a list of
// eligible assets with a dataset and a seed of our choosing, and gives every
// squad player the attributes his asset drew. It is a dry run of the season
// allocation: the teams and the assets they hold are real, the players are
// the new season's.

type dealtAsset struct {
	Name     string `csv:"name"`
	PlayerID string `csv:"player_id"`
	Tier     string `csv:"tier"`
}

func deal(files []teamFile, assetsPath, datasetPath string, seed int64) error {
	raw, err := os.ReadFile(assetsPath)
	if err != nil {
		return err
	}
	var rows []dealtAsset
	if err := gocsv.UnmarshalBytes(raw, &rows); err != nil {
		return err
	}
	assets := make([]allocation.Asset, 0, len(rows))
	for _, row := range rows {
		assets = append(assets, allocation.Asset{ID: row.PlayerID, Name: row.Name, Tier: allocation.AssetTier(row.Tier)})
	}
	r := rand.New(rand.NewSource(seed))
	candidates, err := fifacsv.LoadCandidates(datasetPath, r)
	if err != nil {
		return err
	}
	assignments, err := allocation.Allocate(r, allocation.NewPool(candidates, allocation.DefaultRules()), assets)
	if err != nil {
		return err
	}
	dealt := make(map[string]soccer.PlayerAttributes, len(assignments))
	for _, a := range assignments {
		dealt[a.Asset.ID] = a.Player.Attributes
	}

	matched, missing := 0, 0
	redeal := func(e slotEntry) (slotEntry, bool) {
		attrs, ok := dealt[e.Asset.ID]
		if !ok {
			missing++
			return e, false
		}
		matched++
		e.Attributes = attrs
		e.Role = ""
		return e, true
	}
	for i := range files {
		tf := &files[i]
		// The line-up a manager picked was for last season's players. Under
		// the new deal everyone goes back in the squad to be picked again.
		var squad []slotEntry
		for _, e := range tf.TeamSelection.Slots {
			if d, ok := redeal(e); ok {
				squad = append(squad, d)
			}
		}
		for _, e := range tf.TeamSelection.NotSelected {
			if d, ok := redeal(e); ok {
				squad = append(squad, d)
			}
		}
		tf.TeamSelection.Slots = map[string]slotEntry{}
		tf.TeamSelection.NotSelected = squad
		// Nobody has chosen a shape or tactics for these players yet.
		tf.Team.Formation = string(soccer.FormationTypeDiamond)
		tf.Team.Tactics.Press, tf.Team.Tactics.Tempo, tf.Team.Tactics.LineHeight = "", "", ""
		tf.Team.Tactics.SetPieceTaker = ""
	}
	fmt.Printf("dealt %d squad players from %d assets (seed %d); %d not in the asset list, dropped\n", matched, len(assets), seed, missing)
	return nil
}
