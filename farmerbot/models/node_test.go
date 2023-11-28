// // Package models for farmerbot models.
package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func TestNodeModel(t *testing.T) {
	node := Node{
		Node: substrate.Node{ID: 1, TwinID: 1},
		Resources: ConsumableResources{
			OverProvisionCPU: 1,
			Total:            cap,
		},
		PowerState: ON,
	}

	t.Run("test update node resources", func(t *testing.T) {
		zosResources := ZosResourcesStatistics{
			Total: gridtypes.Capacity{
				CRU:   cap.CRU,
				SRU:   gridtypes.Unit(cap.SRU),
				HRU:   gridtypes.Unit(cap.HRU),
				MRU:   gridtypes.Unit(cap.MRU),
				IPV4U: 1,
			},
			Used:   gridtypes.Capacity{},
			System: gridtypes.Capacity{},
		}

		node.UpdateResources(zosResources)
		assert.True(t, node.Resources.Used.isEmpty())
		assert.True(t, node.IsUnused())
		assert.Equal(t, node.Resources.OverProvisionCPU, float32(1))
		assert.True(t, node.CanClaimResources(node.Resources.Total))

		node.ClaimResources(node.Resources.Total)
		assert.False(t, node.Resources.Used.isEmpty())
		assert.False(t, node.IsUnused())
		assert.False(t, node.CanClaimResources(node.Resources.Total))

		node.Resources.Used = Capacity{}
	})
}
