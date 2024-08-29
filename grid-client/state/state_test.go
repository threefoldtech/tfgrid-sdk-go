// Package state for grid state
package state

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/mocks"
	client "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const deploymentName = "testName"
const invalid = "invalid"

func SetupLoaderTests(t *testing.T, wls []gridtypes.Workload) *State {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cl := mocks.NewRMBMockClient(ctrl)
	sub := mocks.NewMockSubstrateExt(ctrl)
	ncPool := mocks.NewMockNodeClientGetter(ctrl)

	state := NewState(ncPool, sub)
	state.CurrentNodeDeployments = map[uint32]ContractIDs{1: []uint64{10}}

	dl1 := workloads.NewGridDeployment(13, wls)
	dl1.ContractID = 10

	ncPool.EXPECT().
		GetNodeClient(sub, uint32(1)).
		Return(client.NewNodeClient(13, cl, 10), nil).AnyTimes()

	cl.EXPECT().
		Call(gomock.Any(), uint32(13), "zos.deployment.get", gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, twin uint32, fn string, data, result interface{}) error {
			var res *gridtypes.Deployment = result.(*gridtypes.Deployment)
			dl1.Metadata = "{\"type\":\"\",\"name\":\"testName\",\"projectName\":\"\"}"
			*res = dl1
			return nil
		}).AnyTimes()

	sub.EXPECT().
		GetContractIDByNameRegistration("test").
		Return(uint64(11), nil).AnyTimes()

	return state
}

func TestLoadDiskFromGrid(t *testing.T) {
	disk := workloads.Disk{
		Name:        "test",
		SizeGB:      100,
		Description: "test des",
	}

	diskWl := gridtypes.Workload{
		Name:        gridtypes.Name("test"),
		Version:     0,
		Type:        zos.ZMountType,
		Description: "test des",
		Data: gridtypes.MustMarshal(zos.ZMount{
			Size: 100 * gridtypes.Gigabyte,
		}),
	}

	t.Run("success", func(t *testing.T) {
		state := SetupLoaderTests(t, []gridtypes.Workload{diskWl})

		got, err := state.LoadDiskFromGrid(context.Background(), 1, "test", deploymentName)
		assert.NoError(t, err)
		assert.Equal(t, disk, got)
	})

	t.Run("invalid type", func(t *testing.T) {
		diskWlCp := diskWl
		diskWlCp.Type = invalid

		state := SetupLoaderTests(t, []gridtypes.Workload{diskWlCp})

		_, err := state.LoadDiskFromGrid(context.Background(), 1, "test", deploymentName)
		assert.Error(t, err)
	})

	t.Run("wrong workload data", func(t *testing.T) {
		diskWlCp := diskWl
		diskWlCp.Type = zos.GatewayNameProxyType
		diskWlCp.Data = gridtypes.MustMarshal(zos.GatewayNameProxy{
			Name: "name",
		})

		state := SetupLoaderTests(t, []gridtypes.Workload{diskWlCp})

		_, err := state.LoadDiskFromGrid(context.Background(), 1, "test", deploymentName)
		assert.Error(t, err)
	})
}

func TestLoadGatewayFQDNFromGrid(t *testing.T) {
	gatewayWl := gridtypes.Workload{
		Version: 0,
		Type:    zos.GatewayFQDNProxyType,
		Name:    gridtypes.Name("test"),
		Data: gridtypes.MustMarshal(zos.GatewayFQDNProxy{
			GatewayBase: zos.GatewayBase{
				TLSPassthrough: true,
				Backends:       []zos.Backend{"http://1.1.1.1"},
			},
			FQDN: "test",
		}),
	}
	gateway := workloads.GatewayFQDNProxy{
		Name:             "test",
		TLSPassthrough:   true,
		Backends:         []zos.Backend{"http://1.1.1.1"},
		FQDN:             "test",
		NodeID:           1,
		ContractID:       10,
		NodeDeploymentID: map[uint32]uint64{1: 10},
	}

	t.Run("success", func(t *testing.T) {
		state := SetupLoaderTests(t, []gridtypes.Workload{gatewayWl})

		got, err := state.LoadGatewayFQDNFromGrid(context.Background(), 1, "test", deploymentName)
		assert.NoError(t, err)
		assert.Equal(t, gateway, got)
	})

	t.Run("invalid type", func(t *testing.T) {
		gatewayWlCp := gatewayWl
		gatewayWlCp.Type = invalid

		state := SetupLoaderTests(t, []gridtypes.Workload{gatewayWlCp})

		_, err := state.LoadGatewayFQDNFromGrid(context.Background(), 1, "test", deploymentName)
		assert.Error(t, err)
	})

	t.Run("wrong workload data", func(t *testing.T) {
		gatewayWlCp := gatewayWl
		gatewayWlCp.Type = zos.GatewayNameProxyType
		gatewayWlCp.Data = gridtypes.MustMarshal(zos.GatewayNameProxy{
			Name: "name",
		})

		state := SetupLoaderTests(t, []gridtypes.Workload{gatewayWlCp})

		_, err := state.LoadGatewayFQDNFromGrid(context.Background(), 1, "test", deploymentName)
		assert.Error(t, err)
	})
}

func TestLoadGatewayNameFromGrid(t *testing.T) {
	res, err := json.Marshal(zos.GatewayNameProxy{
		Name: "test",
	})
	assert.NoError(t, err)

	gatewayWl := gridtypes.Workload{
		Version: 0,
		Type:    zos.GatewayNameProxyType,
		Name:    gridtypes.Name(deploymentName),
		Data: gridtypes.MustMarshal(zos.GatewayNameProxy{
			GatewayBase: zos.GatewayBase{
				TLSPassthrough: true,
				Backends:       []zos.Backend{"http://1.1.1.1"},
			},
			Name: "test",
		}),
		Result: gridtypes.Result{
			Created: 1000,
			State:   gridtypes.StateOk,
			Data:    res,
		},
	}
	gateway := workloads.GatewayNameProxy{
		Name:             "test",
		TLSPassthrough:   true,
		Backends:         []zos.Backend{"http://1.1.1.1"},
		NameContractID:   11,
		NodeID:           1,
		ContractID:       10,
		NodeDeploymentID: map[uint32]uint64{1: 10},
	}

	t.Run("success", func(t *testing.T) {
		state := SetupLoaderTests(t, []gridtypes.Workload{gatewayWl})

		got, err := state.LoadGatewayNameFromGrid(context.Background(), 1, "test", deploymentName)
		assert.NoError(t, err)
		assert.Equal(t, gateway, got)
	})
	t.Run("invalid type", func(t *testing.T) {
		gatewayWlCp := gatewayWl
		gatewayWlCp.Type = invalid

		state := SetupLoaderTests(t, []gridtypes.Workload{gatewayWlCp})

		_, err := state.LoadGatewayNameFromGrid(context.Background(), 1, "test", deploymentName)
		assert.Error(t, err)
	})

	t.Run("wrong workload data", func(t *testing.T) {
		gatewayWlCp := gatewayWl
		gatewayWlCp.Type = zos.GatewayFQDNProxyType
		gatewayWlCp.Data = gridtypes.MustMarshal(zos.GatewayFQDNProxy{
			FQDN: "123",
		})

		state := SetupLoaderTests(t, []gridtypes.Workload{gatewayWlCp})

		_, err := state.LoadGatewayNameFromGrid(context.Background(), 1, "test", deploymentName)
		assert.Error(t, err)
	})
}

func TestLoadK8sFromGrid(t *testing.T) {
	flist := "https://hub.grid.tf/tf-official-apps/base:latest.flist"
	flistCheckSum, err := workloads.GetFlistChecksum(flist)
	assert.NoError(t, err)

	res, _ := json.Marshal(zos.ZMachineResult{
		IP:          "1.1.1.1",
		PlanetaryIP: "203:8b0b:5f3e:b859:c36:efdf:ab6e:50cc",
	})

	master := workloads.K8sNode{
		VM: &workloads.VM{
			Name:          "test",
			NodeID:        1,
			Flist:         flist,
			FlistChecksum: flistCheckSum,
			PublicIP:      false,
			Planetary:     true,
			CPU:           1,
			MemoryMB:      8,
			PlanetaryIP:   "203:8b0b:5f3e:b859:c36:efdf:ab6e:50cc",
			IP:            "1.1.1.1",
			NetworkName:   "test",
		},
	}

	var Workers []workloads.K8sNode

	ipRange, err := gridtypes.ParseIPNet("1.1.1.1/24")
	assert.NoError(t, err)

	cluster := workloads.K8sCluster{
		Master:           &master,
		Workers:          Workers,
		Token:            "",
		SSHKey:           "",
		NetworkName:      "test",
		NodeDeploymentID: map[uint32]uint64{1: 10},
		NodesIPRange: map[uint32]gridtypes.IPNet{
			1: ipRange,
		},
	}

	k8sWorkload := gridtypes.Workload{
		Version: 0,
		Name:    gridtypes.Name("test"),
		Type:    zos.ZMachineType,
		Data: gridtypes.MustMarshal(zos.ZMachine{
			FList: flist,
			Network: zos.MachineNetwork{
				Interfaces: []zos.MachineInterface{
					{
						Network: gridtypes.Name("test"),
						IP:      net.ParseIP("1.1.1.1"),
					},
				},
				Planetary: true,
			},
			Size: 100,
			ComputeCapacity: zos.MachineCapacity{
				CPU:    1,
				Memory: 8 * gridtypes.Megabyte,
			},
			Mounts:     []zos.MachineMount{},
			Entrypoint: "",
			Env:        map[string]string{},
			Corex:      false,
		}),
		Result: gridtypes.Result{
			Created: 5000,
			State:   gridtypes.StateOk,
			Data:    res,
		},
	}

	metadata, err := json.Marshal(workloads.NetworkMetaData{
		Version: workloads.Version,
		UserAccesses: []workloads.UserAccess{
			{
				Subnet:     "",
				PrivateKey: "",
				NodeID:     0,
			},
		},
	})
	assert.NoError(t, err)

	networkWl := gridtypes.Workload{
		Version: 0,
		Name:    gridtypes.Name("test"),
		Type:    zos.NetworkType,
		Data: gridtypes.MustMarshal(zos.Network{
			NetworkIPRange: gridtypes.MustParseIPNet(ipRange.String()),
			Subnet:         ipRange,
			WGPrivateKey:   "",
			WGListenPort:   0,
			Peers:          []zos.Peer{},
		}),
		Metadata:    string(metadata),
		Description: "test description",
		Result:      gridtypes.Result{},
	}

	t.Run("success", func(t *testing.T) {
		state := SetupLoaderTests(t, []gridtypes.Workload{networkWl, k8sWorkload})

		got, err := state.LoadK8sFromGrid(context.Background(), []uint32{1}, deploymentName)
		assert.NoError(t, err)
		assert.Equal(t, cluster, got)
	})

	t.Run("invalid type", func(t *testing.T) {
		k8sWorkloadCp := k8sWorkload
		k8sWorkloadCp.Type = invalid

		state := SetupLoaderTests(t, []gridtypes.Workload{k8sWorkloadCp})

		_, err := state.LoadK8sFromGrid(context.Background(), []uint32{1}, deploymentName)
		assert.Error(t, err)
	})

	t.Run("wrong workload data", func(t *testing.T) {
		k8sWorkloadCp := k8sWorkload
		k8sWorkloadCp.Type = zos.ZMachineType
		k8sWorkloadCp.Data = gridtypes.MustMarshal(zos.ZMachine{
			FList: "",
		})

		state := SetupLoaderTests(t, []gridtypes.Workload{k8sWorkloadCp})

		_, err := state.LoadK8sFromGrid(context.Background(), []uint32{1}, deploymentName)
		assert.Error(t, err)
	})
}

func TestLoadNetworkFromGrid(t *testing.T) {
	ipRange, err := gridtypes.ParseIPNet("1.1.1.1/24")
	assert.NoError(t, err)

	znet := workloads.ZNet{
		Name:             "test",
		Description:      "test description",
		Nodes:            []uint32{1},
		IPRange:          ipRange,
		AddWGAccess:      false,
		NodeDeploymentID: map[uint32]uint64{1: 10},
		WGPort:           map[uint32]int{},
		Keys:             map[uint32]wgtypes.Key{},
		NodesIPRange:     map[uint32]gridtypes.IPNet{1: ipRange},
		MyceliumKeys:     make(map[uint32][]byte),
	}

	metadata, err := json.Marshal(workloads.NetworkMetaData{
		Version: workloads.Version,
		UserAccesses: []workloads.UserAccess{
			{
				Subnet:     "",
				PrivateKey: "",
				NodeID:     0,
			},
		},
	})
	assert.NoError(t, err)

	networkWl := gridtypes.Workload{
		Version: 0,
		Name:    gridtypes.Name("test"),
		Type:    zos.NetworkType,
		Data: gridtypes.MustMarshal(zos.Network{
			NetworkIPRange: gridtypes.MustParseIPNet(znet.IPRange.String()),
			Subnet:         ipRange,
			WGPrivateKey:   "",
			WGListenPort:   0,
			Peers:          []zos.Peer{},
		}),
		Metadata:    string(metadata),
		Description: "test description",
		Result:      gridtypes.Result{},
	}

	t.Run("success", func(t *testing.T) {
		state := SetupLoaderTests(t, []gridtypes.Workload{networkWl})

		got, err := state.LoadNetworkFromGrid(context.Background(), "test")
		assert.NoError(t, err)
		assert.Equal(t, znet, got)
	})

	t.Run("invalid type", func(t *testing.T) {
		networkWlCp := networkWl
		networkWlCp.Type = invalid

		state := SetupLoaderTests(t, []gridtypes.Workload{networkWlCp})

		_, err := state.LoadNetworkFromGrid(context.Background(), "test")
		assert.Error(t, err)
	})

	t.Run("wrong workload data", func(t *testing.T) {
		networkWlCp := networkWl
		networkWlCp.Type = zos.GatewayNameProxyType
		networkWlCp.Data = gridtypes.MustMarshal(zos.Network{
			WGPrivateKey: "key",
		})

		state := SetupLoaderTests(t, []gridtypes.Workload{networkWlCp})

		_, err := state.LoadNetworkFromGrid(context.Background(), "test")
		assert.Error(t, err)
	})
}

func TestLoadQSFSFromGrid(t *testing.T) {
	res, err := json.Marshal(zos.QuatumSafeFSResult{
		Path:            "path",
		MetricsEndpoint: "endpoint",
	})
	assert.NoError(t, err)

	k, err := hex.DecodeString("4d778ba3216e4da4231540c92a55f06157cabba802f9b68fb0f78375d2e825af")
	assert.NoError(t, err)

	qsfsWl := gridtypes.Workload{
		Version:     0,
		Name:        gridtypes.Name("test"),
		Type:        zos.QuantumSafeFSType,
		Description: "test des",
		Data: gridtypes.MustMarshal(zos.QuantumSafeFS{
			Cache: 2048 * gridtypes.Megabyte,
			Config: zos.QuantumSafeFSConfig{
				MinimalShards:     10,
				ExpectedShards:    20,
				RedundantGroups:   2,
				RedundantNodes:    5,
				MaxZDBDataDirSize: 10,
				Encryption: zos.Encryption{
					Algorithm: zos.EncryptionAlgorithm("AES"),
					Key:       zos.EncryptionKey(k),
				},
				Meta: zos.QuantumSafeMeta{
					Type: "zdb",
					Config: zos.QuantumSafeConfig{
						Prefix: "test",
						Encryption: zos.Encryption{
							Algorithm: zos.EncryptionAlgorithm("AES"),
							Key:       zos.EncryptionKey(k),
						},
						Backends: []zos.ZdbBackend{
							{Address: "1.1.1.1", Namespace: "test ns", Password: "password"},
						},
					},
				},
				Groups: []zos.ZdbGroup{{Backends: []zos.ZdbBackend{
					{Address: "2.2.2.2", Namespace: "test ns2", Password: "password2"},
				}}},
				Compression: zos.QuantumCompression{
					Algorithm: "snappy",
				},
			},
		}),
		Result: gridtypes.Result{
			Created: 10000,
			State:   gridtypes.StateOk,
			Data:    res,
		},
	}

	qsfs := workloads.QSFS{
		Name:                 "test",
		Description:          "test des",
		Cache:                2048,
		MinimalShards:        10,
		ExpectedShards:       20,
		RedundantGroups:      2,
		RedundantNodes:       5,
		MaxZDBDataDirSize:    10,
		EncryptionAlgorithm:  "AES",
		EncryptionKey:        "4d778ba3216e4da4231540c92a55f06157cabba802f9b68fb0f78375d2e825af",
		CompressionAlgorithm: "snappy",
		Metadata: workloads.Metadata{
			Type:                "zdb",
			Prefix:              "test",
			EncryptionAlgorithm: "AES",
			EncryptionKey:       "4d778ba3216e4da4231540c92a55f06157cabba802f9b68fb0f78375d2e825af",
			Backends: workloads.Backends{
				{Address: "1.1.1.1", Namespace: "test ns", Password: "password"},
			},
		},
		Groups: workloads.Groups{{Backends: workloads.Backends{
			{Address: "2.2.2.2", Namespace: "test ns2", Password: "password2"},
		}}},
		MetricsEndpoint: "endpoint",
	}

	t.Run("success", func(t *testing.T) {
		state := SetupLoaderTests(t, []gridtypes.Workload{qsfsWl})

		got, err := state.LoadQSFSFromGrid(context.Background(), 1, "test", deploymentName)
		assert.NoError(t, err)
		assert.Equal(t, qsfs, got)
	})
	t.Run("invalid type", func(t *testing.T) {
		qsfsWlCp := qsfsWl
		qsfsWlCp.Type = invalid

		state := SetupLoaderTests(t, []gridtypes.Workload{qsfsWlCp})

		_, err := state.LoadQSFSFromGrid(context.Background(), 1, "test", deploymentName)
		assert.Error(t, err)
	})

	t.Run("wrong workload data", func(t *testing.T) {
		qsfsWlCp := qsfsWl
		qsfsWlCp.Type = zos.GatewayNameProxyType
		qsfsWlCp.Data = gridtypes.MustMarshal(zos.GatewayNameProxy{
			Name: "name",
		})

		state := SetupLoaderTests(t, []gridtypes.Workload{qsfsWlCp})

		_, err := state.LoadQSFSFromGrid(context.Background(), 1, "test", deploymentName)
		assert.Error(t, err)
	})

	t.Run("invalid result data", func(t *testing.T) {
		qsfsWlCp := qsfsWl
		qsfsWlCp.Result.Data = nil

		state := SetupLoaderTests(t, []gridtypes.Workload{qsfsWlCp})

		_, err := state.LoadQSFSFromGrid(context.Background(), 1, "test", deploymentName)
		assert.Error(t, err)
	})
}

func TestLoadVMFromGrid(t *testing.T) {
	vmRes, err := json.Marshal(zos.ZMachineResult{
		ID:          "5",
		IP:          "5.5.5.5",
		PlanetaryIP: "203:8b0b:5f3e:b859:c36:efdf:ab6e:50cc",
	})
	assert.NoError(t, err)

	var zlogs []workloads.Zlog

	vm := workloads.VM{
		Name:          "test",
		NodeID:        1,
		Flist:         "https://hub.grid.tf/tf-official-apps/base:latest.flist",
		FlistChecksum: "f94b5407f2e8635bd1b6b3dac7fef2d9",
		PublicIP:      false,
		ComputedIP:    "",
		PublicIP6:     false,
		Planetary:     true,
		Corex:         false,
		PlanetaryIP:   "203:8b0b:5f3e:b859:c36:efdf:ab6e:50cc",
		IP:            "1.1.1.1",
		Description:   "test des",
		CPU:           2,
		MemoryMB:      2048,
		RootfsSizeMB:  4096,
		Entrypoint:    "entrypoint",
		Mounts: []workloads.Mount{
			{Name: "disk", MountPoint: "mount"},
		},
		Zlogs:       zlogs,
		EnvVars:     map[string]string{"var1": "val1"},
		NetworkName: "test_network",
	}

	pubWl := gridtypes.Workload{
		Version: 0,
		Name:    gridtypes.Name("testip"),
		Type:    zos.PublicIPType,
		Data: gridtypes.MustMarshal(zos.PublicIP{
			V4: true,
		}),
	}

	vmWl := gridtypes.Workload{
		Version: 0,
		Name:    gridtypes.Name("test"),
		Type:    zos.ZMachineType,
		Data: gridtypes.MustMarshal(zos.ZMachine{
			FList: "https://hub.grid.tf/tf-official-apps/base:latest.flist",
			Network: zos.MachineNetwork{
				Interfaces: []zos.MachineInterface{
					{
						Network: gridtypes.Name("test_network"),
						IP:      net.ParseIP("1.1.1.1"),
					},
				},
				PublicIP:  pubWl.Name,
				Planetary: true,
			},
			ComputeCapacity: zos.MachineCapacity{
				CPU:    uint8(2),
				Memory: 2048 * gridtypes.Megabyte,
			},
			Size:       4096 * gridtypes.Megabyte,
			Entrypoint: "entrypoint",
			Corex:      false,
			Mounts: []zos.MachineMount{
				{Name: gridtypes.Name("disk"), Mountpoint: "mount"},
			},
			Env: map[string]string{"var1": "val1"},
		}),
		Description: "test des",
		Result: gridtypes.Result{
			Created: 5000,
			State:   gridtypes.StateOk,
			Data:    vmRes,
		},
	}

	t.Run("success", func(t *testing.T) {
		state := SetupLoaderTests(t, []gridtypes.Workload{vmWl, pubWl})

		got, err := state.LoadVMFromGrid(context.Background(), vm.NodeID, "test", deploymentName)
		assert.NoError(t, err)
		assert.Equal(t, vm, got)
	})

	t.Run("invalid type", func(t *testing.T) {
		vmWlCp := vmWl
		vmWlCp.Type = invalid

		state := SetupLoaderTests(t, []gridtypes.Workload{vmWlCp})

		_, err := state.LoadVMFromGrid(context.Background(), vm.NodeID, "test", deploymentName)
		assert.Error(t, err)
	})

	t.Run("wrong workload data", func(t *testing.T) {
		vmWlCp := vmWl
		vmWlCp.Type = zos.GatewayFQDNProxyType
		vmWlCp.Data = gridtypes.MustMarshal(zos.GatewayFQDNProxy{
			FQDN: "123",
		})

		state := SetupLoaderTests(t, []gridtypes.Workload{vmWlCp})

		_, err := state.LoadVMFromGrid(context.Background(), vm.NodeID, "test", deploymentName)
		assert.Error(t, err)
	})

	t.Run("invalid result data", func(t *testing.T) {
		vmWlCp := vmWl
		vmWlCp.Result.Data = nil

		state := SetupLoaderTests(t, []gridtypes.Workload{vmWlCp})

		_, err := state.LoadVMFromGrid(context.Background(), vm.NodeID, "test", deploymentName)
		assert.Error(t, err)
	})
}

func TestLoadZdbFromGrid(t *testing.T) {
	res, err := json.Marshal(zos.ZDBResult{
		Namespace: "test name",
		IPs: []string{
			"1.1.1.1",
			"2.2.2.2",
		},
		Port: 5000,
	})
	assert.NoError(t, err)

	zdbWl := gridtypes.Workload{
		Name:        gridtypes.Name("test"),
		Type:        zos.ZDBType,
		Description: "test des",
		Version:     0,
		Result: gridtypes.Result{
			Created: 1000,
			State:   gridtypes.StateOk,
			Data:    res,
		},
		Data: gridtypes.MustMarshal(zos.ZDB{
			Size:     100 * gridtypes.Gigabyte,
			Mode:     zos.ZDBMode("user"),
			Password: "password",
			Public:   true,
		}),
	}
	zdb := workloads.ZDB{
		Name:        "test",
		Password:    "password",
		Public:      true,
		SizeGB:      100,
		Description: "test des",
		Mode:        "user",
		Namespace:   "test name",
		IPs: []string{
			"1.1.1.1",
			"2.2.2.2",
		},
		Port: 5000,
	}

	t.Run("success", func(t *testing.T) {
		state := SetupLoaderTests(t, []gridtypes.Workload{zdbWl})

		got, err := state.LoadZdbFromGrid(context.Background(), 1, "test", deploymentName)
		assert.NoError(t, err)
		assert.Equal(t, zdb, got)
	})

	t.Run("invalid type", func(t *testing.T) {
		zdbWlCp := zdbWl
		zdbWlCp.Type = invalid

		state := SetupLoaderTests(t, []gridtypes.Workload{zdbWlCp})

		_, err := state.LoadZdbFromGrid(context.Background(), 1, "test", deploymentName)
		assert.Error(t, err)
	})

	t.Run("wrong workload data", func(t *testing.T) {
		zdbWlCp := zdbWl
		zdbWlCp.Type = zos.GatewayNameProxyType
		zdbWlCp.Data = gridtypes.MustMarshal(zos.GatewayNameProxy{
			Name: "name",
		})

		state := SetupLoaderTests(t, []gridtypes.Workload{zdbWlCp})

		_, err := state.LoadZdbFromGrid(context.Background(), 1, "test", deploymentName)
		assert.Error(t, err)
	})

	t.Run("invalid result data", func(t *testing.T) {
		zdbWlCp := zdbWl
		zdbWlCp.Result.Data = nil

		state := SetupLoaderTests(t, []gridtypes.Workload{zdbWlCp})

		_, err := state.LoadZdbFromGrid(context.Background(), 1, "test", deploymentName)
		assert.Error(t, err)
	})
}
