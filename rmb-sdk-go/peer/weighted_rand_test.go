package peer

import (
	"fmt"
	"log"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func Example() {
	items := []WeightItem[string]{
		{
			Item:   "Banana",
			Weight: 0,
		},
		{
			Item:   "Apple",
			Weight: 0,
		},
		{
			Item:   "Mellon",
			Weight: 0,
		},
		{
			Item:   "Pineapple",
			Weight: 9,
		},
	}

	randomizer, err := NewWeightSlice(items)
	if err != nil {
		log.Fatal(err)
	}

	fruit := randomizer.Choose()
	fmt.Print(fruit)
	// Output: Pineapple
}

func TestWeightedRandom(t *testing.T) {
	testIterations := 1000000

	items := generateRandomItems(t, 10)

	weightSlice, err := NewWeightSlice(items)
	require.NoError(t, err)

	counts := make(map[int]int)
	for i := 0; i < testIterations; i++ {
		counts[weightSlice.Choose()]++
	}

	require.Equal(t, counts[0], 0) // first weight should have count = 0 because the weight is 0
	// verify counts
	for i, item := range items[0 : len(items)-1] { // ignore last item because we always calculate nextItem item
		nextItem := items[i+1]
		current, next := item.Item, nextItem.Item
		require.Less(t, counts[int(current)], counts[int(next)])
	}
}

func generateRandomItems(t *testing.T, n int) []WeightItem[int] {
	t.Helper()
	items := make([]WeightItem[int], 0, n)

	list := rand.Perm(n)
	for _, v := range list {
		item := WeightItem[int]{v, uint64(v)}
		items = append(items, item)
	}

	return items
}
