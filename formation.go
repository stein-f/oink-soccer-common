package soccer

type FormationConfig struct {
	FormationType   FormationType
	Slots           map[uint64]PlayerPosition
	DefenseModifier float64
	ControlModifier float64
	AttackModifier  float64
}

// TheYFormation (1-1-2) is an attacking formation
// 10% defense penalty
// 10% attack boost
// 4  5
//
//	3
//	2
//	1
var TheYFormation = FormationConfig{
	FormationType: FormationTypeY,
	Slots: map[uint64]PlayerPosition{
		1: PlayerPositionGoalkeeper,
		2: PlayerPositionDefense,
		3: PlayerPositionMidfield,
		4: PlayerPositionAttack,
		5: PlayerPositionAttack,
	},
	DefenseModifier: 0.8,
	ControlModifier: 1,
	AttackModifier:  1.2,
}

// ThePyramidFormation (2-1-1) is a defensive formation
// 20% defense boost
// 20% attack penalty
//
//	     5
//		 4
//		2 3
//		 1
var ThePyramidFormation = FormationConfig{
	FormationType: FormationTypePyramid,
	Slots: map[uint64]PlayerPosition{
		1: PlayerPositionGoalkeeper,
		2: PlayerPositionDefense,
		3: PlayerPositionDefense,
		4: PlayerPositionMidfield,
		5: PlayerPositionAttack,
	},
	DefenseModifier: 1.2,
	ControlModifier: 1,
	AttackModifier:  0.8,
}

// TheDiamondFormation (2-1-1) is a defensive formation
// 10% defense penalty
// 20% control boost
// 10% attack penalty
//
//	     5
//		3 4
//		 2
//		 1
var TheDiamondFormation = FormationConfig{
	FormationType: FormationTypeDiamond,
	Slots: map[uint64]PlayerPosition{
		1: PlayerPositionGoalkeeper,
		2: PlayerPositionDefense,
		3: PlayerPositionMidfield,
		4: PlayerPositionMidfield,
		5: PlayerPositionAttack,
	},
	DefenseModifier: 0.9,
	ControlModifier: 1.2,
	AttackModifier:  0.9,
}
