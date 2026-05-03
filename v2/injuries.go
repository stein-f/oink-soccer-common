package soccer

import (
	"math/rand"
	"time"

	"github.com/stein-f/oink-soccer-common/v2/internal/tuning"
)

type Injury struct {
	Severity       InjurySeverity `json:"severity"`
	MinDays        int            `json:"min_days"`
	MaxDays        int            `json:"max_days"`
	Name           string         `json:"name"`
	StatsReduction float64        `json:"stats_reduction"`
	Description    string         `json:"description"`
	Weight         uint           `json:"weight"`
}

// InjuryEvent describes an injury picked up during a match.
//
// In v1 the engine populated Expires using time.Now() at simulation time,
// which made the function non-deterministic in (seed, lineup). v2 leaves
// Expires as a zero value during simulation; downstream code should call
// ResolveInjuryExpiry with its own clock.
//
// DurationDays carries the rolled duration so callers can pin Expires
// against any clock without re-rolling and breaking determinism.
type InjuryEvent struct {
	TeamID       string    `json:"team_id"`
	PlayerID     string    `json:"player_id"`
	Expires      time.Time `json:"expires"`
	DurationDays int       `json:"duration_days,omitempty"`
	Injury       Injury    `json:"injury"`
}

type Injuries struct {
	HomeTeamInjuries []InjuryEvent `json:"home_team_injuries"`
	AwayTeamInjuries []InjuryEvent `json:"away_team_injuries"`
}

// ResolveInjuryExpiry computes an absolute expiry timestamp for an injury
// event using the supplied clock. v1 baked time.Now() into the simulation;
// v2 splits that out so the engine itself stays deterministic.
func ResolveInjuryExpiry(now time.Time, e InjuryEvent) time.Time {
	expires := now.AddDate(0, 0, e.DurationDays)
	return time.Date(expires.Year(), expires.Month(), expires.Day(), 23, 59, 59, 0, expires.Location())
}

// GetAllInjuries returns the catalogue of injuries the engine may pick from.
// Ported verbatim from v1 — these are flavor text + duration ranges, not
// engine balance. Adjustments to severity weights live in tuning.
func GetAllInjuries() []Injury {
	return injuryCatalogue
}

// rollInjuries decides which players in a lineup picked up an injury this
// match. opponentAggression is the average aggression rating of the other
// team. opponentFormationInjuryRisk is the multiplier from the opponent's
// formation profile (more aggressive shapes ⇒ more injuries inflicted).
func rollInjuries(rand *rand.Rand, lineup GameLineup, opponentAggression int, opponentFormationInjuryRisk float64, teamID string) []InjuryEvent {
	var out []InjuryEvent
	for _, p := range lineup.Players {
		got, injury := rollPlayerInjury(rand, p.Attributes.IsInjuryProne(), opponentAggression, opponentFormationInjuryRisk)
		if !got {
			continue
		}
		days := injury.MinDays
		if injury.MaxDays > injury.MinDays {
			days = rand.Intn(injury.MaxDays-injury.MinDays+1) + injury.MinDays
		}
		out = append(out, InjuryEvent{
			TeamID:       teamID,
			PlayerID:     p.ID,
			DurationDays: days,
			Injury:       injury,
		})
	}
	return out
}

// rollPlayerInjury performs the per-player injury roll. Weights:
//   - base: {false: NoInjuryWeightDefault, true: 1}  (≈ 30:1 ⇒ 1/31 odds)
//   - injury-prone: {false: NoInjuryWeightInjuryProne, true: 1} (≈ 15:1)
//
// Opponent aggression scales the "no injury" weight downward (capped at
// AggressionMaxNoInjuryReduction). Opponent formation injury risk
// multiplies the same downward pressure.
func rollPlayerInjury(r *rand.Rand, prone bool, aggression int, formationRisk float64) (bool, Injury) {
	noInjuryW := float64(tuning.NoInjuryWeightDefault)
	if prone {
		noInjuryW = float64(tuning.NoInjuryWeightInjuryProne)
	}

	// Aggression 0..100 ⇒ scale 1.0 .. (1 - max). Capped at the floor.
	if aggression > 0 {
		if aggression > 100 {
			aggression = 100
		}
		reduction := tuning.AggressionMaxNoInjuryReduction * float64(aggression) / 100.0
		noInjuryW *= 1.0 - reduction
	}

	// Formation injury risk > 1 ⇒ further reduce no-injury weight.
	if formationRisk > 0 {
		noInjuryW /= formationRisk
	}
	if noInjuryW < 1 {
		noInjuryW = 1
	}

	totalW := noInjuryW + 1.0
	if r.Float64()*totalW < noInjuryW {
		return false, Injury{}
	}
	return true, pickInjury(r)
}

func pickInjury(r *rand.Rand) Injury {
	var totalW uint
	for _, i := range injuryCatalogue {
		totalW += i.Weight
	}
	if totalW == 0 {
		return Injury{}
	}
	pick := uint(r.Intn(int(totalW)))
	var cum uint
	for _, i := range injuryCatalogue {
		cum += i.Weight
		if pick < cum {
			return i
		}
	}
	return injuryCatalogue[len(injuryCatalogue)-1]
}

// teamAverageAggression returns the mean aggression rating across a lineup.
func teamAverageAggression(lineup GameLineup) int {
	if len(lineup.Players) == 0 {
		return 0
	}
	total := 0
	for _, p := range lineup.Players {
		total += p.Attributes.AggressionRating
	}
	return total / len(lineup.Players)
}

// injuryCatalogue is ported verbatim from v1. Severity multipliers come
// from tuning so any future balance change to "what does a high-severity
// injury cost" lives in one place.
var injuryCatalogue = []Injury{
	// low severity (1-day, -5%)
	low("Minor sprain", "Overstretched a ligament performing an unsuccessful tackle.", 100),
	low("Squirrel Scare", "Spooked by a squirrel running onto the field, leading to a comical but unfortunate tumble.", 100),
	low("Pie Burn", "Out for a game after trying to eat pie too quickly during half-time match and burning the roof of their mouth.", 100),
	low("Laugh Attack", "Couldn't stop laughing after a teammate's joke and ended up with a side stitch.", 100),
	low("Dance-Off Defeat", "Suffered a minor ego bruise and twisted ankle during an impromptu pre-match dance-off.", 100),
	low("Turf Toe", "Stubbed a toe on the turf while celebrating a goal.", 100),
	low("Pepper Spray Incident", "Accidentally rubbed eyes after handling spicy food after the match.", 100),
	low("Selfie Slip", "Lost balance while taking a selfie on the field after the match, resulting in a harmless but embarrassing fall.", 100),
	low("Paparazzi Panic", "Momentarily blinded by a camera flash from an overzealous fan after the match.", 100),
	low("Locker Room Slippery Floor", "Slipped on a wet spot in the locker room, causing a minor sprain.", 100),
	low("Overzealous Autograph Signing", "Strained wrist after signing too many autographs post-match.", 100),

	// medium severity (2-3 day, -10%)
	med("Powerful sneeze", "'Nasty' back injury caused by a powerful sneeze.", 25),
	med("Hamstring strain", "Minor hamstring tear after sprinting to catch up with a breakaway.", 25),
	med("Concussion", "Head injury after a collision with a teammate during a header.", 25),
	med("Mismatched Boots", "Wore two left boots to the game, resulting in blisters and confused running.", 25),
	med("Charley Horse", "Severe muscle cramp from overexertion during the match.", 25),
	med("Overenthusiastic Headbutt", "Minor concussion after an overzealous attempt to head the ball.", 25),
	med("Mascot Mishap", "Collided with the team mascot during a halftime stunt, resulting in a bruised rib.", 25),
	med("Helmet Hair Disaster", "Spent too much time adjusting hair under the helmet, leading to neck strain.", 25),
	med("Post-Match Pizza Overload", "Ate too much pizza after the match, causing severe stomach cramps.", 25),

	// high severity (3-5 day, -15%)
	high("Achilles Tendon Rupture", "Achilles tendon rupture after a sudden acceleration to chase down a ball.", 10),
	high("High-five fail", "Missed a high-five and accidentally poked themselves in the eye.", 5),
	high("ACL Tear", "Severe knee injury after an awkward landing.", 10),
	high("Ballistic Banana Slip", "Slipped on a stray banana peel on the field, causing a back injury.", 5),
	high("Celebration Injury", "Pulled a muscle during an over-enthusiastic goal celebration.", 5),
	high("Caught on the Corner Flag", "Twisted an ankle after getting tangled with the corner flag during a quick turn.", 5),
	high("Post-Match Cramp", "Severe muscle cramp from dehydration after the match, requiring extended recovery.", 5),
	high("Hydration Hazard", "Slipped on spilled water in the locker room after the match, resulting in a dislocated shoulder.", 5),
}

func low(name, desc string, w uint) Injury {
	return Injury{Severity: InjurySeverityLow, StatsReduction: tuning.StatsReductionLow, MinDays: 1, MaxDays: 1, Name: name, Description: desc, Weight: w}
}

func med(name, desc string, w uint) Injury {
	return Injury{Severity: InjurySeverityMid, StatsReduction: tuning.StatsReductionMed, MinDays: 2, MaxDays: 3, Name: name, Description: desc, Weight: w}
}

func high(name, desc string, w uint) Injury {
	return Injury{Severity: InjurySeverityHigh, StatsReduction: tuning.StatsReductionHigh, MinDays: 3, MaxDays: 5, Name: name, Description: desc, Weight: w}
}
