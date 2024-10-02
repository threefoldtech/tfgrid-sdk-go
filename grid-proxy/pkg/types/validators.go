package types

import (
	"fmt"
	"slices"
)

var (
	Zos3NodesFeatures = []string{"zmount", "zdb", "volume", "ipv4", "ip", "gateway-name-proxy",
		"gateway-fqdn-proxy", "qsfs", "zlogs", "network", "zmachine"}
	Zos4NodesFeatures = []string{"zmount", "zdb", "volume", "ipv4", "ip", "gateway-name-proxy",
		"gateway-fqdn-proxy", "qsfs", "zlogs", "zmachine-light", "network-light"}
	FeaturesSet = []string{"zmount", "zdb", "volume", "ipv4", "ip", "gateway-name-proxy",
		"gateway-fqdn-proxy", "qsfs", "zlogs", "network", "zmachine", "zmachine-light", "network-light"}
)

func validateNodeFeatures(features []string) error {
	for _, feat := range features {
		if !slices.Contains(FeaturesSet, feat) {
			return fmt.Errorf("%s is not a valid node feature", feat)
		}
	}
	return nil
}
