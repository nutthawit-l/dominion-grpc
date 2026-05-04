package engine

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsGameOver_ProvinceEmpty(t *testing.T) {
	gs := newTestState(2)
	gs.Supply.Piles = map[CardID]int{"province": 0, "copper": 10}
	require.True(t, IsGameOver(gs))
}

func TestIsGameOver_ThreePilesEmpty(t *testing.T) {
	gs := newTestState(2)
	gs.Supply.Piles = map[CardID]int{
		"province": 5,
		"a":        0,
		"b":        0,
		"c":        0,
	}
	require.True(t, IsGameOver(gs))
}

func TestIsGameOver_NotYet(t *testing.T) {
	gs := newTestState(2)
	gs.Supply.Piles = map[CardID]int{
		"province": 5,
		"a":        0,
		"b":        0,
		"c":        1,
	}
	require.False(t, IsGameOver(gs))
}

func TestComputeScore_SumsVictoryCardsAcrossAllZones(t *testing.T) {
	lookup := func(id CardID) (*Card, bool) {
		switch id {
		case "estate":
			return &Card{VictoryPoints: func(PlayerState) int { return 1 }}, true
		case "province":
			return &Card{VictoryPoints: func(PlayerState) int { return 6 }}, true
		case "curse":
			return &Card{VictoryPoints: func(PlayerState) int { return -1 }}, true
		}
		return &Card{}, true
	}
	ps := PlayerState{
		Hand:    []CardID{"estate", "estate"},
		Deck:    []CardID{"province"},
		Discard: []CardID{"curse"},
		InPlay:  []CardID{"estate"},
	}
	require.Equal(t, 1+1+6-1+1, ComputeScore(ps, lookup))
}

func TestDetermineWinners_Ties(t *testing.T) {
	scores := []int{10, 10, 5}
	require.Equal(t, []PlayerIdx{0, 1}, DetermineWinners(scores))
}

func TestDetermineWinners_SoloWinner(t *testing.T) {
	scores := []int{5, 10, 5}
	require.Equal(t, []PlayerIdx{1}, DetermineWinners(scores))
}

func TestComputeScore_Gardens_FloorsByTen(t *testing.T) {
	lookup := func(id CardID) (*Card, bool) {
		switch id {
		case "gardens":
			return &Card{
				VictoryPoints: func(p PlayerState) int {
					n := len(p.Hand) + len(p.Deck) + len(p.Discard) + len(p.InPlay) + len(p.SetAside)
					return n / 10
				},
			}, true
		}
		return &Card{}, true
	}
	// 18 cards across all zones; the single Gardens contributes
	// floor(18/10) = 1.
	ps := PlayerState{
		Hand:     []CardID{"gardens", "copper", "copper", "copper"},
		Deck:     []CardID{"copper", "copper", "copper", "copper"},
		Discard:  []CardID{"copper", "copper", "copper", "copper"},
		InPlay:   []CardID{"copper", "copper", "copper"},
		SetAside: []CardID{"copper", "copper", "copper"},
	}
	require.Equal(t, 1, ComputeScore(ps, lookup))
}
