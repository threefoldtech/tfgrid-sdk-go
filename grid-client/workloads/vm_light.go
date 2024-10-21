// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"encoding/json"
	"fmt"
	"net"
	"slices"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
	"github.com/threefoldtech/zos4/pkg/gridtypes"
)

// VMLight is a virtual machine struct
type VMLight struct {
	Name          string `json:"name"`
	NodeID        uint32 `json:"node"`
	NetworkName   string `json:"network_name"`
	Description   string `json:"description"`
	Flist         string `json:"flist"`
	FlistChecksum string `json:"flist_checksum"`
	Entrypoint    string `json:"entrypoint"`
	Corex         bool   `json:"corex"`
	IP            string `json:"ip"`
	// used to get the same mycelium ip for the vm.
	MyceliumIPSeed []byte            `json:"mycelium_ip_seed"`
	GPUs           []zos.GPU         `json:"gpus"`
	CPU            uint8             `json:"cpu"`
	MemoryMB       uint64            `json:"memory"`
	RootfsSizeMB   uint64            `json:"rootfs_size"`
	Mounts         []Mount           `json:"mounts"`
	Zlogs          []Zlog            `json:"zlogs"`
	EnvVars        map[string]string `json:"env_vars"`

	// OUTPUT
	MyceliumIP string `json:"mycelium_ip"`
	ConsoleURL string `json:"console_url"`
}

// NewVMLightFromWorkload generates a new vm from given workloads and deployment
func NewVMLightFromWorkload(wl *zos.Workload, dl *zos.Deployment, nodeID uint32) (VMLight, error) {
	data, err := wl.ZMachineLightWorkload()
	if err != nil {
		return VMLight{}, errors.Errorf("could not create zmachine light workload from data")
	}

	var result zos.ZMachineLightResult

	if err := json.Unmarshal(wl.Result.Data, &result); err != nil {
		return VMLight{}, errors.Wrap(err, "failed to get vm result")
	}

	var myceliumIPSeed []byte
	if data.Network.Mycelium != nil {
		myceliumIPSeed = data.Network.Mycelium.Seed
	}

	flistCheckSum, err := GetFlistChecksum(data.FList)
	if err != nil {
		return VMLight{}, errors.Wrap(err, "failed to get flist checksum")
	}

	var dataGPUs []zos.GPU
	for _, g := range data.GPU {
		dataGPUs = append(dataGPUs, zos.GPU(g))
	}

	var dataMounts []zos.MachineMount
	for _, m := range data.Mounts {
		dataMounts = append(dataMounts, zos.MachineMount{
			Name:       m.Name.String(),
			Mountpoint: m.Mountpoint,
		})
	}

	return VMLight{
		Name:           wl.Name,
		NodeID:         nodeID,
		Description:    wl.Description,
		Flist:          data.FList,
		FlistChecksum:  flistCheckSum,
		Corex:          data.Corex,
		MyceliumIP:     result.MyceliumIP,
		MyceliumIPSeed: myceliumIPSeed,
		IP:             data.Network.Interfaces[0].IP.String(),
		CPU:            data.ComputeCapacity.CPU,
		GPUs:           dataGPUs,
		MemoryMB:       uint64(data.ComputeCapacity.Memory) / zos.Megabyte,
		RootfsSizeMB:   uint64(data.Size) / zos.Megabyte,
		Entrypoint:     data.Entrypoint,
		Mounts:         mounts(dataMounts),
		Zlogs:          zlogs(dl, wl.Name),
		EnvVars:        data.Env,
		NetworkName:    string(data.Network.Interfaces[0].Network),
		ConsoleURL:     result.ConsoleURL,
	}, nil
}

// ZosWorkload generates zos vm workloads
func (vm *VMLight) ZosWorkload() []zos.Workload {
	var workloads []zos.Workload

	var mounts []zos.MachineMount
	for _, mount := range vm.Mounts {
		mounts = append(mounts, zos.MachineMount{Name: mount.Name, Mountpoint: mount.MountPoint})
	}
	for _, zlog := range vm.Zlogs {
		zlogWorkload := zlog.ZosWorkload()
		workloads = append(workloads, zlogWorkload)
	}
	var myceliumIP *zos.MyceliumIP
	if len(vm.MyceliumIPSeed) != 0 {
		myceliumIP = &zos.MyceliumIP{
			Network: vm.NetworkName,
			Seed:    vm.MyceliumIPSeed,
		}
	}
	workload := zos.Workload{
		Version: 0,
		Name:    vm.Name,
		Type:    zos.ZMachineLightType,
		Data: zos.MustMarshal(zos.ZMachineLight{
			FList: vm.Flist,
			Network: zos.MachineNetworkLight{
				Interfaces: []zos.MachineInterface{
					{
						Network: vm.NetworkName,
						IP:      net.ParseIP(vm.IP),
					},
				},
				Mycelium: myceliumIP,
			},
			ComputeCapacity: zos.MachineCapacity{
				CPU:    vm.CPU,
				Memory: vm.MemoryMB * zos.Megabyte,
			},
			Size:       vm.RootfsSizeMB * zos.Megabyte,
			GPU:        vm.GPUs,
			Entrypoint: vm.Entrypoint,
			Corex:      vm.Corex,
			Mounts:     mounts,
			Env:        vm.EnvVars,
		}),
		Description: vm.Description,
	}
	workloads = append(workloads, workload)

	return workloads
}

// Validate validates a virtual machine data
func (vm *VMLight) Validate() error {
	if err := validateName(vm.Name); err != nil {
		return errors.Wrap(err, "virtual machine name is invalid")
	}

	if err := validateName(vm.NetworkName); err != nil {
		return errors.Wrap(err, "network name is invalid")
	}

	if vm.NodeID == 0 {
		return fmt.Errorf("node ID should be a positive integer not zero")
	}

	if len(strings.TrimSpace(vm.IP)) != 0 {
		if ip := net.ParseIP(vm.IP); ip == nil {
			return fmt.Errorf("invalid ip '%s'", vm.IP)
		}
	}

	if err := ValidateFlist(vm.Flist, vm.FlistChecksum); err != nil {
		return errors.Wrap(err, "flist is invalid")
	}

	if vm.CPU < 1 || vm.CPU > 32 {
		return errors.New("CPUs must be more than or equal to 1 and less than or equal to 32")
	}

	if gridtypes.Unit(vm.MemoryMB) < 250 {
		return fmt.Errorf("memory capacity can't be less that 250 MB")
	}

	minRoot := vm.MinRootSize()
	if vm.RootfsSizeMB != 0 && uint64(gridtypes.Unit(vm.RootfsSizeMB)*gridtypes.Megabyte) < minRoot {
		return fmt.Errorf("rootfs size can't be less than %d. Set to 0 for minimum", minRoot)
	}

	for _, g := range vm.GPUs {
		_, _, _, err := g.Parts()
		if err != nil {
			return errors.Wrap(err, "failed to validate GPUs")
		}
	}

	if len(vm.MyceliumIPSeed) != zos.MyceliumIPSeedLen && len(vm.MyceliumIPSeed) != 0 {
		return fmt.Errorf("invalid mycelium ip seed length %d must be %d or empty", len(vm.MyceliumIPSeed), zos.MyceliumIPSeedLen)
	}

	for _, zlog := range vm.Zlogs {
		if err := zlog.Validate(); err != nil {
			return errors.Wrap(err, "invalid zlog")
		}
	}

	for _, mount := range vm.Mounts {
		if err := mount.Validate(); err != nil {
			return errors.Wrap(err, "invalid mount")
		}
	}

	return nil
}

func (vm *VMLight) MinRootSize() uint64 {
	// sru = (cpu * mem_in_gb) / 8
	// each 1 SRU is 50GB of storage
	sru := gridtypes.Unit(vm.CPU) * gridtypes.Unit(vm.MemoryMB) / (8 * gridtypes.Gigabyte)

	if sru == 0 {
		return uint64(500 * gridtypes.Megabyte)
	}

	return uint64(2 * gridtypes.Gigabyte)
}

// LoadFromVM compares the vm with another given vm
func (vm *VMLight) LoadFromVM(vm2 *VMLight) {
	l := len(vm2.Zlogs) + len(vm2.Mounts)
	names := make(map[string]int)
	for idx, zlog := range vm2.Zlogs {
		names[zlog.Output] = idx - l
	}
	for idx, mount := range vm2.Mounts {
		names[mount.Name] = idx - l
	}
	sort.Slice(vm.Zlogs, func(i, j int) bool {
		return names[vm.Zlogs[i].Output] < names[vm.Zlogs[j].Output]
	})
	sort.Slice(vm.Mounts, func(i, j int) bool {
		return names[vm.Mounts[i].Name] < names[vm.Mounts[j].Name]
	})
	vm.FlistChecksum = vm2.FlistChecksum
}

func (vm *VMLight) AssignPrivateIP(
	networkName,
	ipRange string,
	nodeID uint32,
	ipRangeCIDR *net.IPNet,
	ip net.IP,
	curHostID byte,
	usedHosts map[string]map[uint32][]byte,
) (string, error) {
	vmIP := net.ParseIP(vm.IP).To4()

	// if vm private ip is given
	if vmIP != nil {
		vmHostID := vmIP[3] // host ID of the private ip

		nodeUsedHostIDs := usedHosts[networkName][nodeID]

		// TODO: use of a duplicate IP vs an updated vm with a new/old IP
		if slices.Contains(nodeUsedHostIDs, vmHostID) {
			// return "", fmt.Errorf("duplicate private ip '%v' in vm '%s' is used", vmIP, vm.Name)
			return vmIP.String(), nil
		}

		if !ipRangeCIDR.Contains(vmIP) {
			return "", fmt.Errorf("deployment ip range '%v' doesn't contain ip '%v' for vm '%s'", ipRange, vmIP, vm.Name)
		}

		usedHosts[networkName][nodeID] = append(usedHosts[networkName][nodeID], vmHostID)
		return vmIP.String(), nil
	}

	nodeUsedHostIDs := usedHosts[networkName][nodeID]

	// try to find available host ID in the deployment ip range
	for slices.Contains(nodeUsedHostIDs, curHostID) {
		if curHostID == 254 {
			return "", errors.New("all 253 ips of the network are exhausted")
		}
		curHostID++
	}

	usedHosts[networkName][nodeID] = append(usedHosts[networkName][nodeID], curHostID)

	vmIP = ip.To4()
	vmIP[3] = curHostID

	return vmIP.String(), nil
}
