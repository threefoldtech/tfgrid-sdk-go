package zos

const (
	MyceliumKeyLen    = 32
	MyceliumIPSeedLen = 6

	// Kilobyte unit multiplier
	Kilobyte uint64 = 1024
	// Megabyte unit multiplier
	Megabyte uint64 = 1024 * Kilobyte
	// Gigabyte unit multiplier
	Gigabyte uint64 = 1024 * Megabyte
	// Terabyte unit multiplier
	Terabyte uint64 = 1024 * Gigabyte
)

const (
	// ZMountType type
	ZMountType string = "zmount"
	// NetworkType type
	NetworkType string = "network"
	// NetworkLightType type
	NetworkLightType string = "network-light"
	// ZDBType type
	ZDBType string = "zdb"
	// ZMachineType type
	ZMachineType string = "zmachine"
	// ZMachineLightType type
	ZMachineLightType string = "zmachine-light"
	// VolumeType type
	VolumeType string = "volume"
	//PublicIPv4Type type [deprecated]
	PublicIPv4Type string = "ipv4"
	//PublicIPType type is the new way to assign public ips
	// to a VM. this has flags (V4, and V6) that has to be set.
	PublicIPType string = "ip"
	// GatewayNameProxyType type
	GatewayNameProxyType string = "gateway-name-proxy"
	// GatewayFQDNProxyType type
	GatewayFQDNProxyType string = "gateway-fqdn-proxy"
	// QuantumSafeFSType type
	QuantumSafeFSType string = "qsfs"
	// ZLogsType type
	ZLogsType string = "zlogs"
)

const (
	// StateInit is the first state of the workload on storage
	StateInit ResultState = "init"
	// StateUnChanged is a special error state it means there was an error
	// running the action, but this error did not break previous state.
	StateUnChanged ResultState = "unchanged"
	// StateError constant
	StateError ResultState = "error"
	// StateOk constant
	StateOk ResultState = "ok"
	// StateDeleted constant
	StateDeleted ResultState = "deleted"
	// StatePaused constant
	StatePaused ResultState = "paused"
)
