// // Package state for grid state
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
	zosTypes "github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

const (
	deploymentName = "testName"
	invalid        = "invalid"
)

func SetupLoaderTests(t *testing.T, wls []zosTypes.Workload) *State {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cl := mocks.NewRMBMockClient(ctrl)
	sub := mocks.NewMockSubstrateExt(ctrl)
	ncPool := mocks.NewMockNodeClientGetter(ctrl)

	state := NewState(ncPool, sub)
	state.CurrentNodeDeployments = map[uint32]ContractIDs{1: []uint64{10}}

	dl1 := workloads.NewGridDeployment(13, 0, wls)
	dl1.ContractID = 10

	ncPool.EXPECT().
		GetNodeClient(sub, uint32(1)).
		Return(client.NewNodeClient(13, cl, 10), nil).AnyTimes()

	cl.EXPECT().
		Call(gomock.Any(), uint32(13), "zos.deployment.get", gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, twin uint32, fn string, data, result interface{}) error {
			var res *zosTypes.Deployment = result.(*zosTypes.Deployment)
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

	diskWl := zosTypes.Workload{
		Name:        "test",
		Version:     0,
		Type:        zosTypes.ZMountType,
		Description: "test des",
		Data: zosTypes.MustMarshal(zosTypes.ZMount{
			Size: 100 * zosTypes.Gigabyte,
		}),
	}

	t.Run("success", func(t *testing.T) {
		state := SetupLoaderTests(t, []zosTypes.Workload{diskWl})

		got, err := state.LoadDiskFromGrid(context.Background(), 1, "test", deploymentName)
		assert.NoError(t, err)
		assert.Equal(t, disk, got)
	})

	t.Run("invalid type", func(t *testing.T) {
		diskWlCp := diskWl
		diskWlCp.Type = invalid

		state := SetupLoaderTests(t, []zosTypes.Workload{diskWlCp})

		_, err := state.LoadDiskFromGrid(context.Background(), 1, "test", deploymentName)
		assert.Error(t, err)
	})

	t.Run("wrong workload data", func(t *testing.T) {
		diskWlCp := diskWl
		diskWlCp.Type = zosTypes.GatewayNameProxyType
		diskWlCp.Data = zosTypes.MustMarshal(zos.GatewayNameProxy{
			Name: "name",
		})

		state := SetupLoaderTests(t, []zosTypes.Workload{diskWlCp})

		_, err := state.LoadDiskFromGrid(context.Background(), 1, "test", deploymentName)
		assert.Error(t, err)
	})
}

func TestLoadGatewayFQDNFromGrid(t *testing.T) {
	gatewayWl := zosTypes.Workload{
		Version: 0,
		Type:    zosTypes.GatewayFQDNProxyType,
		Name:    "test",
		Data: zosTypes.MustMarshal(zos.GatewayFQDNProxy{
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
		state := SetupLoaderTests(t, []zosTypes.Workload{gatewayWl})

		got, err := state.LoadGatewayFQDNFromGrid(context.Background(), 1, "test", deploymentName)
		assert.NoError(t, err)
		assert.Equal(t, gateway, got)
	})

	t.Run("invalid type", func(t *testing.T) {
		gatewayWlCp := gatewayWl
		gatewayWlCp.Type = invalid

		state := SetupLoaderTests(t, []zosTypes.Workload{gatewayWlCp})

		_, err := state.LoadGatewayFQDNFromGrid(context.Background(), 1, "test", deploymentName)
		assert.Error(t, err)
	})

	t.Run("wrong workload data", func(t *testing.T) {
		gatewayWlCp := gatewayWl
		gatewayWlCp.Type = zosTypes.GatewayNameProxyType
		gatewayWlCp.Data = zosTypes.MustMarshal(zos.GatewayNameProxy{
			Name: "name",
		})

		state := SetupLoaderTests(t, []zosTypes.Workload{gatewayWlCp})

		_, err := state.LoadGatewayFQDNFromGrid(context.Background(), 1, "test", deploymentName)
		assert.Error(t, err)
	})
}

func TestLoadGatewayNameFromGrid(t *testing.T) {
	res, err := json.Marshal(zos.GatewayNameProxy{
		Name: "test",
	})
	assert.NoError(t, err)

	gatewayWl := zosTypes.Workload{
		Version: 0,
		Type:    zosTypes.GatewayNameProxyType,
		Name:    deploymentName,
		Data: zosTypes.MustMarshal(zos.GatewayNameProxy{
			GatewayBase: zos.GatewayBase{
				TLSPassthrough: true,
				Backends:       []zos.Backend{"http://1.1.1.1"},
			},
			Name: "test",
		}),
		Result: zosTypes.Result{
			Created: 1000,
			State:   zosTypes.StateOk,
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
		state := SetupLoaderTests(t, []zosTypes.Workload{gatewayWl})

		got, err := state.LoadGatewayNameFromGrid(context.Background(), 1, "test", deploymentName)
		assert.NoError(t, err)
		assert.Equal(t, gateway, got)
	})
	t.Run("invalid type", func(t *testing.T) {
		gatewayWlCp := gatewayWl
		gatewayWlCp.Type = invalid

		state := SetupLoaderTests(t, []zosTypes.Workload{gatewayWlCp})

		_, err := state.LoadGatewayNameFromGrid(context.Background(), 1, "test", deploymentName)
		assert.Error(t, err)
	})

	t.Run("wrong workload data", func(t *testing.T) {
		gatewayWlCp := gatewayWl
		gatewayWlCp.Type = zosTypes.GatewayFQDNProxyType
		gatewayWlCp.Data = zosTypes.MustMarshal(zos.GatewayFQDNProxy{
			FQDN: "123",
		})

		state := SetupLoaderTests(t, []zosTypes.Workload{gatewayWlCp})

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
			EnvVars:       map[string]string{},
		},
	}

	var Workers []workloads.K8sNode

	ipRange, err := zosTypes.ParseIPNet("1.1.1.1/24")
	assert.NoError(t, err)

	cluster := workloads.K8sCluster{
		Master:           &master,
		Workers:          Workers,
		Token:            "",
		SSHKey:           "",
		NetworkName:      "test",
		Flist:            flist,
		FlistChecksum:    flistCheckSum,
		NodeDeploymentID: map[uint32]uint64{1: 10},
		NodesIPRange: map[uint32]gridtypes.IPNet{
			1: gridtypes.IPNet(ipRange),
		},
	}

	k8sWorkload := zosTypes.Workload{
		Version: 0,
		Name:    "test",
		Type:    zosTypes.ZMachineType,
		Data: zosTypes.MustMarshal(zosTypes.ZMachine{
			FList: flist,
			Network: zosTypes.MachineNetwork{
				Interfaces: []zosTypes.MachineInterface{
					{
						Network: "test",
						IP:      net.ParseIP("1.1.1.1"),
					},
				},
				Planetary: true,
			},
			Size: 100,
			ComputeCapacity: zosTypes.MachineCapacity{
				CPU:    1,
				Memory: 8 * zosTypes.Megabyte,
			},
			Mounts:     []zosTypes.MachineMount{},
			Entrypoint: "",
			Env:        map[string]string{},
			Corex:      false,
		}),
		Result: zosTypes.Result{
			Created: 5000,
			State:   zosTypes.StateOk,
			Data:    res,
		},
	}

	metadata, err := json.Marshal(workloads.NetworkMetaData{
		Version: int(workloads.Version3),
		UserAccesses: []workloads.UserAccess{
			{
				Subnet:     "",
				PrivateKey: "",
				NodeID:     0,
			},
		},
	})
	assert.NoError(t, err)

	networkWl := zosTypes.Workload{
		Version: 0,
		Name:    "test",
		Type:    zosTypes.NetworkType,
		Data: zosTypes.MustMarshal(zosTypes.Network{
			NetworkIPRange: zosTypes.MustParseIPNet(ipRange.String()),
			Subnet:         ipRange,
			WGPrivateKey:   "",
			WGListenPort:   0,
			Peers:          []zosTypes.Peer{},
		}),
		Metadata:    string(metadata),
		Description: "test description",
		Result:      zosTypes.Result{},
	}

	t.Run("success", func(t *testing.T) {
		state := SetupLoaderTests(t, []zosTypes.Workload{networkWl, k8sWorkload})

		got, err := state.LoadK8sFromGrid(context.Background(), []uint32{1}, deploymentName)
		assert.NoError(t, err)
		assert.Equal(t, cluster, got)
	})

	t.Run("invalid type", func(t *testing.T) {
		k8sWorkloadCp := k8sWorkload
		k8sWorkloadCp.Type = invalid

		state := SetupLoaderTests(t, []zosTypes.Workload{k8sWorkloadCp})

		_, err := state.LoadK8sFromGrid(context.Background(), []uint32{1}, deploymentName)
		assert.Error(t, err)
	})

	t.Run("wrong workload data", func(t *testing.T) {
		k8sWorkloadCp := k8sWorkload
		k8sWorkloadCp.Type = zosTypes.ZMachineType
		k8sWorkloadCp.Data = zosTypes.MustMarshal(zosTypes.ZMachine{
			FList: "",
		})

		state := SetupLoaderTests(t, []zosTypes.Workload{k8sWorkloadCp})

		_, err := state.LoadK8sFromGrid(context.Background(), []uint32{1}, deploymentName)
		assert.Error(t, err)
	})
}

func TestLoadNetworkFromGrid(t *testing.T) {
	ipRange, err := zosTypes.ParseIPNet("1.1.1.1/24")
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
		NodesIPRange:     map[uint32]zosTypes.IPNet{1: ipRange},
		MyceliumKeys:     make(map[uint32][]byte),
	}

	metadata, err := json.Marshal(workloads.NetworkMetaData{
		Version: int(workloads.Version3),
		UserAccesses: []workloads.UserAccess{
			{
				Subnet:     "",
				PrivateKey: "",
				NodeID:     0,
			},
		},
	})
	assert.NoError(t, err)

	networkWl := zosTypes.Workload{
		Version: 0,
		Name:    "test",
		Type:    zosTypes.NetworkType,
		Data: zosTypes.MustMarshal(zosTypes.Network{
			NetworkIPRange: zosTypes.MustParseIPNet(znet.IPRange.String()),
			Subnet:         ipRange,
			WGPrivateKey:   "",
			WGListenPort:   0,
			Peers:          []zosTypes.Peer{},
		}),
		Metadata:    string(metadata),
		Description: "test description",
		Result:      zosTypes.Result{},
	}

	t.Run("success", func(t *testing.T) {
		state := SetupLoaderTests(t, []zosTypes.Workload{networkWl})

		got, err := state.LoadNetworkFromGrid(context.Background(), "test")
		assert.NoError(t, err)
		assert.Equal(t, znet, got)
	})

	t.Run("invalid type", func(t *testing.T) {
		networkWlCp := networkWl
		networkWlCp.Type = invalid

		state := SetupLoaderTests(t, []zosTypes.Workload{networkWlCp})

		_, err := state.LoadNetworkFromGrid(context.Background(), "test")
		assert.Error(t, err)
	})

	t.Run("wrong workload data", func(t *testing.T) {
		networkWlCp := networkWl
		networkWlCp.Type = zosTypes.GatewayNameProxyType
		networkWlCp.Data = zosTypes.MustMarshal(zosTypes.Network{
			WGPrivateKey: "key",
		})

		state := SetupLoaderTests(t, []zosTypes.Workload{networkWlCp})

		_, err := state.LoadNetworkFromGrid(context.Background(), "test")
		assert.Error(t, err)
	})
}

func TestLoadNetworkLightFromGrid(t *testing.T) {
	ipRange, err := zosTypes.ParseIPNet("1.1.1.1/24")
	assert.NoError(t, err)

	znet := workloads.ZNetLight{
		Name:             "test",
		Description:      "test description",
		Nodes:            []uint32{1},
		NodeDeploymentID: map[uint32]uint64{1: 10},
		NodesIPRange:     map[uint32]zosTypes.IPNet{1: ipRange},
		MyceliumKeys:     map[uint32][]byte{1: zosTypes.Bytes{}},
	}

	metadata, err := json.Marshal(workloads.NetworkMetaData{
		Version: int(workloads.Version3),
	})
	assert.NoError(t, err)

	networkWl := zosTypes.Workload{
		Version: 0,
		Name:    "test",
		Type:    zosTypes.NetworkLightType,
		Data: zosTypes.MustMarshal(zosTypes.NetworkLight{
			Subnet: ipRange,
		}),
		Metadata:    string(metadata),
		Description: "test description",
		Result:      zosTypes.Result{},
	}

	t.Run("success", func(t *testing.T) {
		state := SetupLoaderTests(t, []zosTypes.Workload{networkWl})

		got, err := state.LoadNetworkLightFromGrid(context.Background(), "test")
		assert.NoError(t, err)
		assert.Equal(t, znet, got)
	})

	t.Run("invalid type", func(t *testing.T) {
		networkWlCp := networkWl
		networkWlCp.Type = invalid

		state := SetupLoaderTests(t, []zosTypes.Workload{networkWlCp})

		_, err := state.LoadNetworkLightFromGrid(context.Background(), "test")
		assert.Error(t, err)
	})

	t.Run("wrong workload data", func(t *testing.T) {
		networkWlCp := networkWl
		networkWlCp.Type = zosTypes.GatewayNameProxyType
		networkWlCp.Data = zosTypes.MustMarshal(zosTypes.Network{
			WGPrivateKey: "key",
		})

		state := SetupLoaderTests(t, []zosTypes.Workload{networkWlCp})

		_, err := state.LoadNetworkLightFromGrid(context.Background(), "test")
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

	qsfsWl := zosTypes.Workload{
		Version:     0,
		Name:        "test",
		Type:        zosTypes.QuantumSafeFSType,
		Description: "test des",
		Data: zosTypes.MustMarshal(zosTypes.QuantumSafeFS{
			Cache: 2048 * zosTypes.Megabyte,
			Config: zosTypes.QuantumSafeFSConfig{
				MinimalShards:     10,
				ExpectedShards:    20,
				RedundantGroups:   2,
				RedundantNodes:    5,
				MaxZDBDataDirSize: 10,
				Encryption: zosTypes.Encryption{
					Algorithm: zosTypes.EncryptionAlgorithm("AES"),
					Key:       zosTypes.EncryptionKey(k),
				},
				Meta: zosTypes.QuantumSafeMeta{
					Type: "zdb",
					Config: zosTypes.QuantumSafeConfig{
						Prefix: "test",
						Encryption: zosTypes.Encryption{
							Algorithm: zosTypes.EncryptionAlgorithm("AES"),
							Key:       zosTypes.EncryptionKey(k),
						},
						Backends: []zosTypes.ZdbBackend{
							{Address: "1.1.1.1", Namespace: "test ns", Password: "password"},
						},
					},
				},
				Groups: []zosTypes.ZdbGroup{{Backends: []zosTypes.ZdbBackend{
					{Address: "2.2.2.2", Namespace: "test ns2", Password: "password2"},
				}}},
				Compression: zosTypes.QuantumCompression{
					Algorithm: "snappy",
				},
			},
		}),
		Result: zosTypes.Result{
			Created: 10000,
			State:   zosTypes.StateOk,
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
		state := SetupLoaderTests(t, []zosTypes.Workload{qsfsWl})

		got, err := state.LoadQSFSFromGrid(context.Background(), 1, "test", deploymentName)
		assert.NoError(t, err)
		assert.Equal(t, qsfs, got)
	})
	t.Run("invalid type", func(t *testing.T) {
		qsfsWlCp := qsfsWl
		qsfsWlCp.Type = invalid

		state := SetupLoaderTests(t, []zosTypes.Workload{qsfsWlCp})

		_, err := state.LoadQSFSFromGrid(context.Background(), 1, "test", deploymentName)
		assert.Error(t, err)
	})

	t.Run("wrong workload data", func(t *testing.T) {
		qsfsWlCp := qsfsWl
		qsfsWlCp.Type = zosTypes.GatewayNameProxyType
		qsfsWlCp.Data = zosTypes.MustMarshal(zos.GatewayNameProxy{
			Name: "name",
		})

		state := SetupLoaderTests(t, []zosTypes.Workload{qsfsWlCp})

		_, err := state.LoadQSFSFromGrid(context.Background(), 1, "test", deploymentName)
		assert.Error(t, err)
	})

	t.Run("invalid result data", func(t *testing.T) {
		qsfsWlCp := qsfsWl
		qsfsWlCp.Result.Data = nil

		state := SetupLoaderTests(t, []zosTypes.Workload{qsfsWlCp})

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

	pubWl := zosTypes.Workload{
		Version: 0,
		Name:    "testip",
		Type:    zosTypes.PublicIPType,
		Data: zosTypes.MustMarshal(zosTypes.PublicIP{
			V4: true,
		}),
	}

	vmWl := zosTypes.Workload{
		Version: 0,
		Name:    "test",
		Type:    zosTypes.ZMachineType,
		Data: zosTypes.MustMarshal(zosTypes.ZMachine{
			FList: "https://hub.grid.tf/tf-official-apps/base:latest.flist",
			Network: zosTypes.MachineNetwork{
				Interfaces: []zosTypes.MachineInterface{
					{
						Network: "test_network",
						IP:      net.ParseIP("1.1.1.1"),
					},
				},
				PublicIP:  pubWl.Name,
				Planetary: true,
			},
			ComputeCapacity: zosTypes.MachineCapacity{
				CPU:    uint8(2),
				Memory: 2048 * zosTypes.Megabyte,
			},
			Size:       4096 * zosTypes.Megabyte,
			Entrypoint: "entrypoint",
			Corex:      false,
			Mounts: []zosTypes.MachineMount{
				{Name: "disk", Mountpoint: "mount"},
			},
			Env: map[string]string{"var1": "val1"},
		}),
		Description: "test des",
		Result: zosTypes.Result{
			Created: 5000,
			State:   zosTypes.StateOk,
			Data:    vmRes,
		},
	}

	t.Run("success", func(t *testing.T) {
		state := SetupLoaderTests(t, []zosTypes.Workload{vmWl, pubWl})

		got, err := state.LoadVMFromGrid(context.Background(), vm.NodeID, "test", deploymentName)
		assert.NoError(t, err)
		assert.Equal(t, vm, got)
	})

	t.Run("invalid type", func(t *testing.T) {
		vmWlCp := vmWl
		vmWlCp.Type = invalid

		state := SetupLoaderTests(t, []zosTypes.Workload{vmWlCp})

		_, err := state.LoadVMFromGrid(context.Background(), vm.NodeID, "test", deploymentName)
		assert.Error(t, err)
	})

	t.Run("wrong workload data", func(t *testing.T) {
		vmWlCp := vmWl
		vmWlCp.Type = zosTypes.GatewayFQDNProxyType
		vmWlCp.Data = zosTypes.MustMarshal(zos.GatewayFQDNProxy{
			FQDN: "123",
		})

		state := SetupLoaderTests(t, []zosTypes.Workload{vmWlCp})

		_, err := state.LoadVMFromGrid(context.Background(), vm.NodeID, "test", deploymentName)
		assert.Error(t, err)
	})

	t.Run("invalid result data", func(t *testing.T) {
		vmWlCp := vmWl
		vmWlCp.Result.Data = nil

		state := SetupLoaderTests(t, []zosTypes.Workload{vmWlCp})

		_, err := state.LoadVMFromGrid(context.Background(), vm.NodeID, "test", deploymentName)
		assert.Error(t, err)
	})
}

func TestLoadVMLightFromGrid(t *testing.T) {
	vmRes, err := json.Marshal(zos.ZMachineResult{
		ID:          "5",
		IP:          "5.5.5.5",
		PlanetaryIP: "203:8b0b:5f3e:b859:c36:efdf:ab6e:50cc",
	})
	assert.NoError(t, err)

	var zlogs []workloads.Zlog

	vm := workloads.VMLight{
		Name:          "test",
		NodeID:        1,
		Flist:         "https://hub.grid.tf/tf-official-apps/base:latest.flist",
		FlistChecksum: "f94b5407f2e8635bd1b6b3dac7fef2d9",
		Corex:         false,
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

	vmWl := zosTypes.Workload{
		Version: 0,
		Name:    "test",
		Type:    zosTypes.ZMachineLightType,
		Data: zosTypes.MustMarshal(zosTypes.ZMachineLight{
			FList: "https://hub.grid.tf/tf-official-apps/base:latest.flist",
			Network: zosTypes.MachineNetworkLight{
				Interfaces: []zosTypes.MachineInterface{
					{
						Network: "test_network",
						IP:      net.ParseIP("1.1.1.1"),
					},
				},
			},
			ComputeCapacity: zosTypes.MachineCapacity{
				CPU:    uint8(2),
				Memory: 2048 * zosTypes.Megabyte,
			},
			Size:       4096 * zosTypes.Megabyte,
			Entrypoint: "entrypoint",
			Corex:      false,
			Mounts: []zosTypes.MachineMount{
				{Name: "disk", Mountpoint: "mount"},
			},
			Env: map[string]string{"var1": "val1"},
		}),
		Description: "test des",
		Result: zosTypes.Result{
			Created: 5000,
			State:   zosTypes.StateOk,
			Data:    vmRes,
		},
	}

	t.Run("success", func(t *testing.T) {
		state := SetupLoaderTests(t, []zosTypes.Workload{vmWl})

		got, err := state.LoadVMLightFromGrid(context.Background(), vm.NodeID, "test", deploymentName)
		assert.NoError(t, err)
		assert.Equal(t, vm, got)
	})

	t.Run("invalid type", func(t *testing.T) {
		vmWlCp := vmWl
		vmWlCp.Type = invalid

		state := SetupLoaderTests(t, []zosTypes.Workload{vmWlCp})

		_, err := state.LoadVMLightFromGrid(context.Background(), vm.NodeID, "test", deploymentName)
		assert.Error(t, err)
	})

	t.Run("wrong workload data", func(t *testing.T) {
		vmWlCp := vmWl
		vmWlCp.Type = zosTypes.GatewayFQDNProxyType
		vmWlCp.Data = zosTypes.MustMarshal(zos.GatewayFQDNProxy{
			FQDN: "123",
		})

		state := SetupLoaderTests(t, []zosTypes.Workload{vmWlCp})

		_, err := state.LoadVMLightFromGrid(context.Background(), vm.NodeID, "test", deploymentName)
		assert.Error(t, err)
	})

	t.Run("invalid result data", func(t *testing.T) {
		vmWlCp := vmWl
		vmWlCp.Result.Data = nil

		state := SetupLoaderTests(t, []zosTypes.Workload{vmWlCp})

		_, err := state.LoadVMLightFromGrid(context.Background(), vm.NodeID, "test", deploymentName)
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

	zdbWl := zosTypes.Workload{
		Name:        "test",
		Type:        zosTypes.ZDBType,
		Description: "test des",
		Version:     0,
		Result: zosTypes.Result{
			Created: 1000,
			State:   zosTypes.StateOk,
			Data:    res,
		},
		Data: zosTypes.MustMarshal(zosTypes.ZDB{
			Size:     100 * zosTypes.Gigabyte,
			Mode:     "user",
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
		state := SetupLoaderTests(t, []zosTypes.Workload{zdbWl})

		got, err := state.LoadZdbFromGrid(context.Background(), 1, "test", deploymentName)
		assert.NoError(t, err)
		assert.Equal(t, zdb, got)
	})

	t.Run("invalid type", func(t *testing.T) {
		zdbWlCp := zdbWl
		zdbWlCp.Type = invalid

		state := SetupLoaderTests(t, []zosTypes.Workload{zdbWlCp})

		_, err := state.LoadZdbFromGrid(context.Background(), 1, "test", deploymentName)
		assert.Error(t, err)
	})

	t.Run("wrong workload data", func(t *testing.T) {
		zdbWlCp := zdbWl
		zdbWlCp.Type = zosTypes.GatewayNameProxyType
		zdbWlCp.Data = zosTypes.MustMarshal(zos.GatewayNameProxy{
			Name: "name",
		})

		state := SetupLoaderTests(t, []zosTypes.Workload{zdbWlCp})

		_, err := state.LoadZdbFromGrid(context.Background(), 1, "test", deploymentName)
		assert.Error(t, err)
	})

	t.Run("invalid result data", func(t *testing.T) {
		zdbWlCp := zdbWl
		zdbWlCp.Result.Data = nil

		state := SetupLoaderTests(t, []zosTypes.Workload{zdbWlCp})

		_, err := state.LoadZdbFromGrid(context.Background(), 1, "test", deploymentName)
		assert.Error(t, err)
	})
}
