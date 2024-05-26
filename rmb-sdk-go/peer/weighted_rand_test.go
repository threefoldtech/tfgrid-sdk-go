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

	i, fruit := randomizer.Choose()
	fmt.Printf("%v: %s", i, fruit)
	// Output: 3: Pineapple
}

func TestWeightedRandom(t *testing.T) {
	testIterations := 1000000

	items := generateRandomItems(t, 10)

	weightSlice, err := NewWeightSlice(items)
	require.NoError(t, err)

	counts := make(map[int]int)
	for i := 0; i < testIterations; i++ {
		_, item := weightSlice.Choose()
		counts[item]++
	}

	require.Equal(t, counts[0], 0) // first weight should have count = 0 because the weight is 0
	// verify counts
	for i, item := range items[0 : len(items)-1] { // ignore last item because we always calculate nextItem item
		nextItem := items[i+1]
		current, next := item.Item, nextItem.Item
		require.Less(t, counts[current], counts[next])
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
