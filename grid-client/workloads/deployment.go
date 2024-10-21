// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"sort"

	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
)

var nameMatch = regexp.MustCompile("^[a-zA-Z0-9_]+$")

// Deployment struct
type Deployment struct {
	Name             string
	NodeID           uint32
	SolutionType     string
	SolutionProvider *uint64
	// TODO: remove
	NetworkName string

	Disks    []Disk
	Zdbs     []ZDB
	Vms      []VM
	VmsLight []VMLight
	QSFS     []QSFS
	Volumes  []Volume

	// computed
	NodeDeploymentID map[uint32]uint64
	ContractID       uint64
	IPrange          string
}

// TODO: NewDeployment should take a list of Workload interface instead of defining
// each type as an argument. That way it is cleaner and also allow for networks to
// be created with VMs in the same deployment.

// NewDeployment generates a new deployment
func NewDeployment(name string, nodeID uint32,
	solutionType string, solutionProvider *uint64,
	NetworkName string,
	disks []Disk,
	zdbs []ZDB,
	vms []VM,
	vmsLight []VMLight,
	QSFS []QSFS,
	volumes []Volume,
) Deployment {
	return Deployment{
		Name:             name,
		NodeID:           nodeID,
		SolutionType:     solutionType,
		SolutionProvider: solutionProvider,
		NetworkName:      NetworkName,
		Disks:            disks,
		Zdbs:             zdbs,
		Vms:              vms,
		VmsLight:         vmsLight,
		QSFS:             QSFS,
		Volumes:          volumes,
	}
}

// Validate validates a deployment
func (d *Deployment) Validate() error {
	if err := validateName(d.Name); err != nil {
		return errors.Wrap(err, "deployment name is invalid")
	}

	if err := validateName(d.NetworkName); len(d.Vms) != 0 && err != nil {
		return errors.Wrap(err, "you passed vm/vms in the deployment but network name is invalid")
	}

	if d.NodeID == 0 {
		return fmt.Errorf("node ID should be a positive integer not zero")
	}

	if len(d.Vms) > 0 && len(d.VmsLight) > 0 {
		return fmt.Errorf("cannot deploy vms with vm-light on the same node")
	}

	for _, vm := range d.Vms {
		if err := vm.Validate(); err != nil {
			return errors.Wrapf(err, "vm '%s' is invalid", vm.Name)
		}
	}

	for _, vm := range d.VmsLight {
		if err := vm.Validate(); err != nil {
			return errors.Wrapf(err, "vm-light '%s' is invalid", vm.Name)
		}
	}

	for _, zdb := range d.Zdbs {
		if err := zdb.Validate(); err != nil {
			return errors.Wrapf(err, "zdb '%s' is invalid", zdb.Name)
		}
	}

	for _, qsfs := range d.QSFS {
		if err := qsfs.Validate(); err != nil {
			return errors.Wrapf(err, "qsfs '%s' is invalid", qsfs.Name)
		}
	}

	for _, disk := range d.Disks {
		if err := disk.Validate(); err != nil {
			return errors.Wrapf(err, "disk '%s' is invalid", disk.Name)
		}
	}

	for _, volume := range d.Volumes {
		if err := volume.Validate(); err != nil {
			return errors.Wrapf(err, "volume '%s' is invalid", volume.Name)
		}
	}

	return nil
}

func validateName(name string) error {
	if len(name) == 0 {
		return fmt.Errorf("name cannot be empty")
	}

	// this because max virtio fs tag length is 36 and it is used by cloud-hypervisor
	if len(name) > 36 {
		return fmt.Errorf("name cannot exceed 36 characters")
	}

	if !nameMatch.MatchString(name) {
		return fmt.Errorf("unsupported character in workload name")
	}

	return nil
}

// GenerateMetadata generates deployment metadata
func (d *Deployment) GenerateMetadata() (string, error) {
	if len(d.SolutionType) == 0 {
		d.SolutionType = fmt.Sprintf("vm/%s", d.Name)
	}

	typ := "vm"
	if len(d.VmsLight) > 0 {
		typ = "vm-light"
	}

	deploymentData := DeploymentData{
		Version:     int(Version3),
		Name:        d.Name,
		Type:        typ,
		ProjectName: d.SolutionType,
	}

	deploymentDataBytes, err := json.Marshal(deploymentData)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse deployment data %v", deploymentData)
	}

	return string(deploymentDataBytes), nil
}

// Nullify resets deployment
func (d *Deployment) Nullify() {
	d.Vms = nil
	d.VmsLight = nil
	d.QSFS = nil
	d.Disks = nil
	d.Zdbs = nil
	d.Volumes = nil
	d.ContractID = 0
}

// Match objects to match the input
func (d *Deployment) Match(disks []Disk, QSFS []QSFS, zdbs []ZDB, vms []VM, vmsLight []VMLight, volumes []Volume) {
	vmMap := make(map[string]*VM)
	vmLightMap := make(map[string]*VMLight)

	l := len(d.Disks) + len(d.QSFS) + len(d.Zdbs) + len(d.Vms) + len(d.VmsLight) + len(d.Volumes)
	names := make(map[string]int)
	for idx, o := range d.Disks {
		names[o.Name] = idx - l
	}
	for idx, o := range d.Volumes {
		names[o.Name] = idx - l
	}
	for idx, o := range d.QSFS {
		names[o.Name] = idx - l
	}
	for idx, o := range d.Zdbs {
		names[o.Name] = idx - l
	}
	for idx, o := range d.Vms {
		names[o.Name] = idx - l
		vmMap[o.Name] = &d.Vms[idx]
	}
	for idx, o := range d.VmsLight {
		names[o.Name] = idx - l
		vmLightMap[o.Name] = &d.VmsLight[idx]
	}
	sort.Slice(disks, func(i, j int) bool {
		return names[disks[i].Name] < names[disks[j].Name]
	})
	sort.Slice(volumes, func(i, j int) bool {
		return names[volumes[i].Name] < names[volumes[j].Name]
	})
	sort.Slice(QSFS, func(i, j int) bool {
		return names[QSFS[i].Name] < names[QSFS[j].Name]
	})
	sort.Slice(zdbs, func(i, j int) bool {
		return names[zdbs[i].Name] < names[zdbs[j].Name]
	})
	sort.Slice(vms, func(i, j int) bool {
		return names[vms[i].Name] < names[vms[j].Name]
	})
	sort.Slice(vmsLight, func(i, j int) bool {
		return names[vmsLight[i].Name] < names[vmsLight[j].Name]
	})
	for idx := range vms {
		vm, ok := vmMap[vms[idx].Name]
		if ok {
			vms[idx].LoadFromVM(vm)
		}
	}
	for idx := range vmsLight {
		vm, ok := vmLightMap[vmsLight[idx].Name]
		if ok {
			vmsLight[idx].LoadFromVM(vm)
		}
	}
}

// ZosDeployment generates a new zos deployment from a deployment
func (d *Deployment) ZosDeployment(twin uint32) (zos.Deployment, error) {
	wls := []zos.Workload{}

	for _, d := range d.Disks {
		wls = append(wls, d.ZosWorkload())
	}

	for _, z := range d.Zdbs {
		wls = append(wls, z.ZosWorkload())
	}

	for _, v := range d.Vms {
		vmWls := v.ZosWorkload()
		wls = append(wls, vmWls...)
	}

	for _, v := range d.VmsLight {
		vmLightWls := v.ZosWorkload()
		wls = append(wls, vmLightWls...)
	}

	for _, q := range d.QSFS {
		qWls, err := q.ZosWorkload()
		if err != nil {
			return zos.Deployment{}, err
		}
		wls = append(wls, qWls)
	}
	for _, v := range d.Volumes {
		wls = append(wls, v.ZosWorkload())
	}

	return NewGridDeployment(twin, d.ContractID, wls), nil
}

// NewGridDeployment generates a new grid deployment
func NewGridDeployment(twin uint32, contractID uint64, workloads []zos.Workload) zos.Deployment {
	return zos.Deployment{
		Version:    0,
		TwinID:     twin, // LocalTwin,
		ContractID: contractID,
		Workloads:  workloads,
		SignatureRequirement: zos.SignatureRequirement{
			WeightRequired: 1,
			Requests: []zos.SignatureRequest{
				{
					TwinID: twin,
					Weight: 1,
				},
			},
		},
	}
}

// GetUsedIPs returns used IPs for a deployment
func GetUsedIPs(dl zos.Deployment, nodeID uint32) ([]byte, error) {
	usedIPs := []byte{}
	for _, w := range dl.Workloads {
		if !w.Result.State.IsOkay() {
			return usedIPs, errors.Errorf("workload %s state failed", w.Name)
		}
		if w.Type == zos.ZMachineType {
			vm, err := NewVMFromWorkload(&w, &dl, nodeID)
			if err != nil {
				return usedIPs, errors.Wrapf(err, "error parsing vm: %s", vm.Name)
			}

			ip := net.ParseIP(vm.IP).To4()
			usedIPs = append(usedIPs, ip[3])
		}
	}
	return usedIPs, nil
}

// ParseDeploymentData parses the deployment metadata
func ParseDeploymentData(deploymentMetaData string) (DeploymentData, error) {
	var deploymentData DeploymentData
	err := json.Unmarshal([]byte(deploymentMetaData), &deploymentData)
	if err != nil {
		return DeploymentData{}, err
	}

	return deploymentData, nil
}

// NewDeploymentFromZosDeployment generates deployment from zos deployment
func NewDeploymentFromZosDeployment(d zos.Deployment, nodeID uint32) (Deployment, error) {
	deploymentData, err := ParseDeploymentData(d.Metadata)
	if err != nil {
		return Deployment{}, errors.Wrap(err, "failed to parse deployment data")
	}

	vms := make([]VM, 0)
	vmsLight := make([]VMLight, 0)
	disks := make([]Disk, 0)
	qs := make([]QSFS, 0)
	zdbs := make([]ZDB, 0)
	volumes := make([]Volume, 0)

	var networkName string
	for _, workload := range d.Workloads {
		switch workload.Type {
		case zos.ZMachineType:
			vm, err := NewVMFromWorkload(&workload, &d, nodeID)
			if err != nil {
				return Deployment{}, errors.Wrap(err, "failed to get vm workload")
			}
			vms = append(vms, vm)
			networkName = vm.NetworkName
		case zos.ZMachineLightType:
			vmLight, err := NewVMLightFromWorkload(&workload, &d, nodeID)
			if err != nil {
				return Deployment{}, errors.Wrap(err, "failed to get vm-light workload")
			}
			vmsLight = append(vmsLight, vmLight)
			networkName = vmLight.NetworkName
		case zos.ZDBType:
			zdb, err := NewZDBFromWorkload(&workload)
			if err != nil {
				return Deployment{}, errors.Wrap(err, "failed to get zdb workload")
			}
			zdbs = append(zdbs, zdb)
		case zos.QuantumSafeFSType:
			q, err := NewQSFSFromWorkload(&workload)
			if err != nil {
				return Deployment{}, errors.Wrap(err, "failed to get qsfs workload")
			}
			qs = append(qs, q)
		case zos.ZMountType:
			disk, err := NewDiskFromWorkload(&workload)
			if err != nil {
				return Deployment{}, errors.Wrap(err, "failed to get disk workload")
			}
			disks = append(disks, disk)
		case zos.VolumeType:

			volume, err := NewVolumeFromWorkload(&workload)
			if err != nil {
				return Deployment{}, errors.Wrap(err, "failed to get volume workload")
			}
			volumes = append(volumes, volume)
		}
	}

	return Deployment{
		Name:             deploymentData.Name,
		SolutionType:     deploymentData.ProjectName,
		NetworkName:      networkName,
		Vms:              vms,
		VmsLight:         vmsLight,
		Disks:            disks,
		QSFS:             qs,
		Zdbs:             zdbs,
		Volumes:          volumes,
		NodeID:           nodeID,
		NodeDeploymentID: map[uint32]uint64{nodeID: d.ContractID},
		ContractID:       d.ContractID,
	}, nil
}
