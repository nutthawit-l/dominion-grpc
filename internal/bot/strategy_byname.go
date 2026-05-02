package bot

import "fmt"

// StrategyByName returns a fresh Strategy by its short name. Used by
// cmd/bot/main.go and by integration tests that pick strategies from
// a parameterized list. Adding a new strategy means adding a case
// here AND extending the closeout sweep's strategies list in
// integration_full_set_test.go.
func StrategyByName(name string) (Strategy, error) {
	switch name {
	case "bigmoney":
		return BigMoney{}, nil
	case "smithy_bm":
		return SmithyBM{}, nil
	case "chapel_bm":
		return NewChapelBM(), nil
	case "remodel_bm":
		return NewRemodelBM(), nil
	case "witch_bm":
		return NewWitchBM(), nil
	case "militia_bm":
		return NewMilitiaBM(), nil
	case "throneroom_bm":
		return NewThroneRoomBM(), nil
	case "library_bm":
		return NewLibraryBM(), nil
	case "sentry_bm":
		return NewSentryBM(), nil
	case "merchant_bm":
		return NewMerchantBM(), nil
	case "gardens_bm":
		return NewGardensBM(), nil
	}
	return nil, fmt.Errorf("unknown strategy %q", name)
}
