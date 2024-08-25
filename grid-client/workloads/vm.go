// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// VM is a virtual machine struct
type VM struct {
	Name        string `json:"name"`
	NodeID      uint32 `json:"node"`
	NetworkName string `json:"network_name"`
	Description string `json:"description"`
	Flist       string `json:"flist"`
	Entrypoint  string `json:"entrypoint"`
	PublicIP    bool   `json:"publicip"`
	PublicIP6   bool   `json:"publicip6"`
	Planetary   bool   `json:"planetary"`
	Corex       bool   `json:"corex"` // TODO: Is it works ??
	IP          string `json:"ip"`
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
func NewVMFromWorkload(wl *gridtypes.Workload, dl *gridtypes.Deployment, nodeID uint32) (VM, error) {
	dataI, err := wl.WorkloadData()
	if err != nil {
		return VM{}, errors.Wrap(err, "failed to get workload data")
	}

	data, ok := dataI.(*zos.ZMachine)
	if !ok {
		return VM{}, errors.Errorf("could not create vm workload from data %v", dataI)
	}

	var result zos.ZMachineResult

	if err := json.Unmarshal(wl.Result.Data, &result); err != nil {
		return VM{}, errors.Wrap(err, "failed to get vm result")
	}

	var pubIPRes zos.PublicIPResult
	if !data.Network.PublicIP.IsEmpty() {
		pubIPRes, err = pubIP(dl, data.Network.PublicIP)
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

	return VM{
		Name:           wl.Name.String(),
		NodeID:         nodeID,
		Description:    wl.Description,
		Flist:          data.FList,
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
		GPUs:           data.GPU,
		MemoryMB:       uint64(data.ComputeCapacity.Memory / gridtypes.Megabyte),
		RootfsSizeMB:   uint64(data.Size / gridtypes.Megabyte),
		Entrypoint:     data.Entrypoint,
		Mounts:         mounts(data.Mounts),
		Zlogs:          zlogs(dl, wl.Name.String()),
		EnvVars:        data.Env,
		NetworkName:    string(data.Network.Interfaces[0].Network),
		ConsoleURL:     result.ConsoleURL,
	}, nil
}

func mounts(mounts []zos.MachineMount) []Mount {
	var res []Mount
	for _, mount := range mounts {
		res = append(res, Mount{
			Name:       mount.Name.String(),
			MountPoint: mount.Mountpoint,
		})
	}
	return res
}

func pubIP(dl *gridtypes.Deployment, name gridtypes.Name) (zos.PublicIPResult, error) {
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
func (vm *VM) ZosWorkload() []gridtypes.Workload {
	var workloads []gridtypes.Workload

	publicIPName := ""
	if vm.PublicIP || vm.PublicIP6 {
		publicIPName = fmt.Sprintf("%sip", vm.Name)
		workloads = append(workloads, ConstructPublicIPWorkload(publicIPName, vm.PublicIP, vm.PublicIP6))
	}

	var mounts []zos.MachineMount
	for _, mount := range vm.Mounts {
		mounts = append(mounts, zos.MachineMount{Name: gridtypes.Name(mount.Name), Mountpoint: mount.MountPoint})
	}
	for _, zlog := range vm.Zlogs {
		zlogWorkload := zlog.ZosWorkload()
		workloads = append(workloads, zlogWorkload)
	}
	var myceliumIP *zos.MyceliumIP
	if len(vm.MyceliumIPSeed) != 0 {
		myceliumIP = &zos.MyceliumIP{
			Network: gridtypes.Name(vm.NetworkName),
			Seed:    vm.MyceliumIPSeed,
		}
	}
	workload := gridtypes.Workload{
		Version: 0,
		Name:    gridtypes.Name(vm.Name),
		Type:    zos.ZMachineType,
		Data: gridtypes.MustMarshal(zos.ZMachine{
			FList: vm.Flist,
			Network: zos.MachineNetwork{
				Interfaces: []zos.MachineInterface{
					{
						Network: gridtypes.Name(vm.NetworkName),
						IP:      net.ParseIP(vm.IP),
					},
				},
				PublicIP:  gridtypes.Name(publicIPName),
				Planetary: vm.Planetary,
				Mycelium:  myceliumIP,
			},
			ComputeCapacity: zos.MachineCapacity{
				CPU:    vm.CPU,
				Memory: gridtypes.Unit(uint(vm.MemoryMB)) * gridtypes.Megabyte,
			},
			Size:       gridtypes.Unit(vm.RootfsSizeMB) * gridtypes.Megabyte,
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

	if err := ValidateFlist(vm.Flist); err != nil {
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
}

func RandomMyceliumIPSeed() ([]byte, error) {
	key := make([]byte, zos.MyceliumIPSeedLen)
	_, err := rand.Read(key)
	return key, err
}
