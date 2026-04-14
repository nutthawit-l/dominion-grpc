package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// replayFixture is the on-disk format of a replay: a seed plus a
// sequence of actions, plus the expected final outcome.
type replayFixture struct {
	Seed            int64        `json:"seed"`
	PlayerNames     []string     `json:"player_names"`
	Actions         []replayStep `json:"actions"`
	ExpectedEnded   bool         `json:"expected_ended"`
	ExpectedWinners []int        `json:"expected_winners"`
}

type replayStep struct {
	Kind   string `json:"kind"` // "play" | "buy" | "end_phase"
	Player int    `json:"player"`
	Card   string `json:"card,omitempty"`
}

func (r replayStep) toAction() Action {
	switch r.Kind {
	case "play":
		return PlayCard{PlayerIdx: r.Player, Card: CardID(r.Card)}
	case "buy":
		return BuyCard{PlayerIdx: r.Player, Card: CardID(r.Card)}
	case "end_phase":
		return EndPhase{PlayerIdx: r.Player}
	}
	return nil
}

// TestReplay_RegressionFixtures runs every file under testdata/replays/.
// Empty directories are valid — the walker just finds nothing to run.
func TestReplay_RegressionFixtures(t *testing.T) {
	entries, err := filepath.Glob(filepath.Join("..", "..", "testdata", "replays", "*.json"))
	require.NoError(t, err)
	for _, path := range entries {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			raw, err := os.ReadFile(path)
			require.NoError(t, err)
			var fx replayFixture
			require.NoError(t, json.Unmarshal(raw, &fx))

			s, err := NewGame("replay", fx.PlayerNames, fx.Seed, basicsLookup2)
			require.NoError(t, err)
			for i, step := range fx.Actions {
				act := step.toAction()
				require.NotNilf(t, act, "step %d has unknown kind %q", i, step.Kind)
				_, _, err := Apply(s, act, basicsLookup2)
				require.NoErrorf(t, err, "step %d (%+v)", i, step)
			}
			require.Equal(t, fx.ExpectedEnded, s.Ended)
			require.Equal(t, fx.ExpectedWinners, s.Winners)
		})
	}
}
