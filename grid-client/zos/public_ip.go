package zos

type PublicIP struct {
	// V4 use one of the reserved Ipv4 from your contract. The Ipv4
	// itself costs money + the network traffic
	V4 bool `json:"v4"`
	// V6 get an ipv6 for the VM. this is for free
	// but the consumed capacity (network traffic) is not
	V6 bool `json:"v6"`
}
