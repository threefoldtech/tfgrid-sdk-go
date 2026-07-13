package indexer

import (
	"reflect"
	"testing"

	"github.com/threefoldtech/zos_sdk_go/grid-proxy/pkg/types"
)

func TestRemoveDuplicates(t *testing.T) {
	t.Run("remove duplicate countries", func(t *testing.T) {
		locations := []types.NodeLocation{
			{Country: "Egypt", Continent: "Africa"},
			{Country: "Egypt", Continent: "Africa"},
			{Country: "Belgium", Continent: "Europe"},
		}

		uniqueLocations := []types.NodeLocation{
			{Country: "Egypt", Continent: "Africa"},
			{Country: "Belgium", Continent: "Europe"},
		}

		gotLocations := removeDuplicates(locations, func(n types.NodeLocation) string {
			return n.Country
		})

		if !reflect.DeepEqual(uniqueLocations, gotLocations) {
			t.Errorf("expected %v, but got %v", uniqueLocations, gotLocations)
		}
	})

	t.Run("remove duplicate health reports", func(t *testing.T) {
		healthReports := []types.HealthReport{
			{NodeTwinId: 1, Healthy: true},
			{NodeTwinId: 1, Healthy: true},
			{NodeTwinId: 2, Healthy: true},
		}

		uniqueReports := []types.HealthReport{
			{NodeTwinId: 1, Healthy: true},
			{NodeTwinId: 2, Healthy: true},
		}

		gotReports := removeDuplicates(healthReports, func(h types.HealthReport) uint32 {
			return h.NodeTwinId
		})

		if !reflect.DeepEqual(gotReports, uniqueReports) {
			t.Errorf("expected %v, but got %v", uniqueReports, gotReports)
		}
	})
}
