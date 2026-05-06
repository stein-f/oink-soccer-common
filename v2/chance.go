package soccer

import (
	"math"
	"math/rand"
	"sort"
)

// chanceTypeProfile shapes how a particular kind of chance plays out:
//   - BaseWeight is its frequency in chance-type rolls
//   - PositionWeights override the default attacker-pick weights for this type
//   - AttackBoost multiplies the attacker's effective attack score
//   - DefenseScale multiplies the defender's effective defense score
//   - AttackScore computes the attacker's raw score from their attributes —
//     each chance type defines its own formula referencing the appropriate
//     specialist attribute (Heading for corners, Technique for long range,
//     Composure for penalties, etc.). Nil ⇒ use the default open-play
//     formula via defaultAttackScore.
//
// E.g. a Penalty has high AttackBoost + low DefenseScale (most defenders
// don't matter; conversion is high) and an AttackScore that pairs the
// player's AttackRating with their Composure (no pace). A LongRange shot
// has low AttackBoost + high DefenseScale (the shot is harder) and
// AttackScore that pairs AttackRating with Technique.
type chanceTypeProfile struct {
	BaseWeight      uint
	PositionWeights map[PlayerPosition]uint
	AttackBoost     float64
	DefenseScale    float64
	AttackScore     func(p PlayerAttributes) float64
}

// defaultAttackScore is v1's (skill*3 + pace*1) / 4 formula. Used for chance
// types that don't declare their own AttackScore.
func defaultAttackScore(p PlayerAttributes) float64 {
	return weightedScore(p.AttackRating*3+p.SpeedRating*1, 4)
}

// weightedScore divides a numerator by a divisor and rounds — mirrors the
// behaviour of the lower-level weighted() helper used elsewhere so attribute
// scores stay integer-valued.
func weightedScore(numerator, divisor int) float64 {
	if divisor <= 0 {
		return 0
	}
	return math.Round(float64(numerator) / float64(divisor))
}

// defaultPositionPickWeights are used when a chance type doesn't override.
// Heavy favouritism toward attackers + midfielders.
var defaultPositionPickWeights = map[PlayerPosition]uint{
	PlayerPositionGoalkeeper: 2,
	PlayerPositionDefense:    10,
	PlayerPositionMidfield:   20,
	PlayerPositionAttack:     70,
}

// chanceTypeProfiles is the central table of chance behaviors. Tuning these
// is the primary lever for making each chance type "feel" distinct in the
// match commentary that lost-pigs renders.
//
// Each chance type's AttackScore formula is what gives different player
// builds purpose:
//   - Open play pairs AttackRating + Finishing + Pace (well-rounded striker)
//   - Crosses pair AttackRating + Heading + Pace (target man on the run)
//   - Corners pair AttackRating + Heading (aerial specialist; no pace)
//   - Long range pairs AttackRating + Technique (technique-driven shot)
//   - Free kicks pair AttackRating + Technique (set-piece specialist)
//   - Penalties pair AttackRating + Composure (clutch finisher; no pace)
//   - 1-on-1 breakaways pair AttackRating + Finishing + Pace, pace-heavy
//
// Net effect: a target man (high heading, low pace) is the corner specialist,
// a clutch finisher (high composure) tops penalties, a technical midfielder
// owns long range and free kicks, and a speedster runs the channels.
var chanceTypeProfiles = map[ChanceType]chanceTypeProfile{
	ChanceTypeOpenPlay: {
		BaseWeight:   8,
		AttackBoost:  1.00,
		DefenseScale: 1.00,
		AttackScore: func(p PlayerAttributes) float64 {
			// (atk*2 + finishing + pace) / 4 — well-rounded forward play.
			return weightedScore(p.AttackRating*2+p.EffectiveFinishing()+p.SpeedRating, 4)
		},
	},
	ChanceTypeCross: {
		BaseWeight: 5,
		PositionWeights: map[PlayerPosition]uint{
			PlayerPositionDefense:  5,
			PlayerPositionMidfield: 25,
			PlayerPositionAttack:   70,
		},
		AttackBoost:  0.95,
		DefenseScale: 1.05,
		AttackScore: func(p PlayerAttributes) float64 {
			// (atk*2 + heading*2 + pace) / 5 — striker arriving on a delivery.
			return weightedScore(p.AttackRating*2+p.EffectiveHeading()*2+p.SpeedRating, 5)
		},
	},
	ChanceTypeCorner: {
		BaseWeight: 3,
		PositionWeights: map[PlayerPosition]uint{
			PlayerPositionDefense:  15, // defenders go up for corners
			PlayerPositionMidfield: 25,
			PlayerPositionAttack:   60,
		},
		AttackBoost:  0.90,
		DefenseScale: 1.10,
		AttackScore: func(p PlayerAttributes) float64 {
			// (atk*2 + heading*3) / 5 — pure aerial duel; pace irrelevant.
			return weightedScore(p.AttackRating*2+p.EffectiveHeading()*3, 5)
		},
	},
	ChanceTypeLongRange: {
		BaseWeight: 3,
		PositionWeights: map[PlayerPosition]uint{
			PlayerPositionDefense:  10,
			PlayerPositionMidfield: 50, // mids fire from distance
			PlayerPositionAttack:   40,
		},
		AttackBoost:  0.70,
		DefenseScale: 1.20,
		AttackScore: func(p PlayerAttributes) float64 {
			// (atk*2 + technique*3) / 5 — technique-driven strike.
			return weightedScore(p.AttackRating*2+p.EffectiveTechnique()*3, 5)
		},
	},
	ChanceTypeFreeKick: {
		BaseWeight: 3,
		PositionWeights: map[PlayerPosition]uint{
			PlayerPositionDefense:  10,
			PlayerPositionMidfield: 45,
			PlayerPositionAttack:   45,
		},
		AttackBoost:  0.85,
		DefenseScale: 1.10,
		AttackScore: func(p PlayerAttributes) float64 {
			// (atk + technique*3) / 4 — set-piece technique dominates.
			return weightedScore(p.AttackRating+p.EffectiveTechnique()*3, 4)
		},
	},
	ChanceTypePenalty: {
		BaseWeight: 2,
		PositionWeights: map[PlayerPosition]uint{
			PlayerPositionDefense:  5,
			PlayerPositionMidfield: 25,
			PlayerPositionAttack:   70,
		},
		AttackBoost:  1.50, // penalties have very high conversion
		DefenseScale: 0.50,
		AttackScore: func(p PlayerAttributes) float64 {
			// (atk*2 + composure*3) / 5 — clutch finisher under pressure.
			return weightedScore(p.AttackRating*2+p.EffectiveComposure()*3, 5)
		},
	},
	ChanceTypeGoalKeeperShot: {
		BaseWeight: 2,
		PositionWeights: map[PlayerPosition]uint{
			PlayerPositionMidfield: 20,
			PlayerPositionAttack:   80,
		},
		AttackBoost:  1.20, // 1-on-1 / breakaway
		DefenseScale: 0.70,
		AttackScore: func(p PlayerAttributes) float64 {
			// (atk + finishing + pace*3) / 5 — speed wins the chase, then convert.
			return weightedScore(p.AttackRating+p.EffectiveFinishing()+p.SpeedRating*3, 5)
		},
	},
}

// chanceTypeOrder pins iteration order for deterministic behaviour. Map
// iteration in Go is randomised, so any rolling that depends on order must
// use this slice instead.
var chanceTypeOrder = []ChanceType{
	ChanceTypeOpenPlay,
	ChanceTypeCross,
	ChanceTypeCorner,
	ChanceTypeLongRange,
	ChanceTypeFreeKick,
	ChanceTypePenalty,
	ChanceTypeGoalKeeperShot,
}

// pickChanceType samples a chance type via weighted random, banning the
// previous chance type to avoid back-to-back duplicates that look weird in
// commentary ("CORNER. CORNER. CORNER.").
func pickChanceType(rand *rand.Rand, previous ChanceType) ChanceType {
	var totalW uint
	weights := make([]uint, len(chanceTypeOrder))
	for i, ct := range chanceTypeOrder {
		w := chanceTypeProfiles[ct].BaseWeight
		if ct == previous {
			w = 0
		}
		weights[i] = w
		totalW += w
	}
	if totalW == 0 {
		return ChanceTypeOpenPlay
	}
	pick := uint(rand.Intn(int(totalW)))
	var cum uint
	for i, w := range weights {
		cum += w
		if pick < cum {
			return chanceTypeOrder[i]
		}
	}
	return chanceTypeOrder[len(chanceTypeOrder)-1]
}

// pickAttacker chooses which player on the attacking team will receive the
// chance. Weighted by position (with chance-type overrides applied) and by
// the player's effective attack score.
//
// The score-weighting is mild — we want the structural distribution
// (attackers more likely than mids more likely than defenders) to dominate,
// but a 95-rated attacker should be more likely to get the chance than a
// 70-rated one in the same position.
//
// excludeID, if non-empty, removes the player with that ID from the pool.
// Used for corners, where the named SetPieceTaker is delivering the ball and
// can't also be the one heading it home.
func pickAttacker(rand *rand.Rand, lineup GameLineup, ct ChanceType, excludeID string) SelectedPlayer {
	posWeights := defaultPositionPickWeights
	if profile, ok := chanceTypeProfiles[ct]; ok && profile.PositionWeights != nil {
		posWeights = profile.PositionWeights
	}

	// Sort players by ID for deterministic iteration.
	players := make([]SelectedPlayer, len(lineup.Players))
	copy(players, lineup.Players)
	sort.Slice(players, func(i, j int) bool { return players[i].ID < players[j].ID })

	weights := make([]float64, len(players))
	var total float64
	for i, p := range players {
		if excludeID != "" && p.ID == excludeID {
			continue
		}
		posW := float64(posWeights[p.SelectedPosition])
		if posW == 0 {
			continue
		}
		score := playerAttackForChance(p, ct)
		if score < 1 {
			score = 1
		}
		w := posW * score
		// TargetMan: bonus selection weight on aerial chance types.
		if p.Role == PlayerRoleTargetMan && (ct == ChanceTypeCorner || ct == ChanceTypeCross) {
			w *= 2.0
		}
		weights[i] = w
		total += w
	}
	if total == 0 {
		// Fallback: pick any player not on the exclude list.
		for _, p := range players {
			if excludeID == "" || p.ID != excludeID {
				return p
			}
		}
		return players[rand.Intn(len(players))]
	}
	pick := rand.Float64() * total
	var cum float64
	for i, w := range weights {
		cum += w
		if pick < cum {
			return players[i]
		}
	}
	return players[len(players)-1]
}

// chanceTypeAttackBoost returns the AttackBoost for a chance type, defaulting
// to 1.0 if the type isn't in the profile table.
func chanceTypeAttackBoost(ct ChanceType) float64 {
	if p, ok := chanceTypeProfiles[ct]; ok {
		return p.AttackBoost
	}
	return 1.0
}

// chanceTypeDefenseScale returns the DefenseScale for a chance type.
func chanceTypeDefenseScale(ct ChanceType) float64 {
	if p, ok := chanceTypeProfiles[ct]; ok {
		return p.DefenseScale
	}
	return 1.0
}
