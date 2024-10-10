// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net"
	"slices"
	"sort"
	"strings"

	"github.com/pkg/errors"
	zosTypes "github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// VM is a virtual machine struct
type VM struct {
	Name          string `json:"name"`
	NodeID        uint32 `json:"node"`
	NetworkName   string `json:"network_name"`
	Description   string `json:"description"`
	Flist         string `json:"flist"`
	FlistChecksum string `json:"flist_checksum"`
	Entrypoint    string `json:"entrypoint"`
	PublicIP      bool   `json:"publicip"`
	PublicIP6     bool   `json:"publicip6"`
	Planetary     bool   `json:"planetary"`
	Corex         bool   `json:"corex"` // TODO: Is it works ??
	IP            string `json:"ip"`
	// used to get the same mycelium ip for the vm.
	MyceliumIPSeed []byte            `json:"mycelium_ip_seed"`
	GPUs           []zosTypes.GPU    `json:"gpus"`
	CPU            uint8             `json:"cpu"`
	MemoryMB       uint64            `json:"memory"`
	RootfsSizeMB   uint64            `json:"rootfs_size"`
	Mounts         []Mount           `json:"mounts"`
	Zlogs          []Zlog            `json:"zlogs"`
	EnvVars        map[string]string `json:"env_vars"`

	// OUTPUT
	ComputedIP  string `json:"computedip"`
	ComputedIP6 string `json:"computedip6"`
	PlanetaryIP string `json:"planetary_ip"`
	MyceliumIP  string `json:"mycelium_ip"`
	ConsoleURL  string `json:"console_url"`
}

// Mount disks/volumes struct
type Mount struct {
	Name       string `json:"name"`
	MountPoint string `yaml:"mount_point" json:"mount_point"`
}

func (m *Mount) Validate() error {
	if err := validateName(m.Name); err != nil {
		return errors.Wrap(err, "mount name is invalid")
	}

	return nil
}

// NewVMFromWorkload generates a new vm from given workloads and deployment
func NewVMFromWorkload(wl *zosTypes.Workload, dl *zosTypes.Deployment, nodeID uint32) (VM, error) {
	data, err := wl.ZMachineWorkload()
	if err != nil {
		return VM{}, errors.Errorf("could not create zmachine workload from data")
	}

	var result zosTypes.ZMachineResult

	if err := json.Unmarshal(wl.Result.Data, &result); err != nil {
		return VM{}, errors.Wrap(err, "failed to get vm result")
	}

	var pubIPRes zos.PublicIPResult
	if !data.Network.PublicIP.IsEmpty() {
		pubIPRes, err = pubIP(dl, data.Network.PublicIP.String())
		if err != nil {
			return VM{}, errors.Wrap(err, "failed to get public ip workload")
		}
	}

	var pubIP4, pubIP6 string

	if !pubIPRes.IP.Nil() {
		pubIP4 = pubIPRes.IP.String()
	}
	if !pubIPRes.IPv6.Nil() {
		pubIP6 = pubIPRes.IPv6.String()
	}

	var myceliumIPSeed []byte
	if data.Network.Mycelium != nil {
		myceliumIPSeed = data.Network.Mycelium.Seed
	}

	flistCheckSum, err := GetFlistChecksum(data.FList)
	if err != nil {
		return VM{}, errors.Wrap(err, "failed to get flist checksum")
	}

	var dataGPUs []zosTypes.GPU
	for _, g := range data.GPU {
		dataGPUs = append(dataGPUs, zosTypes.GPU(g))
	}

	var dataMounts []zosTypes.MachineMount
	for _, m := range data.Mounts {
		dataMounts = append(dataMounts, zosTypes.MachineMount{
			Name:       m.Name.String(),
			Mountpoint: m.Mountpoint,
		})
	}

	return VM{
		Name:           wl.Name,
		NodeID:         nodeID,
		Description:    wl.Description,
		Flist:          data.FList,
		FlistChecksum:  flistCheckSum,
		PublicIP:       !pubIPRes.IP.Nil(),
		ComputedIP:     pubIP4,
		PublicIP6:      !pubIPRes.IPv6.Nil(),
		ComputedIP6:    pubIP6,
		Planetary:      result.PlanetaryIP != "",
		Corex:          data.Corex,
		PlanetaryIP:    result.PlanetaryIP,
		MyceliumIP:     result.MyceliumIP,
		MyceliumIPSeed: myceliumIPSeed,
		IP:             data.Network.Interfaces[0].IP.String(),
		CPU:            data.ComputeCapacity.CPU,
		GPUs:           dataGPUs,
		MemoryMB:       uint64(data.ComputeCapacity.Memory / gridtypes.Megabyte),
		RootfsSizeMB:   uint64(data.Size / gridtypes.Megabyte),
		Entrypoint:     data.Entrypoint,
		Mounts:         mounts(dataMounts),
		Zlogs:          zlogs(dl, wl.Name),
		EnvVars:        data.Env,
		NetworkName:    string(data.Network.Interfaces[0].Network),
		ConsoleURL:     result.ConsoleURL,
	}, nil
}

func mounts(mounts []zosTypes.MachineMount) []Mount {
	var res []Mount
	for _, mount := range mounts {
		res = append(res, Mount{
			Name:       mount.Name,
			MountPoint: mount.Mountpoint,
		})
	}
	return res
}

func pubIP(dl *zosTypes.Deployment, name string) (zos.PublicIPResult, error) {
	pubIPWl, err := dl.Get(name)
	if err != nil || !pubIPWl.Workload.Result.State.IsOkay() {
		pubIPWl = nil
		return zos.PublicIPResult{}, err
	}

	var pubIPResult zos.PublicIPResult
	bytes, err := json.Marshal(pubIPWl.Result.Data)
	if err != nil {
		return zos.PublicIPResult{}, err
	}

	err = json.Unmarshal(bytes, &pubIPResult)
	if err != nil {
		return zos.PublicIPResult{}, err
	}

	return pubIPResult, nil
}

// ZosWorkload generates zos vm workloads
func (vm *VM) ZosWorkload() []zosTypes.Workload {
	var workloads []zosTypes.Workload

	publicIPName := ""
	if vm.PublicIP || vm.PublicIP6 {
		publicIPName = fmt.Sprintf("%sip", vm.Name)
		workloads = append(workloads, ConstructPublicIPWorkload(publicIPName, vm.PublicIP, vm.PublicIP6))
	}

	var mounts []zosTypes.MachineMount
	for _, mount := range vm.Mounts {
		mounts = append(mounts, zosTypes.MachineMount{Name: mount.Name, Mountpoint: mount.MountPoint})
	}
	for _, zlog := range vm.Zlogs {
		zlogWorkload := zlog.ZosWorkload()
		workloads = append(workloads, zlogWorkload)
	}
	var myceliumIP *zosTypes.MyceliumIP
	if len(vm.MyceliumIPSeed) != 0 {
		myceliumIP = &zosTypes.MyceliumIP{
			Network: vm.NetworkName,
			Seed:    vm.MyceliumIPSeed,
		}
	}
	workload := zosTypes.Workload{
		Version: 0,
		Name:    vm.Name,
		Type:    zosTypes.ZMachineType,
		Data: zosTypes.MustMarshal(zosTypes.ZMachine{
			FList: vm.Flist,
			Network: zosTypes.MachineNetwork{
				Interfaces: []zosTypes.MachineInterface{
					{
						Network: vm.NetworkName,
						IP:      net.ParseIP(vm.IP),
					},
				},
				PublicIP:  publicIPName,
				Planetary: vm.Planetary,
				Mycelium:  myceliumIP,
			},
			ComputeCapacity: zosTypes.MachineCapacity{
				CPU:    vm.CPU,
				Memory: vm.MemoryMB * uint64(gridtypes.Megabyte),
			},
			Size:       vm.RootfsSizeMB * uint64(gridtypes.Megabyte),
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
func (vm *VM) Validate() error {
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

	if len(vm.MyceliumIPSeed) != zosTypes.MyceliumIPSeedLen && len(vm.MyceliumIPSeed) != 0 {
		return fmt.Errorf("invalid mycelium ip seed length %d must be %d or empty", len(vm.MyceliumIPSeed), zosTypes.MyceliumIPSeedLen)
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

func (vm *VM) MinRootSize() uint64 {
	// sru = (cpu * mem_in_gb) / 8
	// each 1 SRU is 50GB of storage
	sru := gridtypes.Unit(vm.CPU) * gridtypes.Unit(vm.MemoryMB) / (8 * gridtypes.Gigabyte)

	if sru == 0 {
		return uint64(500 * gridtypes.Megabyte)
	}

	return uint64(2 * gridtypes.Gigabyte)
}

// LoadFromVM compares the vm with another given vm
func (vm *VM) LoadFromVM(vm2 *VM) {
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

func (vm *VM) AssignPrivateIP(
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

func RandomMyceliumIPSeed() ([]byte, error) {
	key := make([]byte, zosTypes.MyceliumIPSeedLen)
	_, err := rand.Read(key)
	return key, err
}
