// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
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
	Name          string            `json:"name"`
	Flist         string            `json:"flist"`
	FlistChecksum string            `json:"flist_checksum"`
	PublicIP      bool              `json:"publicip"`
	PublicIP6     bool              `json:"publicip6"`
	Planetary     bool              `json:"planetary"`
	Corex         bool              `json:"corex"` //TODO: Is it works ??
	ComputedIP    string            `json:"computedip"`
	ComputedIP6   string            `json:"computedip6"`
	YggIP         string            `json:"ygg_ip"`
	IP            string            `json:"ip"`
	Description   string            `json:"description"`
	GPUs          []zos.GPU         `json:"gpus"`
	CPU           int               `json:"cpu"`
	Memory        int               `json:"memory"`
	RootfsSize    int               `json:"rootfs_size"`
	Entrypoint    string            `json:"entrypoint"`
	Mounts        []Mount           `json:"mounts"`
	Zlogs         []Zlog            `json:"zlogs"`
	EnvVars       map[string]string `json:"env_vars"`
	NetworkName   string            `json:"network_name"`
}

// Mount disks struct
type Mount struct {
	DiskName   string `json:"disk_name"`
	MountPoint string `json:"mount_point"`
}

func NewVMFromMap(vm map[string]interface{}) (*VM, error) {
	zlogs := vm["zlogs"].([]interface{})
	for i, v := range zlogs {
		newVal := map[string]interface{}{}
		newVal["zmachine"] = vm["name"].(string)
		newVal["output"] = v.(string)
		zlogs[i] = newVal
	}

	mapBytes, err := json.Marshal(vm)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal vm map")
	}

	res := VM{}
	err = json.Unmarshal(mapBytes, &res)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal vm data")
	}

	return &res, nil
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

	pubIP := pubIP(dl, data.Network.PublicIP)
	var pubIP4, pubIP6 string

	if !pubIP.IP.Nil() {
		pubIP4 = pubIP.IP.String()
	}
	if !pubIP.IPv6.Nil() {
		pubIP6 = pubIP.IPv6.String()
	}

	return VM{
		Name:          wl.Name.String(),
		Description:   wl.Description,
		Flist:         data.FList,
		FlistChecksum: "",
		PublicIP:      !pubIP.IP.Nil(),
		ComputedIP:    pubIP4,
		PublicIP6:     !pubIP.IPv6.Nil(),
		ComputedIP6:   pubIP6,
		Planetary:     result.YggIP != "",
		Corex:         data.Corex,
		YggIP:         result.YggIP,
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

func pubIP(dl *gridtypes.Deployment, name gridtypes.Name) zos.PublicIPResult {

	pubIPWl, err := dl.Get(name)
	if err != nil || !pubIPWl.Workload.Result.State.IsOkay() {
		pubIPWl = nil
		return zos.PublicIPResult{}
	}
	var pubIPResult zos.PublicIPResult

	err = json.Unmarshal(pubIPWl.Result.Data, &pubIPResult)
	if err != nil {
		fmt.Println("error: ", err)
	}

	return pubIPResult
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

// ToMap converts vm data to a map (dict)
func (vm *VM) ToMap() (map[string]interface{}, error) {
	var vmMap map[string]interface{}
	vmBytes, err := json.Marshal(vm)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal vm data")
	}

	err = json.Unmarshal(vmBytes, &vmMap)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal vm bytes to map")
	}

	return vmMap, nil
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
