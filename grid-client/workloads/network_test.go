// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"fmt"
	"net"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
)

// Network
var n = ZNet{
	Name:        "testingNetwork",
	Description: "network for testing",
	Nodes:       []uint32{1},
	IPRange: zos.IPNet{IPNet: net.IPNet{
		IP:   net.IPv4(10, 20, 0, 0),
		Mask: net.CIDRMask(16, 32),
	}},
	AddWGAccess: false,
}

func TestNetwork(t *testing.T) {
	t.Run("test_ip_net", func(t *testing.T) {
		ip := IPNet(10, 20, 0, 0, 16)
		assert.Equal(t, ip, n.IPRange)
	})

	t.Run("test_wg_ip", func(t *testing.T) {
		wgIP := WgIP(n.IPRange)

		wgIPRange, err := zos.ParseIPNet("100.64.20.0/32")
		assert.NoError(t, err)

		assert.Equal(t, wgIP, wgIPRange)
	})

	t.Run("test_generate_wg_config", func(t *testing.T) {
		config := GenerateWGConfig(
			"", "", "", "",
			n.IPRange.String(),
		)

		assert.Equal(t, config, strings.ReplaceAll(fmt.Sprintf(`
			[Interface]
			Address = %s
			PrivateKey = %s
			[Peer]
			PublicKey = %s
			AllowedIPs = %s, 100.64.0.0/16
			PersistentKeepalive = 25
			Endpoint = %s
			`, "", "", "", n.IPRange.String(), ""), "\t", "")+"\t",
		)
	})
}
