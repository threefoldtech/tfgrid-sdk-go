package zos

import (
	"github.com/pkg/errors"
	"github.com/threefoldtech/zos4/pkg/gridtypes/zos"
)

type ZMachineLight struct {
	// Flist of the zmachine, must be a valid url to an flist.
	FList string `json:"flist"`
	// Network configuration for machine network
	Network MachineNetworkLight `json:"network"`
	// Size of zmachine disk
	Size uint64 `json:"size"`
	// ComputeCapacity configuration for machine cpu+memory
	ComputeCapacity MachineCapacity `json:"compute_capacity"`
	// Mounts configure mounts/disks attachments to this machine
	Mounts []MachineMount `json:"mounts"`

	// following items are only available in container mode. if FList is for a container
	// not a VM.

	// Entrypoint entrypoint of the container, if not set the configured one from the flist
	// is going to be used
	Entrypoint string `json:"entrypoint"`
	// Env variables available for a container
	Env map[string]string `json:"env"`
	// Corex works in container mode which forces replace the
	// entrypoint of the container to use `corex`
	Corex bool `json:"corex"`

	// GPU attached to the VM
	// the list of the GPUs ids must:
	// - Exist, obviously
	// - Not used by other VMs
	// - Only possible on `dedicated` nodes
	GPU []GPU `json:"gpu,omitempty"`
}

// MachineNetworkLight structure
type MachineNetworkLight struct {
	// Mycelium IP config, if planetary is true, but Mycelium is not set we fall back
	// to yggdrasil support. Otherwise (if mycelium is set) a mycelium ip is used instead.
	Mycelium *MyceliumIP `json:"mycelium,omitempty"`

	// Interfaces list of user zNets to join
	Interfaces []MachineInterface `json:"interfaces"`
}

// ZMachineLightResult result returned by VM reservation
type ZMachineLightResult struct {
	ID         string `json:"id"`
	IP         string `json:"ip"`
	MyceliumIP string `json:"mycelium_ip"`
	ConsoleURL string `json:"console_url"`
}

func (wl *Workload) ZMachineLightWorkload() (*zos.ZMachineLight, error) {
	dataI, err := wl.Workload4().WorkloadData()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workload data")
	}

	data, ok := dataI.(*zos.ZMachineLight)
	if !ok {
		return nil, errors.Errorf("could not create zmachine workload from data %v", dataI)
	}

	return data, nil
}
