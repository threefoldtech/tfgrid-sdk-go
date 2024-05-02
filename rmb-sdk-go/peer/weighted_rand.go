package peer

import (
	"fmt"
	"math/rand"
	"sort"

	"gonum.org/v1/gonum/floats"
)

// WeightItem is a generic wrapper that can be used to add weights for any item.
type WeightItem[T any] struct {
	Item   T
	Weight uint64
}

// A WeightSlice caches slice options for weighted random selection.
type WeightSlice[T any] struct {
	data   []WeightItem[T]
	totals []float64
}

// NewWeightSlice initializes a new weight slice for picking from the provided choices with their weights.
func NewWeightSlice[T any](items []WeightItem[T]) (*WeightSlice[T], error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("empty items to be chosen")
	}

	sort.Slice(items, func(i, j int) bool { // sort items according to their weights
		return items[i].Weight < items[j].Weight
	})

	var weights []float64
	for i := range items {
		weights = append(weights, float64(items[i].Weight))
	}

	// get cumulative summations for items' weights
	totals := make([]float64, len(weights))
	floats.CumSum(totals, weights)

	return &WeightSlice[T]{data: items, totals: totals}, nil
}

// Choose returns a single weighted random item from the slice.
func (c WeightSlice[T]) Choose() T {
	r := rand.Intn(int(c.totals[len(c.totals)-1]))
	i := sort.Search(len(c.totals), func(i int) bool { return c.totals[i] > float64(r) })
	return c.data[i].Item
}
