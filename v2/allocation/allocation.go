// Package allocation assigns FIFA players to in-game NFT assets at the
// start of each season. The assignment is a deterministic function of
// (seed, pool, assets) — the same Algorand block hash always produces the
// same allocation so it can be re-verified on-chain.
//
// v1 mixed allocation logic, CSV I/O, and a config-file loader inside
// cmd/allocation. v2 splits them: this package contains the pure
// allocation core; the CLI tool that loads CSVs lives in cmd/allocation
// (of the v2 module) and just calls Allocate.
//
// Three special mechanisms sit alongside the plain tier→level draw:
//
//   - Aggressive pool (Tier Aggressive, the SCHIZO collection): candidates
//     with AggressionRating ≥ AggressionMinimum and OverallRating ≤
//     AggressionOverallMax, picked position-blind.
//   - Specialist pools (Tier Specialist, the Schnoz collection): position-
//     respecting pools filtered on the position's signature specialist
//     attribute (ATK→Finishing, MID→Technique, DEF→Tackling; GK unfiltered).
//     The level distribution enforces the published pig:schnoz ratio: World
//     Class odds are Tier S's 49% ÷ 10 = 4.9%.
//   - Legend lottery: curated legend cards (PlayerLevel == Legendary) are
//     seeded exactly once each onto weighted host assets before the normal
//     draw. Tier S/A host at weight 10, Tier Specialist at weight 1 — "a
//     pig is exactly 10× more likely to land a legend than a schnoz". No
//     other tier can host a legend.
//
// Real players whose overall reaches the legendary band (87+) are indexed
// under the Legendary level only, which no tier distribution draws (weight
// 0 everywhere). They are deliberately unreachable — Legendary content is
// exclusively the curated legend cards. This mirrors shipped v1 behavior.
package allocation

import (
	"errors"
	"math/rand"
	"sort"

	soccer "github.com/stein-f/oink-soccer-common/v2"
)

// AssetTier is the rarity tier of an NFT asset. Each tier draws from a
// different distribution of player levels.
type AssetTier string

const (
	AssetTierS          AssetTier = "Tier S"
	AssetTierA          AssetTier = "Tier A"
	AssetTierB          AssetTier = "Tier B"
	AssetTierC          AssetTier = "Tier C"
	AssetTierAggressive AssetTier = "Tier Aggressive" // special pool of high-aggression players (SCHIZO)
	AssetTierSpecialist AssetTier = "Tier Specialist" // position-signature specialist pools (Schnoz)
)

// Asset is an in-game NFT asset that needs a player assignment.
type Asset struct {
	ID   string
	Name string
	Tier AssetTier
}

// Candidate is a real-world player profile drawn from the FIFA dataset.
// Allocations pull from a pool of these. Candidates whose PlayerLevel is
// Legendary are treated as curated legend cards: they never enter the
// random pools and are instead seeded exactly once each via the legend
// lottery.
type Candidate struct {
	ID         string
	Name       string
	Attributes soccer.PlayerAttributes
}

// Assignment is the result of one allocation: an asset, the position it
// was rolled to play, and the candidate filling it.
type Assignment struct {
	Asset    Asset
	Position soccer.PlayerPosition
	Player   Candidate
}

// SpecialistStat names the attribute a specialist pool filters on. Values
// resolve through the same Effective* accessors the engine uses, so
// candidates without explicit specialist data fall back to their composite
// rating exactly as they would in a match.
type SpecialistStat string

const (
	SpecialistStatFinishing SpecialistStat = "Finishing"
	SpecialistStatTechnique SpecialistStat = "Technique"
	SpecialistStatTackling  SpecialistStat = "Tackling"
)

// SpecialistFilter is the entry requirement for one position's specialist
// pool.
type SpecialistFilter struct {
	Stat    SpecialistStat
	Minimum int
}

// Rules govern the allocation logic. The zero value uses sane defaults
// matching shipped behavior; pass a non-zero value to override specific
// dials for season-specific allocation tweaks.
type Rules struct {
	// PositionWeights controls the chance of each position being rolled
	// for an asset. Defaults to {GK:15, DEF:20, MID:20, ATK:20}.
	PositionWeights map[soccer.PlayerPosition]uint

	// TierLevelDistribution maps each asset tier to a level probability
	// distribution. Defaults to the shipped table — higher tiers are more
	// likely to roll World Class players. Legendary is absent everywhere:
	// legend cards are seeded exactly once each by the legend lottery, so
	// the random draw must never surface one.
	TierLevelDistribution map[AssetTier]map[soccer.PlayerLevel]uint

	// AggressionMinimum is the lower bound for a player to qualify for
	// the Aggressive pool. Defaults to 80.
	AggressionMinimum int

	// AggressionOverallMax is the upper bound on overall rating for
	// Aggressive-pool players. Defaults to 86.
	AggressionOverallMax int

	// SpecialistFilters maps a position to the entry requirement for its
	// specialist pool. Positions absent from the map (goalkeepers by
	// default) draw unfiltered from their position bucket. Defaults to
	// ATK→Finishing≥78, MID→Technique≥78, DEF→Tackling≥78.
	SpecialistFilters map[soccer.PlayerPosition]SpecialistFilter

	// SpecialistLevelDistribution is the level distribution for Tier
	// Specialist assets. Defaults to {World Class: 49, Professional: 951}
	// — World Class odds are exactly one tenth of Tier S's 49%, the
	// published "1 pig = 10 schnoz" ratio. The World Class band's 86
	// ceiling doubles as the specialist overall cap.
	SpecialistLevelDistribution map[soccer.PlayerLevel]uint

	// LegendHostWeights is the per-tier weight in the legend lottery.
	// Tiers absent from the map can never host a legend card. Defaults to
	// {Tier S: 10, Tier A: 10, Tier Specialist: 1}.
	LegendHostWeights map[AssetTier]uint
}

// DefaultRules returns the shipped-production rules.
func DefaultRules() Rules {
	return Rules{
		PositionWeights: map[soccer.PlayerPosition]uint{
			soccer.PlayerPositionGoalkeeper: 15,
			soccer.PlayerPositionDefense:    20,
			soccer.PlayerPositionMidfield:   20,
			soccer.PlayerPositionAttack:     20,
		},
		TierLevelDistribution: defaultTierLevelDistribution,
		AggressionMinimum:     80,
		AggressionOverallMax:  86,
		SpecialistFilters: map[soccer.PlayerPosition]SpecialistFilter{
			soccer.PlayerPositionAttack:   {Stat: SpecialistStatFinishing, Minimum: 78},
			soccer.PlayerPositionMidfield: {Stat: SpecialistStatTechnique, Minimum: 78},
			soccer.PlayerPositionDefense:  {Stat: SpecialistStatTackling, Minimum: 78},
		},
		SpecialistLevelDistribution: map[soccer.PlayerLevel]uint{
			soccer.PlayerLevelWorldClass:   49,
			soccer.PlayerLevelProfessional: 951,
		},
		LegendHostWeights: map[AssetTier]uint{
			AssetTierS:          10,
			AssetTierA:          10,
			AssetTierSpecialist: 1,
		},
	}
}

// PlayerLevelBands maps each level to its [min, max] overall-rating band.
var PlayerLevelBands = map[soccer.PlayerLevel][2]int{
	soccer.PlayerLevelLegendary:        {87, 100},
	soccer.PlayerLevelWorldClass:       {80, 86},
	soccer.PlayerLevelProfessional:     {70, 79},
	soccer.PlayerLevelSemiProfessional: {55, 69},
	soccer.PlayerLevelAmateur:          {0, 54},
}

// defaultTierLevelDistribution is the shipped table. Legendary is absent
// from every row — see Rules.TierLevelDistribution.
var defaultTierLevelDistribution = map[AssetTier]map[soccer.PlayerLevel]uint{
	AssetTierS: {
		soccer.PlayerLevelWorldClass:   49,
		soccer.PlayerLevelProfessional: 51,
	},
	AssetTierA: {
		soccer.PlayerLevelWorldClass:   41,
		soccer.PlayerLevelProfessional: 59,
	},
	AssetTierB: {
		soccer.PlayerLevelWorldClass:       8,
		soccer.PlayerLevelProfessional:     42,
		soccer.PlayerLevelSemiProfessional: 40,
		soccer.PlayerLevelAmateur:          10,
	},
	AssetTierC: {
		soccer.PlayerLevelWorldClass:       4,
		soccer.PlayerLevelProfessional:     30,
		soccer.PlayerLevelSemiProfessional: 46,
		soccer.PlayerLevelAmateur:          20,
	},
}

// ErrEmptyPool is returned when allocation runs out of candidates for some
// (position, level) combination — usually a sign that the input dataset is
// too small or the rules are misconfigured.
var ErrEmptyPool = errors.New("allocation: no candidates for requested position/level")

// LegendCardPrefix marks the curated legend rows in the FIFA dataset: their
// sofifa_id (and therefore Candidate.ID) is "L0001", "L0002", … Only these
// may carry PlayerLevel Legendary and enter the legend lottery. Real
// players whose overall reaches the legendary band are presentationally
// capped at World Class by the data loader.
const LegendCardPrefix = "L"

// Pool holds candidates indexed for fast lookup by (position, level), a
// separate aggressive list, position-respecting specialist pools, and the
// curated legend cards. Build with NewPool; Allocate consumes one.
type Pool struct {
	byPosLevel map[poolKey][]Candidate
	specialist map[poolKey][]Candidate
	aggressive []Candidate
	legends    []Candidate
	rules      Rules
}

type poolKey struct {
	Position soccer.PlayerPosition
	Level    soccer.PlayerLevel
}

// NewPool builds a Pool from a flat candidate list using the supplied
// rules. The input slice doesn't need to be sorted; NewPool normalises
// internally so allocation is deterministic for the same inputs.
func NewPool(candidates []Candidate, rules Rules) *Pool {
	p := &Pool{
		byPosLevel: make(map[poolKey][]Candidate),
		specialist: make(map[poolKey][]Candidate),
		rules:      rules,
	}
	for _, c := range candidates {
		// Curated legend cards never enter the random pools — the legend
		// lottery seeds each exactly once.
		if c.Attributes.PlayerLevel == soccer.PlayerLevelLegendary {
			p.legends = append(p.legends, c)
			continue
		}
		pos := c.Attributes.PrimaryPosition
		if pos == "" || pos == soccer.PlayerPositionAny {
			continue
		}
		levels := levelsFor(c.Attributes.OverallRating)
		for _, lvl := range levels {
			key := poolKey{Position: pos, Level: lvl}
			p.byPosLevel[key] = append(p.byPosLevel[key], c)
			if passesSpecialistFilter(c, pos, rules) {
				p.specialist[key] = append(p.specialist[key], c)
			}
		}
		if c.Attributes.AggressionRating >= rules.AggressionMinimum &&
			c.Attributes.OverallRating <= rules.AggressionOverallMax {
			p.aggressive = append(p.aggressive, c)
		}
	}
	// Normalise within each bucket: sort by (overall desc, id asc).
	for k := range p.byPosLevel {
		sortCandidates(p.byPosLevel[k])
	}
	for k := range p.specialist {
		sortCandidates(p.specialist[k])
	}
	sortCandidates(p.aggressive)
	sortCandidates(p.legends)
	return p
}

func sortCandidates(s []Candidate) {
	sort.Slice(s, func(i, j int) bool {
		if s[i].Attributes.OverallRating != s[j].Attributes.OverallRating {
			return s[i].Attributes.OverallRating > s[j].Attributes.OverallRating
		}
		return s[i].ID < s[j].ID
	})
}

// passesSpecialistFilter reports whether a candidate qualifies for its
// position's specialist pool. Positions without a configured filter
// (goalkeepers by default) accept every candidate in the position bucket.
func passesSpecialistFilter(c Candidate, pos soccer.PlayerPosition, rules Rules) bool {
	filter, ok := rules.SpecialistFilters[pos]
	if !ok {
		return true
	}
	return specialistValue(c.Attributes, filter.Stat) >= filter.Minimum
}

// specialistValue resolves a specialist stat through the same Effective*
// accessors the engine uses at match time.
func specialistValue(attrs soccer.PlayerAttributes, stat SpecialistStat) int {
	switch stat {
	case SpecialistStatFinishing:
		return attrs.EffectiveFinishing()
	case SpecialistStatTechnique:
		return attrs.EffectiveTechnique()
	case SpecialistStatTackling:
		return attrs.EffectiveTackling()
	}
	return 0
}

// Allocate assigns one Assignment per Asset using the supplied random
// source. The returned slice has the same length and order as `assets`.
//
// Legend cards are seeded first: each legend is paired with a distinct
// host asset drawn by the weighted lottery (Rules.LegendHostWeights), then
// every remaining asset is allocated normally.
//
// Determinism: same (seed, candidates, rules, assets) ⇒ same output.
func Allocate(r *rand.Rand, pool *Pool, assets []Asset) ([]Assignment, error) {
	if r == nil {
		return nil, errors.New("allocation: rand source is required")
	}
	if pool == nil {
		return nil, errors.New("allocation: pool is required")
	}
	seeded := seedLegends(r, pool, assets)

	out := make([]Assignment, 0, len(assets))
	for i, asset := range assets {
		if legend, ok := seeded[i]; ok {
			out = append(out, Assignment{
				Asset:    asset,
				Position: legend.Attributes.PrimaryPosition,
				Player:   legend,
			})
			continue
		}
		pos, err := rollPosition(r, pool.rules)
		if err != nil {
			return nil, err
		}
		cand, err := pool.pickCandidate(r, asset.Tier, pos)
		if err != nil {
			return nil, err
		}
		out = append(out, Assignment{Asset: asset, Position: pos, Player: cand})
	}
	return out, nil
}

// seedLegends pairs each legend card with a distinct host asset via a
// weighted sample without replacement. Hosts are drawn proportionally to
// their tier's LegendHostWeights entry; tiers without an entry never host.
// Returns a map of asset index → legend. If hosts run out, the remaining
// legends are simply not seeded this season.
func seedLegends(r *rand.Rand, pool *Pool, assets []Asset) map[int]Candidate {
	if len(pool.legends) == 0 {
		return nil
	}
	type host struct {
		index  int
		weight uint
	}
	var hosts []host
	for i, a := range assets {
		w := pool.rules.LegendHostWeights[a.Tier]
		if w > 0 {
			hosts = append(hosts, host{index: i, weight: w})
		}
	}
	seeded := make(map[int]Candidate, len(pool.legends))
	for _, legend := range pool.legends {
		if len(hosts) == 0 {
			break
		}
		var total uint
		for _, h := range hosts {
			total += h.weight
		}
		pick := uint(r.Intn(int(total)))
		var cum uint
		chosen := len(hosts) - 1
		for hi, h := range hosts {
			cum += h.weight
			if pick < cum {
				chosen = hi
				break
			}
		}
		seeded[hosts[chosen].index] = legend
		hosts = append(hosts[:chosen], hosts[chosen+1:]...)
	}
	return seeded
}

func (p *Pool) pickCandidate(r *rand.Rand, tier AssetTier, pos soccer.PlayerPosition) (Candidate, error) {
	switch tier {
	case AssetTierAggressive:
		// Position-blind by design (v1 parity): the asset takes whatever
		// position the aggressive player naturally plays.
		if len(p.aggressive) == 0 {
			return Candidate{}, ErrEmptyPool
		}
		return p.aggressive[r.Intn(len(p.aggressive))], nil
	case AssetTierSpecialist:
		level := pickLevel(r, p.rules.SpecialistLevelDistribution)
		candidates := p.specialist[poolKey{Position: pos, Level: level}]
		if len(candidates) == 0 {
			return Candidate{}, ErrEmptyPool
		}
		return candidates[r.Intn(len(candidates))], nil
	}
	dist, ok := p.rules.TierLevelDistribution[tier]
	if !ok {
		return Candidate{}, errors.New("allocation: unknown asset tier " + string(tier))
	}
	level := pickLevel(r, dist)
	candidates := p.byPosLevel[poolKey{Position: pos, Level: level}]
	if len(candidates) == 0 {
		return Candidate{}, ErrEmptyPool
	}
	return candidates[r.Intn(len(candidates))], nil
}

// rollPosition picks a position weighted by the rules' PositionWeights.
func rollPosition(r *rand.Rand, rules Rules) (soccer.PlayerPosition, error) {
	w := rules.PositionWeights
	if len(w) == 0 {
		return "", errors.New("allocation: no position weights configured")
	}
	keys := make([]soccer.PlayerPosition, 0, len(w))
	for k := range w {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	var total uint
	for _, k := range keys {
		total += w[k]
	}
	if total == 0 {
		return "", errors.New("allocation: position weights sum to zero")
	}
	pick := uint(r.Intn(int(total)))
	var cum uint
	for _, k := range keys {
		cum += w[k]
		if pick < cum {
			return k, nil
		}
	}
	return keys[len(keys)-1], nil
}

func pickLevel(r *rand.Rand, dist map[soccer.PlayerLevel]uint) soccer.PlayerLevel {
	keys := make([]soccer.PlayerLevel, 0, len(dist))
	for k := range dist {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })

	var total uint
	for _, k := range keys {
		total += dist[k]
	}
	if total == 0 {
		return soccer.PlayerLevelProfessional
	}
	pick := uint(r.Intn(int(total)))
	var cum uint
	for _, k := range keys {
		cum += dist[k]
		if pick < cum {
			return k
		}
	}
	return keys[len(keys)-1]
}

// levelsFor returns every level whose band contains overall. Players that
// straddle band boundaries (rare with fixed bands) appear in multiple
// pools — kept for v1 parity. Real (non-legend-card) players in the 87+
// band land only in the Legendary bucket, which no distribution draws —
// they are deliberately out of the game.
func levelsFor(overall int) []soccer.PlayerLevel {
	var out []soccer.PlayerLevel
	for _, lvl := range sortedLevels() {
		band := PlayerLevelBands[lvl]
		if overall >= band[0] && overall <= band[1] {
			out = append(out, lvl)
		}
	}
	return out
}

func sortedLevels() []soccer.PlayerLevel {
	out := make([]soccer.PlayerLevel, 0, len(PlayerLevelBands))
	for lvl := range PlayerLevelBands {
		out = append(out, lvl)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
