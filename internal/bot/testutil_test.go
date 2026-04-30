package bot

import (
	pb "github.com/nutthawit-l/dominion-grpc/gen/go/dominion/v1"
)

// snapshotWithSupply builds a minimal GameStateSnapshot with the given supply
// pile counts and a single player at index 0.
func snapshotWithSupply(counts map[string]int) *pb.GameStateSnapshot {
	piles := make([]*pb.SupplyPile, 0, len(counts))
	for id, n := range counts {
		piles = append(piles, &pb.SupplyPile{CardId: id, Count: int32(n)})
	}
	return &pb.GameStateSnapshot{
		Players: []*pb.PlayerView{{PlayerIdx: 0}},
		Supply:  piles,
	}
}
