package types

import (
	"fmt"
	"slices"
)

var (
	Zos3NodesFeatures = []string{"zmount", "zdb", "volume", "ipv4", "ip", "gateway-name-proxy",
		"gateway-fqdn-proxy", "qsfs", "zlogs", "network", "zmachine", "wireguard", "yggdrasil", "mycelium"}
	Zos4NodesFeatures = []string{"zmount", "zdb", "volume", "gateway-name-proxy",
		"gateway-fqdn-proxy", "qsfs", "zlogs", "zmachine-light", "network-light", "mycelium"}
	FeaturesSet = []string{"zmount", "zdb", "volume", "ipv4", "ip", "gateway-name-proxy",
		"gateway-fqdn-proxy", "qsfs", "zlogs", "network", "zmachine", "zmachine-light", "network-light",
		"wireguard", "yggdrasil", "mycelium"}
)

func validateNodeFeatures(features []string) error {
	for _, feat := range features {
		if !slices.Contains(FeaturesSet, feat) {
			return fmt.Errorf("%s is not a valid node feature", feat)
		}
	}
	return nil
}
