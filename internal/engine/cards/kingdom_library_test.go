package cards

import (
	"testing"

	"github.com/nutthawit-l/dominion-grpc/internal/engine"
	"github.com/stretchr/testify/require"
)

func TestLibrary_Metadata(t *testing.T) {
	require.Equal(t, engine.CardID("library"), Library.ID)
	require.Equal(t, "Library", Library.Name)
	require.Equal(t, 5, Library.Cost)
	require.True(t, Library.HasType(engine.TypeAction))
}

func TestLibrary_OnPlay_HandAlreadySeven_NoDraw_NoDecision(t *testing.T) {
	gs := newTestStateForCardsWithRNG(2, 1)
	gs.Players[0].Hand = []engine.CardID{
		"copper", "copper", "copper", "copper", "copper", "copper", "copper",
	}
	gs.Players[0].Deck = []engine.CardID{"silver", "silver"}
	gs.Players[0].InPlay = []engine.CardID{"library"}

	Library.OnPlay(gs, 0)
	require.Nil(t, gs.PendingDecision, "hand already 7 → no decision")
	require.Len(t, gs.Players[0].Hand, 7, "no draw")
	require.Len(t, gs.Players[0].Deck, 2, "deck untouched")
	require.Empty(t, gs.Players[0].SetAside)
}

func TestLibrary_OnPlay_DrawsNonAction_Continues(t *testing.T) {
	gs := newTestStateForCardsWithRNG(2, 1)
	gs.Players[0].Hand = []engine.CardID{"copper", "copper", "copper", "copper", "copper"}
	gs.Players[0].Deck = []engine.CardID{"silver", "silver"}
	gs.Players[0].InPlay = []engine.CardID{"library"}

	Library.OnPlay(gs, 0)
	require.Nil(t, gs.PendingDecision, "no Action drawn → no prompt")
	require.Len(t, gs.Players[0].Hand, 7, "drew up to 7")
}

func TestLibrary_OnPlay_DrawsAction_Prompts(t *testing.T) {
	gs := newTestStateForCardsWithRNG(2, 1)
	gs.Players[0].Hand = []engine.CardID{"copper", "copper", "copper"}
	// Top of deck (slice end) is smithy — Library will draw it next.
	gs.Players[0].Deck = []engine.CardID{"copper", "smithy"}
	gs.Players[0].InPlay = []engine.CardID{"library"}

	Library.OnPlay(gs, 0)
	require.NotNil(t, gs.PendingDecision)
	p, ok := gs.PendingDecision.Prompt.(engine.SetAsideActionPrompt)
	require.True(t, ok)
	require.Equal(t, engine.CardID("smithy"), p.Card)
	require.Equal(t, engine.CardID("smithy"),
		gs.PendingDecision.Context[engine.CtxKeyCard].(engine.CardID))
}

func TestLibrary_Resolve_Yes_SetsAsideAndLoops(t *testing.T) {
	gs := newTestStateForCardsWithRNG(2, 1)
	gs.Players[0].Hand = []engine.CardID{"copper", "copper", "copper"}
	// 4 silvers + smithy: smithy is set aside (Yes), then 4 silvers drawn → hand=7
	gs.Players[0].Deck = []engine.CardID{"silver", "silver", "silver", "silver", "smithy"}
	gs.Players[0].InPlay = []engine.CardID{"library"}

	Library.OnPlay(gs, 0)
	require.NotNil(t, gs.PendingDecision)
	require.Equal(t, engine.CardID("smithy"),
		gs.PendingDecision.Context[engine.CtxKeyCard].(engine.CardID))

	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.YesNoAnswer{Yes: true},
	}, registryLookup)
	require.NoError(t, err)

	require.Len(t, gs.Players[0].Hand, 7)
	require.Empty(t, gs.Players[0].SetAside)
	require.Contains(t, gs.Players[0].Discard, engine.CardID("smithy"))
	require.Nil(t, gs.PendingDecision)
}

func TestLibrary_Resolve_No_KeepsAndLoops(t *testing.T) {
	gs := newTestStateForCardsWithRNG(2, 1)
	gs.Players[0].Hand = []engine.CardID{"copper", "copper", "copper", "copper", "copper", "copper"}
	gs.Players[0].Deck = []engine.CardID{"smithy"}
	gs.Players[0].InPlay = []engine.CardID{"library"}

	Library.OnPlay(gs, 0)
	require.NotNil(t, gs.PendingDecision)

	_, _, err := engine.Apply(gs, engine.ResolveDecision{
		PlayerIdx: 0, DecisionID: gs.PendingDecision.ID,
		Answer: engine.YesNoAnswer{Yes: false},
	}, registryLookup)
	require.NoError(t, err)

	require.Len(t, gs.Players[0].Hand, 7)
	require.Contains(t, gs.Players[0].Hand, engine.CardID("smithy"),
		"Smithy stayed in hand on No")
	require.Empty(t, gs.Players[0].SetAside)
	require.Nil(t, gs.PendingDecision)
}

func TestLibrary_StopsWhenBothDeckAndDiscardEmpty(t *testing.T) {
	gs := newTestStateForCardsWithRNG(2, 1)
	gs.Players[0].Hand = []engine.CardID{"copper", "copper"}
	gs.Players[0].Deck = []engine.CardID{"silver", "silver"}
	gs.Players[0].Discard = nil
	gs.Players[0].InPlay = []engine.CardID{"library"}

	Library.OnPlay(gs, 0)
	require.Nil(t, gs.PendingDecision, "no Action ever drawn")
	require.Len(t, gs.Players[0].Hand, 4, "drew everything left, stopped at 4")
	require.Empty(t, gs.Players[0].Deck)
	require.Empty(t, gs.Players[0].Discard)
}
