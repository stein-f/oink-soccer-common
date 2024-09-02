package soccer

type FormationConfig struct {
	FormationType   FormationType
	Slots           map[uint64]PlayerPosition
	DefenseModifier float64
	ControlModifier float64
	AttackModifier  float64
}

// ThePyramidFormation (2-1-1) is a defensive formation
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
	DefenseModifier: 1.1,
	ControlModifier: 0.975,
	AttackModifier:  0.9,
}

// TheDiamondFormation (2-1-1) is a defensive formation
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
	DefenseModifier: 0.95,
	ControlModifier: 1.05,
	AttackModifier:  0.95,
}

// TheYFormation (1-1-2) is an attacking formation
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
	DefenseModifier: 0.95,
	ControlModifier: 1.02,
	AttackModifier:  1.1,
}

// TheBoxFormation (2-0-2) a balanced formation for direct play and counter-attacking
//
//	    4 5
//
//		2 3
//		 1
var TheBoxFormation = FormationConfig{
	FormationType: FormationTypeBox,
	Slots: map[uint64]PlayerPosition{
		1: PlayerPositionGoalkeeper,
		2: PlayerPositionDefense,
		3: PlayerPositionDefense,
		4: PlayerPositionAttack,
		5: PlayerPositionAttack,
	},
	DefenseModifier: 1.1,
	ControlModifier: 0.95,
	AttackModifier:  1.1,
}
