package usermgmt

import (
	"fmt"
	"testing"

	"github.com/larsartmann/go-cqrs-lite/dedup/v4"
	"github.com/larsartmann/go-cqrs-lite/id/v4"
)

// BenchmarkDedupRing_vs_Map compares the bounded Ring buffer (O(1) memory)
// against an unbounded map for typical journal replay dedup scenarios.
// The Ring should be faster at larger sizes due to cache locality and
// zero GC pressure, while the map should be faster at small sizes.

func makeEventIDs(n int) []id.EventID {
	ids := make([]id.EventID, n)
	for i := range ids {
		ids[i] = id.NewEventID()
	}
	return ids
}

func benchmarkRing(b *testing.B, ids []id.EventID) {
	b.Helper()
	b.ResetTimer()
	for range b.N {
		r := dedup.NewRing(dedup.DefaultCapacity)
		for _, eid := range ids {
			r.Add(eid.String())
		}
	}
}

func benchmarkMap(b *testing.B, ids []id.EventID) {
	b.Helper()
	b.ResetTimer()
	for range b.N {
		m := make(map[id.EventID]struct{}, len(ids))
		for _, eid := range ids {
			m[eid] = struct{}{}
		}
	}
}

func BenchmarkDedupRing_vs_Map(b *testing.B) {
	sizes := []int{100, 1000, 10000}
	for _, size := range sizes {
		ids := makeEventIDs(size)
		b.Run(fmt.Sprintf("Ring/%d", size), func(b *testing.B) {
			benchmarkRing(b, ids)
		})
		b.Run(fmt.Sprintf("Map/%d", size), func(b *testing.B) {
			benchmarkMap(b, ids)
		})
	}
}
