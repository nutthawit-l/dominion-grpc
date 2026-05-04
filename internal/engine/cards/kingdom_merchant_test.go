package cards

import (
	"testing"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/stretchr/testify/require"
)

func TestMerchant_Metadata(t *testing.T) {
	require.Equal(t, engine.CardID("merchant"), Merchant.ID)
	require.Equal(t, "Merchant", Merchant.Name)
	require.Equal(t, 3, Merchant.Cost)
	require.True(t, Merchant.HasType(engine.TypeAction))
	_, ok := DefaultRegistry.Lookup("merchant")
	require.True(t, ok, "Merchant must be registered in DefaultRegistry")
}

func TestMerchant_OnPlay_DrawsAndAddsAction(t *testing.T) {
	gs := newTestStateForCardsWithRNG(2, 1)
	gs.Players[0].Deck = []engine.CardID{"copper", "estate", "silver"}

	Merchant.OnPlay(gs, 0)

	require.Len(t, gs.Players[0].Hand, 1, "Merchant draws 1 card")
	require.Equal(t, 1, gs.Players[0].Actions, "Merchant adds 1 Action")
}

func TestMerchant_OnPlay_IncrementsBonusCharges(t *testing.T) {
	gs := newTestStateForCardsWithRNG(2, 1)
	gs.Players[0].Deck = []engine.CardID{"copper", "copper", "copper"}

	Merchant.OnPlay(gs, 0)
	require.Equal(t, 1, gs.Players[0].MerchantBonusCharges,
		"first Merchant play → 1 charge")

	Merchant.OnPlay(gs, 0)
	require.Equal(t, 2, gs.Players[0].MerchantBonusCharges,
		"second Merchant play → 2 charges")
}

func TestMerchant_PlayedThenSilver_AddsOneCoinBonus(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"merchant", "silver"}
	gs.Players[0].Deck = []engine.CardID{"copper", "copper", "copper"}

	// Play Merchant in Action phase.
	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "merchant"}, testLookup)
	require.NoError(t, err)

	// End Action phase; enter Buy.
	_, _, err = engine.Apply(gs, engine.EndPhase{PlayerIdx: 0}, testLookup)
	require.NoError(t, err)
	require.Equal(t, engine.PhaseBuy, gs.Phase)

	// Play Silver — should pay 2 (Silver) + 1 (Merchant bonus).
	_, _, err = engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "silver"}, testLookup)
	require.NoError(t, err)

	require.Equal(t, 3, gs.Players[0].Coins,
		"Silver (+2) + Merchant bonus (+1) = 3 coins")
	require.True(t, gs.Players[0].FirstSilverPlayedThisTurn,
		"FirstSilverPlayedThisTurn must be true after first Silver play")
}

func TestMerchant_TwoMerchants_ThenSilver_AddsTwoCoinBonus(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Actions = 2
	gs.Players[0].Hand = []engine.CardID{"merchant", "merchant", "silver"}
	gs.Players[0].Deck = []engine.CardID{"copper", "copper", "copper"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "merchant"}, testLookup)
	require.NoError(t, err)
	_, _, err = engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "merchant"}, testLookup)
	require.NoError(t, err)
	_, _, err = engine.Apply(gs, engine.EndPhase{PlayerIdx: 0}, testLookup)
	require.NoError(t, err)
	_, _, err = engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "silver"}, testLookup)
	require.NoError(t, err)

	require.Equal(t, 4, gs.Players[0].Coins,
		"Silver (+2) + 2× Merchant bonus (+2) = 4 coins")
}

func TestMerchant_OnlyFirstSilverTriggersBonus(t *testing.T) {
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"merchant", "silver", "silver"}
	gs.Players[0].Deck = []engine.CardID{"copper", "copper", "copper"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "merchant"}, testLookup)
	require.NoError(t, err)
	_, _, err = engine.Apply(gs, engine.EndPhase{PlayerIdx: 0}, testLookup)
	require.NoError(t, err)
	_, _, err = engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "silver"}, testLookup)
	require.NoError(t, err)
	require.Equal(t, 3, gs.Players[0].Coins, "first Silver: 2+1=3")

	_, _, err = engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "silver"}, testLookup)
	require.NoError(t, err)
	require.Equal(t, 5, gs.Players[0].Coins, "second Silver: +2 only (no bonus)")
}

func TestMerchant_SilverThenMerchant_NoBonusThisTurn(t *testing.T) {
	gs := newActionPhaseState()
	// Switch directly to Buy phase to play Silver first; Merchant lives
	// in Hand but cannot be played in Buy phase. We skip Merchant
	// entirely this turn and assert no bonus.
	gs.Phase = engine.PhaseBuy
	gs.Players[0].Hand = []engine.CardID{"silver", "silver"}

	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "silver"}, testLookup)
	require.NoError(t, err)
	require.Equal(t, 2, gs.Players[0].Coins, "Silver alone: +2 (no Merchant)")
	require.True(t, gs.Players[0].FirstSilverPlayedThisTurn)

	// Now hand-poke MerchantBonusCharges to simulate "Merchant played
	// after first Silver in same turn." Per Dominion rules, this should
	// NOT retroactively award a bonus on the next Silver — the flag is
	// already set.
	gs.Players[0].MerchantBonusCharges = 1

	_, _, err = engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "silver"}, testLookup)
	require.NoError(t, err)
	require.Equal(t, 4, gs.Players[0].Coins,
		"second Silver: +2, NO bonus because flag already set")
}

func TestMerchant_ThroneRoom_DoublesBonusOnFirstSilver(t *testing.T) {
	// Regression coverage for §3.7 of the Tier 5 spec — TR(Merchant) +
	// Silver gives +$2 bonus from doubled charges. Replaces the JSON
	// replay fixture the spec originally proposed.
	gs := newActionPhaseState()
	gs.Players[0].Hand = []engine.CardID{"throne_room", "merchant", "silver"}
	gs.Players[0].Deck = []engine.CardID{"copper", "copper", "copper", "copper"}

	// Play Throne Room.
	_, _, err := engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "throne_room"}, testLookup)
	require.NoError(t, err)
	require.NotNil(t, gs.PendingDecision, "TR prompts ChooseActionFromHand")

	// Resolve TR's choose-action prompt with Merchant.
	_, _, err = engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx:  0,
		DecisionID: gs.PendingDecision.ID,
		Answer:     engine.CardChoiceAnswer{Card: "merchant"},
	}, testLookup)
	require.NoError(t, err)

	require.Equal(t, 2, gs.Players[0].MerchantBonusCharges,
		"TR(Merchant) accumulates 2 charges")

	// End Action phase; enter Buy.
	_, _, err = engine.Apply(gs, engine.EndPhase{PlayerIdx: 0}, testLookup)
	require.NoError(t, err)

	// Play Silver — should pay 2 (Silver) + 2 (TR-doubled Merchant bonus).
	_, _, err = engine.Apply(gs, engine.PlayCard{PlayerIdx: 0, Card: "silver"}, testLookup)
	require.NoError(t, err)

	require.Equal(t, 4, gs.Players[0].Coins,
		"TR(Merchant) + Silver: 2 + 2 = 4 coins")
	require.True(t, gs.Players[0].FirstSilverPlayedThisTurn)
}
