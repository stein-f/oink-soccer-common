package main

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	soccer "github.com/stein-f/oink-soccer-common/v2"
)

// The formations report is the gate for formation balance. It asks the
// question a manager asks: with my squad, in my league, which shape should I
// play, and what does it cost me to play another?
//
// Every team in a league gets its best line-up and tactics in each formation
// (a hill-climb over its whole squad), and plays the other teams as their
// managers have them set. A balanced engine has every formation the right
// choice for some squads, and none that takes a strong squad out of the
// running.
//
// Identical squads can't show this. The classic and contest models both
// passed the identical-squad balance test while one formation was the worst
// choice for every real team: what a shape costs depends on the players a
// squad has for it.
//
// Input is a directory of lost-pigs team API responses
// (GET /v2/soccer/team/<id>), one file per team of one league.
//
//	go run ./cmd/calibrate -formations <dir> -model steps [-enrich <dataset.csv>]

// formationsModel is the model the report plays on.
var formationsModel soccer.Model

func parallelFor(n int, fn func(i int)) {
	var idx int64 = -1
	var wg sync.WaitGroup
	for w := 0; w < runtime.NumCPU(); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				i := int(atomic.AddInt64(&idx, 1))
				if i >= n {
					return
				}
				fn(i)
			}
		}()
	}
	wg.Wait()
}

type teamFile struct {
	Team struct {
		ID         string `json:"id"`
		CustomName string `json:"custom_name"`
		Formation  string `json:"formation"`
		Tactics    struct {
			Press         string `json:"Press"`
			Tempo         string `json:"Tempo"`
			LineHeight    string `json:"LineHeight"`
			SetPieceTaker string `json:"SetPieceTaker"`
		} `json:"tactics"`
	} `json:"team"`
	TeamSelection struct {
		Slots       map[string]slotEntry `json:"slots"`
		NotSelected []slotEntry          `json:"not_selected"`
	} `json:"team_selection"`
}

type slotEntry struct {
	Asset struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"asset"`
	Attributes soccer.PlayerAttributes `json:"player_attributes"`
	Role       string                  `json:"role"`
}

type player struct {
	ID   string
	Name string
	Attr soccer.PlayerAttributes
}

func (p player) canPlay(pos soccer.PlayerPosition) bool {
	if p.Attr.PrimaryPosition == pos || p.Attr.PrimaryPosition == soccer.PlayerPositionAny {
		return true
	}
	for _, q := range p.Attr.Positions {
		if q == pos || q == soccer.PlayerPositionAny {
			return true
		}
	}
	return false
}

var formations = []soccer.FormationType{
	soccer.FormationTypePyramid, soccer.FormationTypeDiamond, soccer.FormationTypeY, soccer.FormationTypeBox,
}

var formationSlots = map[soccer.FormationType]map[uint64]soccer.PlayerPosition{
	soccer.FormationTypePyramid: soccer.ThePyramidFormation.Slots,
	soccer.FormationTypeDiamond: soccer.TheDiamondFormation.Slots,
	soccer.FormationTypeY:       soccer.TheYFormation.Slots,
	soccer.FormationTypeBox:     soccer.TheBoxFormation.Slots,
}

var positionKey = map[soccer.PlayerPosition]func(soccer.PlayerAttributes) int{
	soccer.PlayerPositionGoalkeeper: func(a soccer.PlayerAttributes) int { return a.GoalkeeperRating },
	soccer.PlayerPositionDefense:    func(a soccer.PlayerAttributes) int { return a.DefenseRating + a.EffectiveTackling() },
	soccer.PlayerPositionMidfield:   func(a soccer.PlayerAttributes) int { return a.ControlRating + a.AttackRating },
	soccer.PlayerPositionAttack:     func(a soccer.PlayerAttributes) int { return a.AttackRating },
}

func positionCounts(f soccer.FormationType) (def, mid, atk int) {
	for _, pos := range formationSlots[f] {
		switch pos {
		case soccer.PlayerPositionDefense:
			def++
		case soccer.PlayerPositionMidfield:
			mid++
		case soccer.PlayerPositionAttack:
			atk++
		}
	}
	return
}

func mustLoad(path string) teamFile {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var tf teamFile
	if err := json.Unmarshal(b, &tf); err != nil {
		panic(fmt.Errorf("%s: %w", path, err))
	}
	return tf
}

func poolFromFile(tf teamFile) []player {
	m := map[string]player{}
	add := func(e slotEntry) {
		if e.Asset.ID != "" {
			m[e.Asset.ID] = player{ID: e.Asset.ID, Name: e.Asset.Name, Attr: e.Attributes}
		}
	}
	for _, e := range tf.TeamSelection.Slots {
		add(e)
	}
	for _, e := range tf.TeamSelection.NotSelected {
		add(e)
	}
	pool := make([]player, 0, len(m))
	for _, p := range m {
		pool = append(pool, p)
	}
	sort.Slice(pool, func(i, j int) bool { return pool[i].ID < pool[j].ID })
	return pool
}

type lineup struct {
	formation soccer.FormationType
	gk        player
	def       []player
	mid       []player
	atk       []player
}

func (l lineup) all() []player {
	out := []player{l.gk}
	out = append(out, l.def...)
	out = append(out, l.mid...)
	return append(out, l.atk...)
}

func (l lineup) names() string {
	var s []string
	for _, p := range l.all() {
		s = append(s, p.Name)
	}
	return strings.Join(s, ", ")
}

type tactics struct {
	press soccer.PressLevel
	tempo soccer.TempoLevel
	line  soccer.LineHeight
}

func (t tactics) String() string {
	or := func(s string) string {
		if s == "" {
			return "-"
		}
		return s
	}
	return or(string(t.press)) + "/" + or(string(t.tempo)) + "/" + or(string(t.line))
}

func (l lineup) gameLineup(id string, t tactics) soccer.GameLineup {
	var captain, playmaker, ballWinner, targetMan, setPiece string
	best := -1
	for _, p := range l.all() {
		q := (p.Attr.ControlRating + p.Attr.EffectiveComposure()) / 2
		if p.Attr.PrimaryPosition == soccer.PlayerPositionGoalkeeper {
			q = (p.Attr.GoalkeeperRating + p.Attr.EffectiveComposure()) / 2
		}
		if q > best {
			best, captain = q, p.ID
		}
	}
	if len(l.mid) >= 2 {
		best = -1
		for _, p := range l.mid {
			if p.Attr.ControlRating > best {
				best, playmaker = p.Attr.ControlRating, p.ID
			}
		}
	}
	if len(l.def) >= 2 {
		best = -1
		for _, p := range l.def {
			if p.Attr.EffectiveTackling() > best {
				best, ballWinner = p.Attr.EffectiveTackling(), p.ID
			}
		}
	}
	best = -1
	for _, p := range l.atk {
		if p.Attr.EffectiveHeading() > best {
			best, targetMan = p.Attr.EffectiveHeading(), p.ID
		}
	}
	best = -1
	for _, p := range l.all() {
		if p.Attr.PrimaryPosition == soccer.PlayerPositionGoalkeeper {
			continue
		}
		if p.Attr.EffectiveTechnique() > best {
			best, setPiece = p.Attr.EffectiveTechnique(), p.ID
		}
	}

	var players []soccer.SelectedPlayer
	add := func(p player, pos soccer.PlayerPosition) {
		if p.ID == "" {
			return
		}
		sp := soccer.SelectedPlayer{ID: p.ID, Name: p.Name, Attributes: p.Attr, SelectedPosition: pos}
		switch p.ID {
		case captain:
			sp.Role = soccer.PlayerRoleCaptain
		case playmaker:
			sp.Role = soccer.PlayerRolePlaymaker
		case ballWinner:
			sp.Role = soccer.PlayerRoleBallWinner
		case targetMan:
			sp.Role = soccer.PlayerRoleTargetMan
		}
		players = append(players, sp)
	}
	add(l.gk, soccer.PlayerPositionGoalkeeper)
	for _, p := range l.def {
		add(p, soccer.PlayerPositionDefense)
	}
	for _, p := range l.mid {
		add(p, soccer.PlayerPositionMidfield)
	}
	for _, p := range l.atk {
		add(p, soccer.PlayerPositionAttack)
	}
	return soccer.GameLineup{
		Team: soccer.Team{
			ID: id, Formation: l.formation,
			Tactics: soccer.Tactics{Press: t.press, Tempo: t.tempo, LineHeight: t.line, SetPieceTaker: setPiece},
		},
		Players: players,
	}
}

func candidates(pool []player, pos soccer.PlayerPosition, n, need int) []player {
	key := positionKey[pos]
	var elig, rest []player
	for _, p := range pool {
		if p.canPlay(pos) {
			elig = append(elig, p)
		} else if !p.canPlay(soccer.PlayerPositionGoalkeeper) || pos == soccer.PlayerPositionGoalkeeper {
			rest = append(rest, p)
		}
	}
	sort.SliceStable(elig, func(i, j int) bool { return key(elig[i].Attr) > key(elig[j].Attr) })
	sort.SliceStable(rest, func(i, j int) bool { return key(rest[i].Attr) > key(rest[j].Attr) })
	if len(elig) > n {
		elig = elig[:n]
	}
	for len(elig) < need && len(rest) > 0 {
		elig = append(elig, rest[0])
		rest = rest[1:]
	}
	return elig
}

var searchKeys = map[soccer.PlayerPosition][]func(soccer.PlayerAttributes) int{
	soccer.PlayerPositionGoalkeeper: {
		func(a soccer.PlayerAttributes) int { return a.GoalkeeperRating*5 + a.SpeedRating },
	},
	soccer.PlayerPositionDefense: {
		func(a soccer.PlayerAttributes) int {
			return a.DefenseRating*5 + a.EffectiveTackling()*2 + a.SpeedRating
		},
		func(a soccer.PlayerAttributes) int { return a.ControlRating*4 + a.EffectiveWorkRate() },
		func(a soccer.PlayerAttributes) int { return a.DefenseRating + a.ControlRating },
	},
	soccer.PlayerPositionMidfield: {
		func(a soccer.PlayerAttributes) int { return a.ControlRating*4 + a.EffectiveWorkRate() },
		func(a soccer.PlayerAttributes) int { return a.ControlRating*3 + a.EffectiveWorkRate()*2 },
		func(a soccer.PlayerAttributes) int { return a.ControlRating + a.AttackRating },
	},
	soccer.PlayerPositionAttack: {
		func(a soccer.PlayerAttributes) int { return a.AttackRating*3 + a.SpeedRating },
		func(a soccer.PlayerAttributes) int { return a.ControlRating*4 + a.EffectiveWorkRate() },
		func(a soccer.PlayerAttributes) int { return a.AttackRating + a.ControlRating },
		func(a soccer.PlayerAttributes) int { return a.EffectiveFinishing() + a.AttackRating },
	},
}

func shortlist(pool []player, pos soccer.PlayerPosition, n, need int, fielded []player) []player {
	seen := map[string]bool{}
	var out []player
	add := func(p player) {
		if p.ID != "" && !seen[p.ID] {
			seen[p.ID] = true
			out = append(out, p)
		}
	}
	var elig []player
	for _, p := range pool {
		if p.canPlay(pos) {
			elig = append(elig, p)
		}
	}
	for _, key := range searchKeys[pos] {
		ranked := append([]player(nil), elig...)
		sort.SliceStable(ranked, func(i, j int) bool { return key(ranked[i].Attr) > key(ranked[j].Attr) })
		for i := 0; i < n && i < len(ranked); i++ {
			add(ranked[i])
		}
	}
	for _, p := range fielded {
		if p.canPlay(pos) {
			add(p)
		}
	}
	// A squad that can't fill the position plays its best others out of position.
	if len(out) < need+1 {
		for _, p := range candidates(pool, pos, 0, need+2) {
			add(p)
		}
	}
	return out
}

func (l lineup) slots() []*player {
	out := []*player{&l.gk}
	for i := range l.def {
		out = append(out, &l.def[i])
	}
	for i := range l.mid {
		out = append(out, &l.mid[i])
	}
	for i := range l.atk {
		out = append(out, &l.atk[i])
	}
	return out
}

func (l lineup) clone() lineup {
	l.def = append([]player(nil), l.def...)
	l.mid = append([]player(nil), l.mid...)
	l.atk = append([]player(nil), l.atk...)
	return l
}

func (l lineup) has(id string) bool {
	for _, p := range l.all() {
		if p.ID == id {
			return true
		}
	}
	return false
}

func search(pool []player, fielded lineup, f soccer.FormationType, t tactics, opps []soccer.GameLineup) lineup {
	nd, nm, na := positionCounts(f)
	short := map[soccer.PlayerPosition][]player{
		soccer.PlayerPositionGoalkeeper: shortlist(pool, soccer.PlayerPositionGoalkeeper, 2, 1, fielded.all()),
		soccer.PlayerPositionDefense:    shortlist(pool, soccer.PlayerPositionDefense, 3, nd, fielded.all()),
		soccer.PlayerPositionMidfield:   shortlist(pool, soccer.PlayerPositionMidfield, 3, nm, fielded.all()),
		soccer.PlayerPositionAttack:     shortlist(pool, soccer.PlayerPositionAttack, 3, na, fielded.all()),
	}
	cur := lineup{formation: f}
	if fielded.formation == f {
		cur = fielded.clone()
	} else {
		pick := func(pos soccer.PlayerPosition, n int) []player {
			var got []player
			for _, p := range short[pos] {
				if len(got) < n && !cur.has(p.ID) {
					got = append(got, p)
					switch pos {
					case soccer.PlayerPositionGoalkeeper:
						cur.gk = p
					case soccer.PlayerPositionDefense:
						cur.def = append(cur.def, p)
					case soccer.PlayerPositionMidfield:
						cur.mid = append(cur.mid, p)
					case soccer.PlayerPositionAttack:
						cur.atk = append(cur.atk, p)
					}
				}
			}
			return got
		}
		pick(soccer.PlayerPositionGoalkeeper, 1)
		pick(soccer.PlayerPositionDefense, nd)
		pick(soccer.PlayerPositionMidfield, nm)
		pick(soccer.PlayerPositionAttack, na)
	}
	posOf := func(i int) soccer.PlayerPosition {
		switch {
		case i == 0:
			return soccer.PlayerPositionGoalkeeper
		case i <= nd:
			return soccer.PlayerPositionDefense
		case i <= nd+nm:
			return soccer.PlayerPositionMidfield
		}
		return soccer.PlayerPositionAttack
	}
	const seeds = 24
	score := evaluatePar(cur.gameLineup("MINE", t), opps, 1, seeds).ppg()
	for iter := 0; iter < 8; iter++ {
		var next []scored
		for i := range cur.slots() {
			for _, c := range short[posOf(i)] {
				if cur.has(c.ID) {
					continue
				}
				n := cur.clone()
				*n.slots()[i] = c
				next = append(next, scored{l: n, t: t})
			}
		}
		if len(next) == 0 {
			break
		}
		evalAll(next, opps, 1, seeds)
		if next[0].r.ppg() < score+0.02 {
			break
		}
		cur, score = next[0].l, next[0].r.ppg()
	}
	return cur
}

func asConfigured(tf teamFile) (lineup, tactics) {
	f := soccer.FormationType(tf.Team.Formation)
	slots, ok := formationSlots[f]
	if !ok {
		f = soccer.FormationTypeDiamond
		slots = formationSlots[f]
	}
	pool := poolFromFile(tf)
	used := map[string]bool{}
	bySlot := map[uint64]player{}
	for k, e := range tf.TeamSelection.Slots {
		var num uint64
		if _, err := fmt.Sscanf(k, "%d", &num); err != nil || e.Asset.ID == "" {
			continue
		}
		bySlot[num] = player{ID: e.Asset.ID, Name: e.Asset.Name, Attr: e.Attributes}
		used[e.Asset.ID] = true
	}
	l := lineup{formation: f}
	for num := uint64(1); num <= 5; num++ {
		pos := slots[num]
		p, ok := bySlot[num]
		if !ok {
			best := -1
			for pass := 0; pass < 2 && best < 0; pass++ {
				for _, c := range pool {
					if used[c.ID] || (pass == 0 && !c.canPlay(pos)) {
						continue
					}
					if q := positionKey[pos](c.Attr); q > best {
						best, p = q, c
					}
				}
			}
			if best < 0 {
				continue
			}
			used[p.ID] = true
		}
		switch pos {
		case soccer.PlayerPositionGoalkeeper:
			l.gk = p
		case soccer.PlayerPositionDefense:
			l.def = append(l.def, p)
		case soccer.PlayerPositionMidfield:
			l.mid = append(l.mid, p)
		case soccer.PlayerPositionAttack:
			l.atk = append(l.atk, p)
		}
	}
	return l, tactics{
		press: soccer.PressLevel(tf.Team.Tactics.Press),
		tempo: soccer.TempoLevel(tf.Team.Tactics.Tempo),
		line:  soccer.LineHeight(tf.Team.Tactics.LineHeight),
	}
}

type result struct {
	Games           int
	Pts             float64
	W, D, L         int
	GF, GA          int
	Shots, ShotsAgn int
}

func (r *result) tally(mg, og, ms, os int) {
	r.Games++
	r.GF += mg
	r.GA += og
	r.Shots += ms
	r.ShotsAgn += os
	switch {
	case mg > og:
		r.W++
		r.Pts += 3
	case mg == og:
		r.D++
		r.Pts++
	default:
		r.L++
	}
}

func (r result) ppg() float64 { return r.Pts / float64(r.Games) }

func play(seed int64, home, away soccer.GameLineup) soccer.GameStats {
	out, err := soccer.RunGameWithModel(rand.New(rand.NewSource(seed)), home, away, formationsModel)
	if err != nil {
		panic(err)
	}
	return soccer.CreateGameStats(out.Events)
}

func evaluate(my soccer.GameLineup, opps []soccer.GameLineup, seedBase int64, seeds int) result {
	var r result
	for _, opp := range opps {
		for s := 0; s < seeds; s++ {
			seed := seedBase + int64(s)
			st := play(seed, my, opp)
			r.tally(st.HomeTeamStats.Goals, st.AwayTeamStats.Goals, st.HomeTeamStats.Shots, st.AwayTeamStats.Shots)
			st = play(seed, opp, my)
			r.tally(st.AwayTeamStats.Goals, st.HomeTeamStats.Goals, st.AwayTeamStats.Shots, st.HomeTeamStats.Shots)
		}
	}
	return r
}

func evaluatePar(my soccer.GameLineup, opps []soccer.GameLineup, seedBase int64, seeds int) result {
	parts := make([]result, len(opps))
	parallelFor(len(opps), func(i int) {
		parts[i] = evaluate(my, opps[i:i+1], seedBase, seeds)
	})
	var r result
	for _, p := range parts {
		r.Games += p.Games
		r.Pts += p.Pts
		r.W += p.W
		r.D += p.D
		r.L += p.L
		r.GF += p.GF
		r.GA += p.GA
		r.Shots += p.Shots
		r.ShotsAgn += p.ShotsAgn
	}
	return r
}

type scored struct {
	l lineup
	t tactics
	r result
}

func evalAll(cands []scored, opps []soccer.GameLineup, seedBase int64, seeds int) {
	parallelFor(len(cands), func(i int) {
		cands[i].r = evaluate(cands[i].l.gameLineup("MINE", cands[i].t), opps, seedBase, seeds)
	})
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].r.ppg() > cands[j].r.ppg() })
}

func top(c []scored, n int) []scored {
	if len(c) > n {
		c = c[:n]
	}
	return append([]scored(nil), c...)
}

var allTactics = func() []tactics {
	var out []tactics
	for _, p := range []soccer.PressLevel{"low", "medium", "high"} {
		for _, t := range []soccer.TempoLevel{"slow", "normal", "fast"} {
			for _, l := range []soccer.LineHeight{"deep", "normal", "high"} {
				out = append(out, tactics{p, t, l})
			}
		}
	}
	return out
}()

// loadTeams reads every team file in dir, filling in dataset attributes when
// an enricher is given.
func loadTeams(dir string, e *enricher) ([]teamFile, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	var out []teamFile
	for _, p := range paths {
		tf := mustLoad(p)
		if tf.Team.ID == "" {
			continue
		}
		if e != nil {
			for k, entry := range tf.TeamSelection.Slots {
				entry.Attributes = e.player(entry.Attributes)
				tf.TeamSelection.Slots[k] = entry
			}
			for i := range tf.TeamSelection.NotSelected {
				tf.TeamSelection.NotSelected[i].Attributes = e.player(tf.TeamSelection.NotSelected[i].Attributes)
			}
		}
		out = append(out, tf)
	}
	return out, nil
}

// bestSetup is a team's best line-up and tactics in one formation, and how it
// does against the league.
type bestSetup struct {
	formation soccer.FormationType
	lineup    lineup
	tactics   tactics
	result    result
}

// bestIn finds a team's best line-up and tactics in a formation: the line-up
// under the manager's own tactics, then the tactics for that line-up, then
// the line-up again under them.
func bestIn(pool []player, fielded lineup, own tactics, f soccer.FormationType, opps []soccer.GameLineup, finalSeeds int) bestSetup {
	best := search(pool, fielded, f, own, opps)
	var tac []scored
	for _, t := range allTactics {
		tac = append(tac, scored{l: best, t: t})
	}
	evalAll(tac, opps, 2000, 16)
	tac = top(tac, 3)
	tac = append(tac, scored{l: best, t: own})
	evalAll(tac, opps, 3000, 60)
	final := tac[0]
	if final.t != own {
		final.l = search(pool, final.l, f, final.t, opps)
	}
	return bestSetup{
		formation: f, lineup: final.l, tactics: final.t,
		result: evaluatePar(final.l.gameLineup("MINE", final.t), opps, 5000, finalSeeds),
	}
}

// formationsOptions is what the formations report runs on.
type formationsOptions struct {
	dir        string
	model      soccer.Model
	enrich     *enricher // fill in dataset attributes for the squads as they are
	dealAssets string    // or: deal every squad afresh from dealDataset (see deal.go)
	dealData   string
	dealSeed   int64
	finalSeeds int
	seasons    int
	// lineupsPath caches step 1 (every team's best line-up per formation),
	// the slow part: written if the file doesn't exist, read if it does.
	// For tuning the engine's constants without searching again each time.
	lineupsPath string
}

// cachedSetup is a bestSetup as stored in the line-ups cache.
type cachedSetup struct {
	GK    string   `json:"gk"`
	Def   []string `json:"def"`
	Mid   []string `json:"mid"`
	Atk   []string `json:"atk"`
	Press string   `json:"press"`
	Tempo string   `json:"tempo"`
	Line  string   `json:"line"`
}

func loadLineups(path string, files []teamFile) ([]map[soccer.FormationType]bestSetup, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var cache map[string]map[string]cachedSetup
	if err := json.Unmarshal(raw, &cache); err != nil {
		return nil, false
	}
	out := make([]map[soccer.FormationType]bestSetup, len(files))
	for i, tf := range files {
		byID := map[string]player{}
		for _, p := range poolFromFile(tf) {
			byID[p.ID] = p
		}
		find := func(ids []string) []player {
			var ps []player
			for _, id := range ids {
				ps = append(ps, byID[id])
			}
			return ps
		}
		out[i] = map[soccer.FormationType]bestSetup{}
		for _, f := range formations {
			c, ok := cache[tf.Team.ID][string(f)]
			if !ok {
				return nil, false
			}
			out[i][f] = bestSetup{
				formation: f,
				lineup:    lineup{formation: f, gk: byID[c.GK], def: find(c.Def), mid: find(c.Mid), atk: find(c.Atk)},
				tactics:   tactics{soccer.PressLevel(c.Press), soccer.TempoLevel(c.Tempo), soccer.LineHeight(c.Line)},
			}
		}
	}
	return out, true
}

func saveLineups(path string, files []teamFile, setups []map[soccer.FormationType]bestSetup) error {
	ids := func(ps []player) []string {
		out := []string{}
		for _, p := range ps {
			out = append(out, p.ID)
		}
		return out
	}
	cache := map[string]map[string]cachedSetup{}
	for i, tf := range files {
		cache[tf.Team.ID] = map[string]cachedSetup{}
		for f, s := range setups[i] {
			cache[tf.Team.ID][string(f)] = cachedSetup{
				GK: s.lineup.gk.ID, Def: ids(s.lineup.def), Mid: ids(s.lineup.mid), Atk: ids(s.lineup.atk),
				Press: string(s.tactics.press), Tempo: string(s.tactics.tempo), Line: string(s.tactics.line),
			}
		}
	}
	raw, err := json.MarshalIndent(cache, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func formationsReport(o formationsOptions) error {
	formationsModel = o.model
	files, err := loadTeams(o.dir, o.enrich)
	if err != nil {
		return err
	}
	if len(files) < 2 {
		return fmt.Errorf("%s: need at least two team files", o.dir)
	}
	if o.dealAssets != "" {
		if err := deal(files, o.dealAssets, o.dealData, o.dealSeed); err != nil {
			return err
		}
	}
	n := len(files)

	// 1. Every team's best line-up and tactics in each formation, found
	// against the league as it is set up (or, under a fresh deal, against
	// everyone's best eligible players in a Diamond). Which players suit a
	// shape depends little on who the opponents are.
	fielded := make([]lineup, n)
	fieldedTactics := make([]tactics, n)
	field := make([]soccer.GameLineup, n)
	for i, tf := range files {
		fielded[i], fieldedTactics[i] = asConfigured(tf)
		field[i] = fielded[i].gameLineup(tf.Team.ID, fieldedTactics[i])
	}
	setups, cached := loadLineups(o.lineupsPath, files)
	if !cached {
		setups = make([]map[soccer.FormationType]bestSetup, n)
	}
	for i, tf := range files {
		if cached {
			break
		}
		var opps []soccer.GameLineup
		for j, gl := range field {
			if j != i {
				opps = append(opps, gl)
			}
		}
		pool := poolFromFile(tf)
		setups[i] = map[soccer.FormationType]bestSetup{}
		for _, f := range formations {
			setups[i][f] = bestIn(pool, fielded[i], fieldedTactics[i], f, opps, o.finalSeeds/2)
		}
		fmt.Fprintf(os.Stderr, "  %2d/%d %s\n", i+1, n, tf.Team.CustomName)
	}
	if !cached && o.lineupsPath != "" {
		if err := saveLineups(o.lineupsPath, files, setups); err != nil {
			return err
		}
	}
	lineupOf := func(i int, f soccer.FormationType) soccer.GameLineup {
		return setups[i][f].lineup.gameLineup(files[i].Team.ID, setups[i][f].tactics)
	}

	// 2. A shape's worth depends on what the rest of the league plays: one
	// that feasts on a league of Diamonds can be the worst choice against a
	// league that has switched to counter it. So each team in each shape is
	// measured against a mixed field: every opponent in every shape, equally.
	type cell struct{ i, f int }
	cells := make([]cell, 0, n*len(formations))
	for i := 0; i < n; i++ {
		for f := range formations {
			cells = append(cells, cell{i, f})
		}
	}
	crossSeeds := o.finalSeeds / 8
	if crossSeeds < 10 {
		crossSeeds = 10
	}
	mixed := make([][]result, n)
	for i := range mixed {
		mixed[i] = make([]result, len(formations))
	}
	// versus[cell][g] is that team-and-shape's record against shape g.
	versus := make([][]result, len(cells))
	parallelFor(len(cells), func(c int) {
		i, f := cells[c].i, cells[c].f
		mine := lineupOf(i, formations[f])
		versus[c] = make([]result, len(formations))
		for g := range formations {
			for j := 0; j < n; j++ {
				if j == i {
					continue
				}
				r := evaluate(mine, []soccer.GameLineup{lineupOf(j, formations[g])}, 7000, crossSeeds)
				versus[c][g].add(r)
			}
			mixed[i][f].add(versus[c][g])
		}
	})

	type teamResult struct {
		name  string
		index int
		plays soccer.FormationType
		best  int
	}
	teams := make([]teamResult, n)
	for i, tf := range files {
		t := teamResult{name: tf.Team.CustomName, index: i, plays: fielded[i].formation}
		for f := range formations {
			if mixed[i][f].ppg() > mixed[i][t.best].ppg() {
				t.best = f
			}
		}
		teams[i] = t
	}
	sort.SliceStable(teams, func(a, b int) bool {
		return mixed[teams[a].index][teams[a].best].ppg() > mixed[teams[b].index][teams[b].best].ppg()
	})
	gap := func(t teamResult, f int) float64 { return mixed[t.index][f].ppg() - mixed[t.index][t.best].ppg() }
	worstGap := func(t teamResult) float64 {
		w := 0.0
		for f := range formations {
			if g := gap(t, f); g < w {
				w = g
			}
		}
		return w
	}

	label := string(o.model)
	if label == "" {
		label = "classic"
	}
	short := func(f soccer.FormationType) string { return strings.TrimPrefix(string(f), "The ") }
	fmt.Printf("\n== formations on real squads, model %s: %d teams, each in its best line-up per formation, against a mixed field (%d games per cell)\n\n",
		label, n, crossSeeds*2*(n-1)*len(formations))
	fmt.Printf("%-24s %-8s", "team (strongest first)", "set up")
	for _, f := range formations {
		fmt.Printf(" %8s", short(f))
	}
	fmt.Printf("   %-8s %s\n", "best", "worst costs")
	for _, t := range teams {
		fmt.Printf("%-24.24s %-8s", t.name, short(t.plays))
		for f := range formations {
			fmt.Printf(" %8.2f", mixed[t.index][f].ppg())
		}
		fmt.Printf("   %-8s %+.2f\n", short(formations[t.best]), worstGap(t))
	}

	// Head to head: the row shape against the column shape, over every pair
	// of teams. A balanced engine has counters, but none so strong that the
	// choice is made for you.
	head := make([][]result, len(formations))
	for f := range head {
		head[f] = make([]result, len(formations))
	}
	for c, cl := range cells {
		for g := range formations {
			head[cl.f][g].add(versus[c][g])
		}
	}
	fmt.Printf("\nhead to head, points per game for the row shape against the column shape:\n%-10s", "")
	for _, g := range formations {
		fmt.Printf(" %8s", short(g))
	}
	fmt.Println()
	for f := range formations {
		fmt.Printf("%-10s", short(formations[f]))
		for g := range formations {
			fmt.Printf(" %8.2f", head[f][g].ppg())
		}
		fmt.Println()
	}

	pass := true
	check := func(ok bool, format string, args ...any) {
		mark := "ok  "
		if !ok {
			mark, pass = "MISS", false
		}
		fmt.Printf("  %s %s\n", mark, fmt.Sprintf(format, args...))
	}
	type summary struct {
		bestFor        int
		meanGap, worst float64
	}
	sums := make([]summary, len(formations))
	fmt.Printf("\n%-10s %9s %10s %10s %10s %11s\n", "formation", "mean PPG", "best for", "mean gap", "worst gap", "shot share")
	for f := range formations {
		var sm summary
		var ppg, shots float64
		for _, t := range teams {
			g := gap(t, f)
			sm.meanGap += g
			if g < sm.worst {
				sm.worst = g
			}
			if t.best == f {
				sm.bestFor++
			}
			r := mixed[t.index][f]
			ppg += r.ppg()
			shots += float64(r.Shots) / float64(r.Shots+r.ShotsAgn)
		}
		sm.meanGap /= float64(n)
		sums[f] = sm
		fmt.Printf("%-10s %9.2f %7d/%-2d %+10.2f %+10.2f %10.1f%%\n", short(formations[f]), ppg/float64(n), sm.bestFor, n, sm.meanGap, sm.worst, 100*shots/float64(n))
	}

	// The gate. Targets from the season 19 engine plan.
	wantBestFor := (n*3 + 19) / 20
	fmt.Printf("\ngate (against a mixed field):\n")
	for f := range formations {
		check(sums[f].bestFor >= wantBestFor, "%s is the best choice for %d teams (at least %d)", short(formations[f]), sums[f].bestFor, wantBestFor)
		// Never zero, even when the shapes are level: a team's best of four
		// is picked from four noisy numbers, and squads really do suit some
		// shapes better. Level shapes measure 0.05 to 0.10 here.
		check(sums[f].meanGap >= -0.105, "%s costs %.2f points per game on average (at most 0.10)", short(formations[f]), -sums[f].meanGap)
	}
	for f := range formations {
		for g := range formations {
			if edge := head[f][g].ppg() - head[g][f].ppg(); edge > 0.30 {
				check(false, "%s beats %s head to head by %.2f points per game (at most 0.30)", short(formations[f]), short(formations[g]), edge)
			}
		}
	}
	// The title race. A squad is dealt, so it suits some shapes better than
	// others and that is the point of having them: what must not happen is a
	// shape that no strong squad can use. Among the strongest six, the
	// shapes must do about as well as each other, and each must be the right
	// choice (or within a whisker of it) for at least one of them.
	topN := 6
	if n < topN {
		topN = n
	}
	topMean := make([]float64, len(formations))
	contender := make([]bool, len(formations))
	for _, t := range teams[:topN] {
		for f := range formations {
			topMean[f] += mixed[t.index][f].ppg() / float64(topN)
			if gap(t, f) >= -0.10 {
				contender[f] = true
			}
		}
	}
	lo, hi := topMean[0], topMean[0]
	for _, v := range topMean {
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}
	fmt.Printf("\ntop %d squads, mean points per game by shape:", topN)
	for f := range formations {
		fmt.Printf("  %s %.2f", short(formations[f]), topMean[f])
	}
	fmt.Println()
	check(hi-lo <= 0.15, "among the top %d squads the shapes are within %.2f points per game of each other (at most 0.15)", topN, hi-lo)
	for f := range formations {
		check(contender[f], "%s is the best choice, or within 0.10 of it, for at least one top-%d squad", short(formations[f]), topN)
	}

	// For information: the strongest side's title odds in each shape, with
	// everyone else in the shape that suits them best.
	if o.seasons > 0 {
		leader := teams[0]
		fmt.Printf("\n%s over %d seasons, by formation (the rest in their own best shape):\n", leader.name, o.seasons)
		league := make([]soccer.GameLineup, n)
		for _, t := range teams {
			league[t.index] = lineupOf(t.index, formations[t.best])
		}
		for f := range formations {
			league[leader.index] = lineupOf(leader.index, formations[f])
			title, avgPos := titleOdds(league, leader.index, o.seasons)
			fmt.Printf("  %-8s wins the league %5.1f%%, average finish %.1f\n", short(formations[f]), 100*title, avgPos)
		}
	}

	// As a manager would see it today: against the league as it is set up.
	if o.dealAssets == "" && !cached {
		counts := make([]int, len(formations))
		for i := 0; i < n; i++ {
			best := 0
			for f := range formations {
				if setups[i][formations[f]].result.ppg() > setups[i][formations[best]].result.ppg() {
					best = f
				}
			}
			counts[best]++
		}
		fmt.Printf("\nagainst the league as it is set up today, the best shape is:")
		for f := range formations {
			fmt.Printf("  %s for %d", short(formations[f]), counts[f])
		}
		fmt.Println()
	}

	tacticsOnSquads(files, setups)

	if pass {
		fmt.Printf("\nformations gate: PASS\n")
	} else {
		fmt.Printf("\nformations gate: MISS\n")
	}
	return nil
}

// add folds another record into r.
func (r *result) add(o result) {
	r.Games += o.Games
	r.Pts += o.Pts
	r.W += o.W
	r.D += o.D
	r.L += o.L
	r.GF += o.GF
	r.GA += o.GA
	r.Shots += o.Shots
	r.ShotsAgn += o.ShotsAgn
}

// titleOdds plays whole double round-robin seasons and returns how often team
// me finishes top, and its average finishing position.
func titleOdds(league []soccer.GameLineup, me, seasons int) (title, avgPos float64) {
	n := len(league)
	titles := make([]int, seasons)
	pos := make([]int, seasons)
	parallelFor(seasons, func(s int) {
		p := make([]int, n)
		gd := make([]int, n)
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				if i == j {
					continue
				}
				st := play(int64(s)*1000003+int64(i)*1009+int64(j), league[i], league[j])
				h, a := st.HomeTeamStats.Goals, st.AwayTeamStats.Goals
				gd[i] += h - a
				gd[j] += a - h
				switch {
				case h > a:
					p[i] += 3
				case h < a:
					p[j] += 3
				default:
					p[i]++
					p[j]++
				}
			}
		}
		rank := 1
		for k := 0; k < n; k++ {
			if k != me && (p[k] > p[me] || (p[k] == p[me] && gd[k] > gd[me])) {
				rank++
			}
		}
		pos[s] = rank
		if rank == 1 {
			titles[s] = 1
		}
	})
	var t, ps int
	for s := 0; s < seasons; s++ {
		t += titles[s]
		ps += pos[s]
	}
	return float64(t) / float64(seasons), float64(ps) / float64(seasons)
}

// tacticsOnSquads reports what each tactic setting is worth on these squads.
// Every team's best line-up in each shape plays every other team's, in every
// shape, with everyone on the default settings; then again with one setting
// changed, on the same seeds. A setting that is a real choice is worth about
// nothing on average and something, one way or the other, to particular
// squads. One that is worth the same to everybody is not a choice. (The
// tactics report asks the same of invented squads; this is the real ones.)
func tacticsOnSquads(files []teamFile, setups []map[soccer.FormationType]bestSetup) {
	changes := []struct {
		name string
		t    tactics
	}{
		{"press low", tactics{press: soccer.PressLevelLow}},
		{"press high", tactics{press: soccer.PressLevelHigh}},
		{"tempo slow", tactics{tempo: soccer.TempoLevelSlow}},
		{"tempo fast", tactics{tempo: soccer.TempoLevelFast}},
		{"line deep", tactics{line: soccer.LineHeightDeep}},
		{"line high", tactics{line: soccer.LineHeightHigh}},
	}
	type cell struct{ i, f int }
	var cells []cell
	for i := range files {
		for f := range formations {
			cells = append(cells, cell{i, f})
		}
	}
	deltas := make([][]float64, len(cells))
	parallelFor(len(cells), func(c int) {
		i, f := cells[c].i, formations[cells[c].f]
		var opps []soccer.GameLineup
		for j := range files {
			if j == i {
				continue
			}
			for _, g := range formations {
				opps = append(opps, setups[j][g].lineup.gameLineup(files[j].Team.ID, tactics{}))
			}
		}
		const seeds = 8
		base := evaluate(setups[i][f].lineup.gameLineup("MINE", tactics{}), opps, 9000, seeds).ppg()
		deltas[c] = make([]float64, len(changes))
		for k, ch := range changes {
			deltas[c][k] = evaluate(setups[i][f].lineup.gameLineup("MINE", ch.t), opps, 9000, seeds).ppg() - base
		}
	})

	fmt.Printf("\ntactics on these squads: points per game for changing one setting from the default, each team's best line-up in each shape against a mixed field\n")
	fmt.Printf("%-18s", "")
	for _, ch := range changes {
		fmt.Printf(" %11s", ch.name)
	}
	fmt.Println()
	row := func(name string, keep func(cell) bool) {
		fmt.Printf("%-18s", name)
		for k := range changes {
			var sum float64
			count := 0
			for c := range cells {
				if keep(cells[c]) {
					sum += deltas[c][k]
					count++
				}
			}
			fmt.Printf(" %+11.3f", sum/float64(count))
		}
		fmt.Println()
	}
	row("every set-up", func(cell) bool { return true })
	for f := range formations {
		row("  "+strings.TrimPrefix(string(formations[f]), "The "), func(c cell) bool { return c.f == f })
	}
	// How far it depends on the squad: the set-ups it suits least and most.
	fmt.Printf("%-18s", "  suits least")
	spread := make([][]float64, len(changes))
	for k := range changes {
		for c := range cells {
			spread[k] = append(spread[k], deltas[c][k])
		}
		sort.Float64s(spread[k])
		fmt.Printf(" %+11.3f", spread[k][len(spread[k])/10])
	}
	fmt.Printf("\n%-18s", "  suits most")
	for k := range changes {
		fmt.Printf(" %+11.3f", spread[k][len(spread[k])*9/10])
	}
	fmt.Printf("\n(least and most are the 10th and 90th of %d set-ups; one set-up's figure is good to about 0.05)\n", len(cells))
}
