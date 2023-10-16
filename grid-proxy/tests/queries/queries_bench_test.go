package test

import (
	"testing"

	"github.com/stretchr/testify/require"
	proxytypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

func BenchmarkNodes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		agg := calcNodesAggregates(&data)
		l := proxytypes.Limit{
			Size:     999999999999,
			Page:     1,
			RetCount: true,
		}
		f, err := randomNodeFilter(&agg)
		require.NoError(b, err)

		b.StartTimer()
		_, _, err = DBClient.GetNodes(f, l)
		require.NoError(b, err)
	}
}

func BenchmarkFarms(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		agg := calcFarmsAggregates(&data)
		l := proxytypes.Limit{
			Size:     999999999999,
			Page:     1,
			RetCount: true,
		}
		f, err := randomFarmsFilter(&agg)
		require.NoError(b, err)

		b.StartTimer()
		_, _, err = DBClient.GetFarms(f, l)
		require.NoError(b, err)
	}
}

func BenchmarkContracts(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		agg := calcContractsAggregates(&data)
		l := proxytypes.Limit{
			Size:     999999999999,
			Page:     1,
			RetCount: true,
		}
		f, err := randomContractsFilter(&agg)
		require.NoError(b, err)

		b.StartTimer()
		_, _, err = DBClient.GetContracts(f, l)
		require.NoError(b, err)
	}
}

func BenchmarkTwins(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		agg := calcTwinsAggregates(&data)
		l := proxytypes.Limit{
			Size:     999999999999,
			Page:     1,
			RetCount: true,
		}
		f, err := randomTwinsFilter(&agg)
		require.NoError(b, err)

		b.StartTimer()
		_, _, err = DBClient.GetTwins(f, l)
		require.NoError(b, err)
	}
}
