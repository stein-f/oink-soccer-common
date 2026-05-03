// Package allocation assigns FIFA players to in-game NFT assets at the
// start of each season. The assignment is a deterministic function of
// (seed, pool, assets) — the same Algorand block hash always produces the
// same allocation so it can be re-verified on-chain.
//
// v1 mixed allocation logic, CSV I/O, and a config-file loader inside
// cmd/allocation. v2 splits them: this package contains the pure
// allocation core; the CLI tool that loads CSVs lives separately and
// just calls Allocate.
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
	AssetTierAggressive AssetTier = "Tier Aggressive" // special pool of high-aggression players
)

// Asset is an in-game NFT asset that needs a player assignment.
type Asset struct {
	ID   string
	Name string
	Tier AssetTier
}

// Candidate is a real-world player profile drawn from the FIFA dataset.
// Allocations pull from a pool of these.
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

// Rules govern the allocation logic. The zero value uses sane defaults
// matching v1 behavior; pass a non-zero value to override specific dials
// for season-specific allocation tweaks.
type Rules struct {
	// PositionWeights controls the chance of each position being rolled
	// for an asset. Defaults to {GK:15, DEF:20, MID:20, ATK:20}.
	PositionWeights map[soccer.PlayerPosition]uint

	// TierLevelDistribution maps each asset tier to a level probability
	// distribution. Defaults to v1's table — higher tiers are more
	// likely to roll Legendary / World Class players.
	TierLevelDistribution map[AssetTier]map[soccer.PlayerLevel]uint

	// AggressionMinimum is the lower bound for a player to qualify for
	// the Aggressive pool. Defaults to 80.
	AggressionMinimum int

	// AggressionOverallMax is the upper bound on overall rating for
	// Aggressive-pool players. Defaults to 86.
	AggressionOverallMax int
}

// DefaultRules returns the v1-compatible rules.
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

var defaultTierLevelDistribution = map[AssetTier]map[soccer.PlayerLevel]uint{
	AssetTierS: {
		soccer.PlayerLevelLegendary:    3,
		soccer.PlayerLevelWorldClass:   45,
		soccer.PlayerLevelProfessional: 52,
	},
	AssetTierA: {
		soccer.PlayerLevelLegendary:    3,
		soccer.PlayerLevelWorldClass:   37,
		soccer.PlayerLevelProfessional: 60,
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

// Pool holds candidates indexed for fast lookup by (position, level) and a
// separate aggressive list. Build with NewPool; Allocate consumes one.
type Pool struct {
	byPosLevel map[poolKey][]Candidate
	aggressive []Candidate
	rules      Rules
	allTiers   []soccer.PlayerLevel // sorted tier keys for deterministic iteration
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
		rules:      rules,
	}
	for _, c := range candidates {
		levels := levelsFor(c.Attributes.OverallRating)
		for _, lvl := range levels {
			pos := c.Attributes.PrimaryPosition
			if pos == "" || pos == soccer.PlayerPositionAny {
				continue
			}
			key := poolKey{Position: pos, Level: lvl}
			p.byPosLevel[key] = append(p.byPosLevel[key], c)
		}
		if c.Attributes.AggressionRating >= rules.AggressionMinimum &&
			c.Attributes.OverallRating <= rules.AggressionOverallMax {
			p.aggressive = append(p.aggressive, c)
		}
	}
	// Normalise within each bucket: sort by (overall desc, id asc).
	sortFn := func(s []Candidate) {
		sort.Slice(s, func(i, j int) bool {
			if s[i].Attributes.OverallRating != s[j].Attributes.OverallRating {
				return s[i].Attributes.OverallRating > s[j].Attributes.OverallRating
			}
			return s[i].ID < s[j].ID
		})
	}
	for k := range p.byPosLevel {
		sortFn(p.byPosLevel[k])
	}
	sortFn(p.aggressive)
	// Pin a stable iteration order over the level keys.
	for lvl := range PlayerLevelBands {
		p.allTiers = append(p.allTiers, lvl)
	}
	sort.Slice(p.allTiers, func(i, j int) bool { return p.allTiers[i] < p.allTiers[j] })
	return p
}

// Allocate assigns one Assignment per Asset using the supplied random
// source. The returned slice has the same length and order as `assets`.
//
// Determinism: same (seed, candidates, rules, assets) ⇒ same output.
func Allocate(r *rand.Rand, pool *Pool, assets []Asset) ([]Assignment, error) {
	if r == nil {
		return nil, errors.New("allocation: rand source is required")
	}
	if pool == nil {
		return nil, errors.New("allocation: pool is required")
	}
	out := make([]Assignment, 0, len(assets))
	for _, asset := range assets {
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

func (p *Pool) pickCandidate(r *rand.Rand, tier AssetTier, pos soccer.PlayerPosition) (Candidate, error) {
	if tier == AssetTierAggressive {
		if len(p.aggressive) == 0 {
			return Candidate{}, ErrEmptyPool
		}
		return p.aggressive[r.Intn(len(p.aggressive))], nil
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
// pools — kept for v1 parity.
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
