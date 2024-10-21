// Package integration for integration tests
package integration

import (
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

var (
	minRootfs uint64 = 2
	minMemory uint64 = 2
	minCPU    uint8  = 2
)

type nodeFilterCfg struct {
	freeSRU  uint64
	freeHRU  uint64
	freeMRU  uint64
	freeIPs  uint64
	ipv4     bool
	ipv6     bool
	domain   bool
	hasGPU   bool
	rentedBy uint64
	features []string
}

type nodeFilterOpts func(*nodeFilterCfg)

func WithFeatures(features []string) nodeFilterOpts {
	return func(p *nodeFilterCfg) {
		p.features = features
	}
}

func WithFreeSRU(sruGB uint64) nodeFilterOpts {
	return func(p *nodeFilterCfg) {
		p.freeSRU = sruGB
	}
}

func WithFreeMRU(mruGB uint64) nodeFilterOpts {
	return func(p *nodeFilterCfg) {
		p.freeMRU = mruGB
	}
}

func WithFreeHRU(hruGB uint64) nodeFilterOpts {
	return func(p *nodeFilterCfg) {
		p.freeHRU = hruGB
	}
}

func WithFreeIPs(ipsCount uint64) nodeFilterOpts {
	return func(p *nodeFilterCfg) {
		p.freeIPs = ipsCount
	}
}

func WithIPV4() nodeFilterOpts {
	return func(p *nodeFilterCfg) {
		p.ipv4 = true
	}
}

func WithIPV6() nodeFilterOpts {
	return func(p *nodeFilterCfg) {
		p.ipv6 = true
	}
}

func WithDomain() nodeFilterOpts {
	return func(p *nodeFilterCfg) {
		p.domain = true
	}
}

func WithGPU() nodeFilterOpts {
	return func(p *nodeFilterCfg) {
		p.hasGPU = true
	}
}

func WithRentedBy(rentedBy uint64) nodeFilterOpts {
	return func(p *nodeFilterCfg) {
		p.rentedBy = rentedBy
	}
}

func generateNodeFilter(opts ...nodeFilterOpts) types.NodeFilter {
	cfg := nodeFilterCfg{
		freeMRU: minMemory,
	}

	for _, o := range opts {
		o(&cfg)
	}

	nodeFilter := types.NodeFilter{
		FarmIDs:  []uint64{1}, // freefarm is used in tests
		Status:   []string{"up"},
		FreeSRU:  convertGBToBytes(cfg.freeSRU + minRootfs),
		Features: cfg.features,
	}

	if cfg.freeHRU > 0 {
		nodeFilter.FreeHRU = convertGBToBytes(cfg.freeHRU)
	}

	if cfg.freeMRU > 0 {
		nodeFilter.FreeMRU = convertGBToBytes(cfg.freeMRU)
	}

	if cfg.freeIPs > 0 {
		nodeFilter.FreeIPs = &cfg.freeIPs
	}

	if cfg.ipv4 {
		nodeFilter.IPv4 = &cfg.ipv4
	}

	if cfg.ipv6 {
		nodeFilter.IPv6 = &cfg.ipv6
	}

	if cfg.domain {
		nodeFilter.Domain = &cfg.domain
	}

	if cfg.hasGPU {
		nodeFilter.HasGPU = &cfg.hasGPU
	}

	if cfg.rentedBy > 0 {
		nodeFilter.RentedBy = &cfg.rentedBy
	}

	return nodeFilter
}

func convertGBToBytes(gb uint64) *uint64 {
	bytes := gb * 1024 * 1024 * 1024
	return &bytes
}
