package cards

import "github.com/nutthawit-l/dominion-grpc/internal/engine"

var Estate = &engine.Card{
	ID:            "estate",
	Name:          "Estate",
	Cost:          2,
	Types:         []engine.CardType{engine.TypeVictory},
	VictoryPoints: func(engine.PlayerState) int { return 1 },
}

var Duchy = &engine.Card{
	ID:            "duchy",
	Name:          "Duchy",
	Cost:          5,
	Types:         []engine.CardType{engine.TypeVictory},
	VictoryPoints: func(engine.PlayerState) int { return 3 },
}

var Province = &engine.Card{
	ID:            "province",
	Name:          "Province",
	Cost:          8,
	Types:         []engine.CardType{engine.TypeVictory},
	VictoryPoints: func(engine.PlayerState) int { return 6 },
}

var Curse = &engine.Card{
	ID:            "curse",
	Name:          "Curse",
	Cost:          0,
	Types:         []engine.CardType{engine.TypeCurse},
	VictoryPoints: func(engine.PlayerState) int { return -1 },
}

func init() {
	DefaultRegistry.Register(Estate)
	DefaultRegistry.Register(Duchy)
	DefaultRegistry.Register(Province)
	DefaultRegistry.Register(Curse)
}
