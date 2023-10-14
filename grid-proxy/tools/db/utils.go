package main

import (
	crand "crypto/rand"
	"math/rand"
	"net"
)

func rnd(min, max uint64) uint64 {
	return rand.Uint64()%(max-min+1) + min
}

func flip(success float32) bool {
	return rand.Float32() < success
}

func randomIPv4() (net.IP, error) {
	ip := make([]byte, 4)
	if _, err := crand.Read(ip); err != nil {
		return nil, err
	}

	return net.IP(ip), nil
}

// IPv4Subnet gets the ipv4 subnet given the ip
func IPv4Subnet(ip net.IP) *net.IPNet {
	return &net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(24, 32),
	}
}

func min(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}
func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
