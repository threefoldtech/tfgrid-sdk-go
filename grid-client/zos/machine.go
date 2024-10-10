package zos

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// ZMachine reservation data
type ZMachine struct {
	// Flist of the zmachine, must be a valid url to an flist.
	FList string `json:"flist"`
	// Network configuration for machine network
	Network MachineNetwork `json:"network"`
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

// MachineNetwork structure
type MachineNetwork struct {
	// PublicIP optional public IP attached to this machine. If set
	// it must be a valid name of a PublicIP workload in the same deployment
	PublicIP string `json:"public_ip"`
	// Planetary support planetary network
	Planetary bool `json:"planetary"`

	// Mycelium IP config, if planetary is true, but Mycelium is not set we fall back
	// to yggdrasil support. Otherwise (if mycelium is set) a mycelium ip is used instead.
	Mycelium *MyceliumIP `json:"mycelium,omitempty"`

	// Interfaces list of user znets to join
	Interfaces []MachineInterface `json:"interfaces"`
}

type MyceliumIP struct {
	// Network name (znet name) to join
	Network string
	// Seed is a six bytes random number that is used
	// as a seed to derive a vm mycelium IP.
	//
	// This means that a VM "ip" can be moved to another VM if needed
	// by simply using the same seed.
	// This of course will only work if the network mycelium setup is using
	// the same HexKey
	Seed Bytes `json:"hex_seed"`
}

// MachineInterface structure
type MachineInterface struct {
	// Network name (znet name) to join
	Network string `json:"network"`
	// IP of the zmachine on this network must be a valid Ip in the
	// selected network
	IP net.IP `json:"ip"`
}

// MachineMount structure
type MachineMount struct {
	// Name is name of a zmount. The name must be a valid zmount
	// in the same deployment as the zmachine
	Name string `json:"name"`
	// MountPoint inside the container. Not used if the zmachine
	// is running in a vm mode.
	Mountpoint string `json:"mountpoint"`
}

// MachineCapacity structure
type MachineCapacity struct {
	CPU    uint8  `json:"cpu"`
	Memory uint64 `json:"memory"`
}

type GPU string

func (g GPU) Parts() (slot, vendor, device string, err error) {
	parts := strings.Split(string(g), "/")
	if len(parts) != 3 {
		err = fmt.Errorf("invalid GPU id format '%s'", g)
		return
	}

	return parts[0], parts[1], parts[2], nil
}

// ZMachineResult result returned by VM reservation
type ZMachineResult struct {
	ID          string `json:"id"`
	IP          string `json:"ip"`
	PlanetaryIP string `json:"planetary_ip"`
	MyceliumIP  string `json:"mycelium_ip"`
	ConsoleURL  string `json:"console_url"`
}

func (r *ZMachineResult) UnmarshalJSON(data []byte) error {
	var deprecated struct {
		ID          string `json:"id"`
		IP          string `json:"ip"`
		YggIP       string `json:"ygg_ip"`
		PlanetaryIP string `json:"planetary_ip"`
		MyceliumIP  string `json:"mycelium_ip"`
		ConsoleURL  string `json:"console_url"`
	}

	if err := json.Unmarshal(data, &deprecated); err != nil {
		return err
	}

	r.ID = deprecated.ID
	r.IP = deprecated.IP
	r.PlanetaryIP = deprecated.PlanetaryIP
	if deprecated.YggIP != "" {
		r.PlanetaryIP = deprecated.YggIP
	}
	r.MyceliumIP = deprecated.MyceliumIP
	r.ConsoleURL = deprecated.ConsoleURL

	return nil
}

func (wl *Workload) ZMachineWorkload() (*zos.ZMachine, error) {
	dataI, err := wl.Workload3().WorkloadData()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get workload data")
	}

	data, ok := dataI.(*zos.ZMachine)
	if !ok {
		return nil, errors.Errorf("could not create zmachine workload from data %v", dataI)
	}

	return data, nil
}
