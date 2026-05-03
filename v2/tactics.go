package soccer

// Tactics is the optional bundle of manager-controlled levers a team can
// set before kick-off. The zero value means "neutral" — every existing v1
// lineup that doesn't populate Tactics still produces a sensible v2 game.
//
// Each lever trades one axis for another (rock-paper-scissors style); none
// is meant to be a strict upgrade. See doc.go for the full list.
type Tactics struct {
	Press         PressLevel
	Tempo         TempoLevel
	LineHeight    LineHeight
	SetPieceTaker string // PlayerID — receives free kicks / corners / penalties
}

// PressLevel controls how much defensive pressure a team applies. Higher
// press shrinks the opponent's effective control (fewer clean possessions)
// but raises this team's own injury risk (more tackles ⇒ more knocks).
type PressLevel string

const (
	PressLevelNone   PressLevel = ""
	PressLevelLow    PressLevel = "low"
	PressLevelMedium PressLevel = "medium"
	PressLevelHigh   PressLevel = "high"
)

// TempoLevel controls how quickly the team plays. Faster tempo creates
// more chances but each chance is slightly lower quality (rushed shots).
type TempoLevel string

const (
	TempoLevelNone   TempoLevel = ""
	TempoLevelSlow   TempoLevel = "slow"
	TempoLevelNormal TempoLevel = "normal"
	TempoLevelFast   TempoLevel = "fast"
)

// LineHeight controls how high the defensive line plays. A high line
// compresses the pitch (more pressure on the opponent) but is vulnerable
// to balls in behind (slight boost to opponent breakaway-style chances).
type LineHeight string

const (
	LineHeightNone   LineHeight = ""
	LineHeightDeep   LineHeight = "deep"
	LineHeightNormal LineHeight = "normal"
	LineHeightHigh   LineHeight = "high"
)

// PlayerRole is an optional tag a manager can apply to a single player to
// reshape their contribution. Multiple players can share the same role
// (e.g. two playmakers) — the engine just looks at the field per-player.
type PlayerRole string

const (
	PlayerRoleNone       PlayerRole = ""
	PlayerRoleCaptain    PlayerRole = "captain"     // small team-wide boost
	PlayerRoleTargetMan  PlayerRole = "target_man"  // bonus on corners + crosses
	PlayerRolePlaymaker  PlayerRole = "playmaker"   // boosts effective control
	PlayerRoleBallWinner PlayerRole = "ball_winner" // boosts effective defense + aggression
)

// pressControlFactor returns the multiplier applied to the *opposing*
// team's control when this team is pressing.
func pressControlFactor(p PressLevel) float64 {
	switch p {
	case PressLevelLow:
		return 1.02
	case PressLevelMedium:
		return 0.98
	case PressLevelHigh:
		return 0.94
	default:
		return 1.0
	}
}

// pressInjuryFactor returns the multiplier applied to *own* team injury
// risk. Higher press = more own injuries.
//
// Note: injuries are a *next-game* cost (and managers can use recovery items
// to mitigate them). The press tactic also imposes a within-match cost via
// pressFatigueFactor below — without it, high press would be a free lunch.
func pressInjuryFactor(p PressLevel) float64 {
	switch p {
	case PressLevelLow:
		return 0.95
	case PressLevelMedium:
		return 1.0
	case PressLevelHigh:
		return 1.10
	default:
		return 1.0
	}
}

// pressFatigueFactor returns the multiplier applied to a pressing team's
// *own attack quality* in the final third of the match. Pressing high is
// physically taxing — the team that's been chasing all game fades after
// the 60th minute and is genuinely gassed by the 75th. This is what makes
// the press tactic a real in-match trade-off rather than a free lunch
// (the injury bump applies to the next game, which managers can mitigate).
//
//   - Low / medium press → no fatigue.
//   - High press, 0-59 min → fresh, no penalty.
//   - High press, 60-74 min → -10% attack quality (legs going).
//   - High press, 75+ min → -18% attack quality (fully gassed).
//
// Sized so that a high-pressing team's gain from cutting opponent control
// (~5% fewer opp goals) is roughly washed out by the late-game scoring
// drop on their own end. Pressing is then a stylistic choice — you trade
// late-game attack for possession control and injury pressure on the
// opponent — instead of a strict upgrade.
func pressFatigueFactor(p PressLevel, minute int) float64 {
	if p != PressLevelHigh {
		return 1.0
	}
	switch {
	case minute < 60:
		return 1.0
	case minute < 75:
		return 0.90
	default:
		return 0.82
	}
}

// tempoChanceFactor returns the multiplier on this team's chance volume.
func tempoChanceFactor(t TempoLevel) float64 {
	switch t {
	case TempoLevelSlow:
		return 0.92
	case TempoLevelNormal:
		return 1.0
	case TempoLevelFast:
		return 1.10
	default:
		return 1.0
	}
}

// tempoQualityFactor returns the multiplier on this team's chance quality.
// Faster tempo lowers quality slightly (more rushed shots).
func tempoQualityFactor(t TempoLevel) float64 {
	switch t {
	case TempoLevelSlow:
		return 1.05
	case TempoLevelNormal:
		return 1.0
	case TempoLevelFast:
		return 0.96
	default:
		return 1.0
	}
}

// lineHeightControlFactor — a deep line cedes the midfield to the opponent
// (more space to dictate), while a high line compresses the pitch and
// suppresses opponent control. This is the standard football intuition.
//
// Combined with lineHeightDefenseFactor (deep ×1.05, high ×0.96) the choice
// becomes a real trade: deep = better defense / opponent dictates more,
// high = brittle defense / opponent suppressed.
func lineHeightControlFactor(l LineHeight) float64 {
	switch l {
	case LineHeightDeep:
		return 1.03
	case LineHeightNormal:
		return 1.0
	case LineHeightHigh:
		return 0.97
	default:
		return 1.0
	}
}

// lineHeightDefenseFactor — a high line is offside-trap brittle; a deep
// line is more solid.
func lineHeightDefenseFactor(l LineHeight) float64 {
	switch l {
	case LineHeightDeep:
		return 1.05
	case LineHeightNormal:
		return 1.0
	case LineHeightHigh:
		return 0.96
	default:
		return 1.0
	}
}
