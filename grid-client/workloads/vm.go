// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net"
	"sort"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// ErrInvalidInput for invalid inputs
var ErrInvalidInput = errors.New("invalid input")

// VM is a virtual machine struct
type VM struct {
	Name          string `json:"name"`
	Flist         string `json:"flist"`
	FlistChecksum string `json:"flist_checksum"`
	PublicIP      bool   `json:"publicip"`
	PublicIP6     bool   `json:"publicip6"`
	Planetary     bool   `json:"planetary"`
	Corex         bool   `json:"corex"` //TODO: Is it works ??
	ComputedIP    string `json:"computedip"`
	ComputedIP6   string `json:"computedip6"`
	PlanetaryIP   string `json:"planetary_ip"`
	IP            string `json:"ip"`
	// used to get the same mycelium ip for the vm. if not set and planetary is used
	// it will fallback to yggdrasil.
	MyceliumIPSeed []byte            `json:"mycelium_ip_seed"`
	Description    string            `json:"description"`
	GPUs           []zos.GPU         `json:"gpus"`
	CPU            int               `json:"cpu"`
	Memory         int               `json:"memory"`
	RootfsSize     int               `json:"rootfs_size"`
	Entrypoint     string            `json:"entrypoint"`
	Mounts         []Mount           `json:"mounts"`
	Zlogs          []Zlog            `json:"zlogs"`
	EnvVars        map[string]string `json:"env_vars"`
	NetworkName    string            `json:"network_name"`
	ConsoleURL     string            `json:"console_url"`
}

// Mount disks struct
type Mount struct {
	DiskName   string `json:"disk_name"`
	MountPoint string `json:"mount_point"`
}

// NewVMFromWorkload generates a new vm from given workloads and deployment
func NewVMFromWorkload(wl *gridtypes.Workload, dl *gridtypes.Deployment) (VM, error) {
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

	return VM{
		Name:          wl.Name.String(),
		Description:   wl.Description,
		Flist:         data.FList,
		FlistChecksum: "",
		PublicIP:      !pubIPRes.IP.Nil(),
		ComputedIP:    pubIP4,
		PublicIP6:     !pubIPRes.IPv6.Nil(),
		ComputedIP6:   pubIP6,
		Planetary:     result.PlanetaryIP != "",
		Corex:         data.Corex,
		PlanetaryIP:   result.PlanetaryIP,
		IP:            data.Network.Interfaces[0].IP.String(),
		CPU:           int(data.ComputeCapacity.CPU),
		GPUs:          data.GPU,
		Memory:        int(data.ComputeCapacity.Memory / gridtypes.Megabyte),
		RootfsSize:    int(data.Size / gridtypes.Megabyte),
		Entrypoint:    data.Entrypoint,
		Mounts:        mounts(data.Mounts),
		Zlogs:         zlogs(dl, wl.Name.String()),
		EnvVars:       data.Env,
		NetworkName:   string(data.Network.Interfaces[0].Network),
		ConsoleURL:    result.ConsoleURL,
	}, nil
}

func mounts(mounts []zos.MachineMount) []Mount {
	var res []Mount
	for _, mount := range mounts {
		res = append(res, Mount{
			DiskName:   mount.Name.String(),
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
		mounts = append(mounts, zos.MachineMount{Name: gridtypes.Name(mount.DiskName), Mountpoint: mount.MountPoint})
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
				CPU:    uint8(vm.CPU),
				Memory: gridtypes.Unit(uint(vm.Memory)) * gridtypes.Megabyte,
			},
			Size:       gridtypes.Unit(vm.RootfsSize) * gridtypes.Megabyte,
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
// cpu: from 1:32
// checks if the given flistChecksum equals the checksum of the given flist
func (vm *VM) Validate() error {
	if vm.CPU < 1 || vm.CPU > 32 {
		return errors.Wrap(ErrInvalidInput, "CPUs must be more than or equal to 1 and less than or equal to 32")
	}

	for _, g := range vm.GPUs {
		_, _, _, err := g.Parts()
		if err != nil {
			return errors.Wrap(ErrInvalidInput, "failed to validate GPUs")
		}
	}

	if vm.FlistChecksum != "" {
		checksum, err := GetFlistChecksum(vm.Flist)
		if err != nil {
			return errors.Wrap(err, "failed to get flist checksum")
		}
		if vm.FlistChecksum != checksum {
			return errors.Errorf(
				"passed checksum %s of %s does not match %s returned from %s",
				vm.FlistChecksum,
				vm.Name,
				checksum,
				FlistChecksumURL(vm.Flist),
			)
		}
	}
	if len(vm.MyceliumIPSeed) != zos.MyceliumIPSeedLen && len(vm.MyceliumIPSeed) != 0 {
		return fmt.Errorf("invalid mycelium ip seed length %d must be %d or empty", len(vm.MyceliumIPSeed), zos.MyceliumIPSeedLen)
	}
	return nil
}

// LoadFromVM compares the vm with another given vm
func (vm *VM) LoadFromVM(vm2 *VM) {
	l := len(vm2.Zlogs) + len(vm2.Mounts)
	names := make(map[string]int)
	for idx, zlog := range vm2.Zlogs {
		names[zlog.Output] = idx - l
	}
	for idx, mount := range vm2.Mounts {
		names[mount.DiskName] = idx - l
	}
	sort.Slice(vm.Zlogs, func(i, j int) bool {
		return names[vm.Zlogs[i].Output] < names[vm.Zlogs[j].Output]
	})
	sort.Slice(vm.Mounts, func(i, j int) bool {
		return names[vm.Mounts[i].DiskName] < names[vm.Mounts[j].DiskName]
	})
	vm.FlistChecksum = vm2.FlistChecksum
}

func RandomMyceliumIPSeed() ([]byte, error) {
	key := make([]byte, zos.MyceliumIPSeedLen)
	_, err := rand.Read(key)
	return key, err
}
