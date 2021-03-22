package lib

type Score struct {
	game           Game
	variant        Variant
	scenarioData   *Data
	commanderFlags *CommanderFlags
	options        *Options

	MenLost                   [2]int // 29927 + side*2
	TanksLost                 [2]int // 29927 + 4 + side*2
	CitiesHeld                [2]int // 29927 + 13 + side*2
	CriticalLocationsCaptured [2]int // 29927 + 21 + side*2
}

func newScore(game Game, variant Variant, scenarioData *Data, commanderFlags *CommanderFlags, options *Options) *Score {
	return &Score{
		game:           game,
		variant:        variant,
		scenarioData:   scenarioData,
		commanderFlags: commanderFlags,
		options:        options,
		CitiesHeld:     variant.CitiesHeld}
}

func (s Score) WinningSideAndAdvantage() (winningSide int, advantage int) {
	side0Score := (1 + s.MenLost[1] + s.TanksLost[1]) * s.variant.Data3 / 8
	side1Score := 1 + s.MenLost[0] + s.TanksLost[0]
	if s.game != Conflict {
		side0Score += s.CitiesHeld[0] * 3
		side1Score += s.CitiesHeld[1] * 3
	} else {
		side0Score += s.CitiesHeld[0] * 6 / (s.scenarioData.Data174 + 1)
		side1Score += s.CitiesHeld[1] * 6 / (s.scenarioData.Data174 + 1)
	}
	var score int
	if side0Score < side1Score {
		score = side1Score * 3 / side0Score
		winningSide = 1
	} else {
		score = side0Score * 3 / side1Score
		winningSide = 0
	}
	advantage = 4
	if score >= 3 {
		advantage = Clamp(score-3, 0, 4)
	}
	return
}

func (s Score) FinalResults(playerSide int) (int, int, int) {
	winningSide, advantage := s.WinningSideAndAdvantage()
	var absoluteAdvantage int // a number from [1..10]
	if winningSide == 0 {
		absoluteAdvantage = advantage + 6
	} else {
		absoluteAdvantage = 5 - advantage
	}
	v73 := playerSide
	if s.commanderFlags.PlayerControlled[0] && s.commanderFlags.PlayerControlled[1] {
		if advantage < 6 {
			v73 = 1
		} else {
			v73 = 0
		}
	}
	var v74 int
	if v73 == 0 {
		v74 = absoluteAdvantage
	} else {
		v74 = 11 - absoluteAdvantage
	}

	criticalLocationBalance := s.CriticalLocationsCaptured[0] - s.CriticalLocationsCaptured[1]
	if criticalLocationBalance >= s.variant.CriticalLocations[0] {
		v74 = 1 + 9*(1-v73)
	}
	if -criticalLocationBalance >= s.variant.CriticalLocations[1] {
		v74 = 1 + 9*v73
	}
	var difficulty int
	if v73 == 0 {
		difficulty = s.options.GameBalance
	} else {
		difficulty = 4 - s.options.GameBalance
	}
	rank := Min(v74-2*difficulty+4, 12)
	return v74 - 1, difficulty, rank - 1
}
