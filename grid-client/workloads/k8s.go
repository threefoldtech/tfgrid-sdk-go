// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"regexp"

	"github.com/pkg/errors"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/subi"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// old: https://hub.grid.tf/tf-official-apps/threefoldtech-k3s-latest.flist
var K8sFlist = "https://hub.grid.tf/tf-official-apps/threefolddev-k3s-v1.31.0.flist"

// K8sNode kubernetes data
type K8sNode struct {
	*VM
	DiskSizeGB uint64 `json:"disk_size"`
}

// K8sCluster struct for k8s cluster
type K8sCluster struct {
	Master      *K8sNode
	Workers     []K8sNode
	Token       string
	NetworkName string

	Flist         string `json:"flist"`
	FlistChecksum string `json:"flist_checksum"`
	Entrypoint    string `json:"entry_point"`

	// optional
	SolutionType string
	SSHKey       string

	// computed
	NodesIPRange     map[uint32]gridtypes.IPNet
	NodeDeploymentID map[uint32]uint64
}

// NewK8sNodeFromWorkload generates a new k8s from a workload
func NewK8sNodeFromWorkload(wl gridtypes.Workload, nodeID uint32, diskSize uint64, computedIP string, computedIP6 string) (K8sNode, error) {
	var k K8sNode
	data, err := wl.WorkloadData()
	if err != nil {
		return k, err
	}
	d := data.(*zos.ZMachine)
	var result zos.ZMachineResult

	if !reflect.DeepEqual(wl.Result, gridtypes.Result{}) {
		err = wl.Result.Unmarshal(&result)
		if err != nil {
			return k, err
		}
	}

	flistCheckSum, err := GetFlistChecksum(d.FList)
	if err != nil {
		return k, err
	}

	var myceliumIPSeed []byte
	if d.Network.Mycelium != nil {
		myceliumIPSeed = d.Network.Mycelium.Seed
	}

	var ip, networkName string
	if len(d.Network.Interfaces) > 0 {
		ip = d.Network.Interfaces[0].IP.String()
		networkName = string(d.Network.Interfaces[0].Network)
	}

	return K8sNode{
		VM: &VM{
			Name:           string(wl.Name),
			NodeID:         nodeID,
			PublicIP:       computedIP != "",
			PublicIP6:      computedIP6 != "",
			Planetary:      result.PlanetaryIP != "",
			Flist:          d.FList,
			FlistChecksum:  flistCheckSum,
			ComputedIP:     computedIP,
			ComputedIP6:    computedIP6,
			PlanetaryIP:    result.PlanetaryIP,
			MyceliumIP:     result.MyceliumIP,
			MyceliumIPSeed: myceliumIPSeed,
			IP:             ip,
			CPU:            d.ComputeCapacity.CPU,
			MemoryMB:       uint64(d.ComputeCapacity.Memory / gridtypes.Megabyte),
			NetworkName:    networkName,
			ConsoleURL:     result.ConsoleURL,
			EnvVars:        d.Env,
		},
		DiskSizeGB: diskSize,
	}, nil
}

// MasterZosWorkload generates a k8s master workload from a k8s node
func (k *K8sNode) MasterZosWorkload(cluster *K8sCluster) (K8sWorkloads []gridtypes.Workload) {
	return k.zosWorkload(cluster, false)
}

// WorkerZosWorkload generates a k8s worker workload from a k8s node
func (k *K8sNode) WorkerZosWorkload(cluster *K8sCluster) (K8sWorkloads []gridtypes.Workload) {
	return k.zosWorkload(cluster, true)
}

// ZosWorkloads generates k8s workloads from a k8s cluster
func (k *K8sCluster) ZosWorkloads() ([]gridtypes.Workload, error) {
	k8sWorkloads := []gridtypes.Workload{}
	k8sWorkloads = append(k8sWorkloads, k.Master.MasterZosWorkload(k)...)

	for _, worker := range k.Workers {
		k8sWorkloads = append(k8sWorkloads, worker.WorkerZosWorkload(k)...)
	}

	return k8sWorkloads, nil
}

// GenerateMetadata generates deployment metadata
func (k *K8sCluster) GenerateMetadata() (string, error) {
	if len(k.SolutionType) == 0 {
		k.SolutionType = fmt.Sprintf("kubernetes/%s", k.Master.Name)
	}

	deploymentData := DeploymentData{
		Version:     int(Version3),
		Name:        k.Master.Name,
		Type:        "kubernetes",
		ProjectName: k.SolutionType,
	}

	deploymentDataBytes, err := json.Marshal(deploymentData)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse deployment data %v", deploymentData)
	}

	return string(deploymentDataBytes), nil
}

func (k *K8sCluster) Validate() error {
	if err := k.Master.Validate(); err != nil {
		return errors.Wrap(err, "master is invalid")
	}

	names := make(map[string]bool)
	names[k.Master.Name] = true

	for _, w := range k.Workers {
		if _, ok := names[w.Name]; ok {
			return errors.Errorf("k8s workers and master must have unique names: %s occurred more than once", w.Name)
		}
		names[w.Name] = true

		if err := w.Validate(); err != nil {
			return errors.Wrap(err, "worker is invalid")
		}
	}

	if err := validateName(k.NetworkName); err != nil {
		return errors.Wrap(err, "master name is invalid")
	}

	if err := k.ValidateToken(); err != nil {
		return err
	}

	if len(k.NodesIPRange) != 0 {
		if err := k.ValidateIPranges(); err != nil {
			return err
		}
	}

	return nil
}

// ValidateToken validate cluster token
func (k *K8sCluster) ValidateToken() error {
	if len(k.Token) < 6 {
		return errors.New("token must be at least 6 characters")
	}
	if len(k.Token) > 15 {
		return errors.New("token must be at most 15 characters")
	}

	isAlphanumeric := regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString(k.Token)
	if !isAlphanumeric {
		return errors.New("token should be alphanumeric")
	}

	return nil
}

// ValidateIPranges validates NodesIPRange of master && workers of k8s cluster
func (k *K8sCluster) ValidateIPranges() error {
	if _, ok := k.NodesIPRange[k.Master.NodeID]; !ok {
		return errors.Errorf("the master node %d does not exist in the network's ip ranges", k.Master.NodeID)
	}

	for _, w := range k.Workers {
		if _, ok := k.NodesIPRange[w.NodeID]; !ok {
			return errors.Errorf("the node with id %d in worker %s does not exist in the network's ip ranges", w.NodeID, w.Name)
		}
	}
	return nil
}

// InvalidateBrokenAttributes removes outdated attrs and deleted contracts
func (k *K8sCluster) InvalidateBrokenAttributes(sub subi.SubstrateExt) error {
	if len(k.NodeDeploymentID) == 0 {
		return nil
	}

	validNodes := make(map[uint32]struct{})
	for node, contractID := range k.NodeDeploymentID {
		contract, err := sub.GetContract(contractID)
		if (err == nil && !contract.State.IsCreated) || errors.Is(err, substrate.ErrNotFound) {
			delete(k.NodeDeploymentID, node)
			delete(k.NodesIPRange, node)
		} else if err != nil {
			return errors.Wrapf(err, "could not get node %d contract %d", node, contractID)
		} else {
			validNodes[node] = struct{}{}
		}

	}
	if _, ok := validNodes[k.Master.NodeID]; !ok {
		k.Master = &K8sNode{}
	}
	return nil
}

func (k *K8sNode) zosWorkload(cluster *K8sCluster, isWorker bool) (K8sWorkloads []gridtypes.Workload) {
	diskName := fmt.Sprintf("%sdisk", k.Name)
	diskWorkload := gridtypes.Workload{
		Name:        gridtypes.Name(diskName),
		Version:     0,
		Type:        zos.ZMountType,
		Description: "",
		Data: gridtypes.MustMarshal(zos.ZMount{
			Size: gridtypes.Unit(k.DiskSizeGB) * gridtypes.Gigabyte,
		}),
	}

	K8sWorkloads = append(K8sWorkloads, diskWorkload)
	publicIPName := ""
	if k.PublicIP || k.PublicIP6 {
		publicIPName = fmt.Sprintf("%sip", k.Name)
		K8sWorkloads = append(K8sWorkloads, ConstructK8sPublicIPWorkload(publicIPName, k.PublicIP, k.PublicIP6))
	}
	envVars := map[string]string{
		"SSH_KEY":           cluster.SSHKey,
		"K3S_TOKEN":         cluster.Token,
		"K3S_DATA_DIR":      "/mydisk",
		"K3S_FLANNEL_IFACE": "eth0",
		"K3S_NODE_NAME":     k.Name,
		"K3S_URL":           "",
	}
	if isWorker {
		// K3S_URL marks where to find the master node
		envVars["K3S_URL"] = fmt.Sprintf("https://%s:6443", cluster.Master.IP)
	}
	var myceliumIP *zos.MyceliumIP
	if len(k.MyceliumIPSeed) != 0 {
		myceliumIP = &zos.MyceliumIP{
			Network: gridtypes.Name(cluster.NetworkName),
			Seed:    k.MyceliumIPSeed,
		}
	}
	workload := gridtypes.Workload{
		Version: 0,
		Name:    gridtypes.Name(k.Name),
		Type:    zos.ZMachineType,
		Data: gridtypes.MustMarshal(zos.ZMachine{
			FList: cluster.Flist,
			Network: zos.MachineNetwork{
				Interfaces: []zos.MachineInterface{
					{
						Network: gridtypes.Name(cluster.NetworkName),
						IP:      net.ParseIP(k.IP),
					},
				},
				PublicIP:  gridtypes.Name(publicIPName),
				Planetary: k.Planetary,
				Mycelium:  myceliumIP,
			},
			ComputeCapacity: zos.MachineCapacity{
				CPU:    k.CPU,
				Memory: gridtypes.Unit(uint(k.MemoryMB)) * gridtypes.Megabyte,
			},
			Entrypoint: cluster.Entrypoint,
			Mounts: []zos.MachineMount{
				{Name: gridtypes.Name(diskName), Mountpoint: "/mydisk"},
			},
			Env: envVars,
		}),
	}
	K8sWorkloads = append(K8sWorkloads, workload)

	return K8sWorkloads
}

// ConstructPublicIPWorkload constructs a public IP workload
func ConstructK8sPublicIPWorkload(workloadName string, ipv4 bool, ipv6 bool) gridtypes.Workload {
	return gridtypes.Workload{
		Version: 0,
		Name:    gridtypes.Name(workloadName),
		Type:    zos.PublicIPType,
		Data: gridtypes.MustMarshal(zos.PublicIP{
			V4: ipv4,
			V6: ipv6,
		}),
	}
}
