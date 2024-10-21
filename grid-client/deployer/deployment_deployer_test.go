package deployer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"net"
	"testing"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/mocks"
	client "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/state"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	zosTypes "github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

var (
	twinID        uint32 = 13
	contractID    uint64 = 100
	netContractID uint64 = 101
	nodeID        uint32 = 10
)

func constructTestDeployment() workloads.Deployment {
	disks := []workloads.Disk{
		{
			Name:        "disk1",
			SizeGB:      1024,
			Description: "disk1_description",
		},
		{
			Name:        "disk2",
			SizeGB:      2048,
			Description: "disk2_description",
		},
	}

	zdbs := []workloads.ZDB{
		{
			Name:        "zdb1",
			Password:    "pass1",
			Public:      true,
			SizeGB:      1024,
			Description: "zdb_description",
			Mode:        workloads.ZDBModeUser,
			IPs: []string{
				"::1",
				"::2",
			},
			Port:      9000,
			Namespace: "ns1",
		},
		{
			Name:        "zdb2",
			Password:    "pass2",
			Public:      true,
			SizeGB:      1024,
			Description: "zdb2_description",
			Mode:        workloads.ZDBModeUser,
			IPs: []string{
				"::3",
				"::4",
			},
			Port:      9001,
			Namespace: "ns2",
		},
	}

	vms := []workloads.VM{
		{
			Name:         "vm1",
			NodeID:       nodeID,
			Flist:        "https://hub.grid.tf/tf-official-apps/discourse-v4.0.flist",
			PublicIP:     true,
			PublicIP6:    true,
			Planetary:    true,
			Corex:        true,
			ComputedIP:   "5.5.5.5/24",
			ComputedIP6:  "::7/64",
			PlanetaryIP:  "::8/64",
			IP:           "10.1.0.2",
			Description:  "vm1_description",
			CPU:          1,
			MemoryMB:     1024,
			RootfsSizeMB: 1024,
			Entrypoint:   "/sbin/zinit init",
			Mounts: []workloads.Mount{
				{
					Name:       "disk1",
					MountPoint: "/data1",
				},
				{
					Name:       "disk2",
					MountPoint: "/data2",
				},
			},
			Zlogs: []workloads.Zlog{
				{
					Zmachine: "vm1",
					Output:   "redis://codescalers1.com",
				},
				{
					Zmachine: "vm1",
					Output:   "redis://threefold1.io",
				},
			},
			EnvVars: map[string]string{
				"ssh_key":  "asd",
				"ssh_key2": "asd2",
			},
			NetworkName: "network",
		},
		{
			Name:          "vm2",
			NodeID:        nodeID,
			Flist:         "https://hub.grid.tf/omar0.3bot/omarelawady-ubuntu-20.04.flist",
			FlistChecksum: "f0ae02b6244db3a5f842decd082c4e08",
			PublicIP:      false,
			PublicIP6:     true,
			Planetary:     true,
			Corex:         true,
			ComputedIP:    "",
			ComputedIP6:   "::7/64",
			PlanetaryIP:   "::8/64",
			IP:            "10.1.0.2",
			Description:   "vm2_description",
			CPU:           1,
			MemoryMB:      1024,
			RootfsSizeMB:  1024,
			Entrypoint:    "/sbin/zinit init",
			Mounts: []workloads.Mount{
				{
					Name:       "disk1",
					MountPoint: "/data1",
				},
				{
					Name:       "disk2",
					MountPoint: "/data2",
				},
			},
			Zlogs: []workloads.Zlog{
				{
					Zmachine: "vm2",
					Output:   "redis://codescalers.com",
				},
				{
					Zmachine: "vm2",
					Output:   "redis://threefold.io",
				},
			},
			EnvVars: map[string]string{
				"ssh_key":  "asd",
				"ssh_key2": "asd2",
			},
			NetworkName: "network",
		},
	}

	qsfs := []workloads.QSFS{
		{
			Name:                 "name1",
			Description:          "description1",
			Cache:                1024,
			MinimalShards:        4,
			ExpectedShards:       4,
			RedundantGroups:      0,
			RedundantNodes:       0,
			MaxZDBDataDirSize:    512,
			EncryptionAlgorithm:  "AES",
			EncryptionKey:        "4d778ba3216e4da4231540c92a55f06157cabba802f9b68fb0f78375d2e825af",
			CompressionAlgorithm: "snappy",
			Metadata: workloads.Metadata{
				Type:                "zdb",
				Prefix:              "hamada",
				EncryptionAlgorithm: "AES",
				EncryptionKey:       "4d778ba3216e4da4231540c92a55f06157cabba802f9b68fb0f78375d2e825af",
				Backends: workloads.Backends{
					{
						Address:   "[::10]:8080",
						Namespace: "ns1",
						Password:  "123",
					},
					{
						Address:   "[::11]:8080",
						Namespace: "ns2",
						Password:  "1234",
					},
					{
						Address:   "[::12]:8080",
						Namespace: "ns3",
						Password:  "1235",
					},
					{
						Address:   "[::13]:8080",
						Namespace: "ns4",
						Password:  "1236",
					},
				},
			},
			Groups: workloads.Groups{
				{
					Backends: workloads.Backends{
						{
							Address:   "[::110]:8080",
							Namespace: "ns5",
							Password:  "123",
						},
						{
							Address:   "[::111]:8080",
							Namespace: "ns6",
							Password:  "1234",
						},
						{
							Address:   "[::112]:8080",
							Namespace: "ns7",
							Password:  "1235",
						},
						{
							Address:   "[::113]:8080",
							Namespace: "ns8",
							Password:  "1236",
						},
					},
				},
			},
			MetricsEndpoint: "http://[::12]:9090/metrics",
		},
	}

	return workloads.Deployment{
		Name:        "test",
		NodeID:      nodeID,
		Disks:       disks,
		Zdbs:        zdbs,
		Vms:         vms,
		QSFS:        qsfs,
		NetworkName: "network",
	}
}

func constructTestDeployer(t *testing.T, tfPluginClient TFPluginClient, mock bool) (DeploymentDeployer, *mocks.RMBMockClient, *mocks.MockSubstrateExt, *mocks.MockNodeClientGetter, *mocks.MockDeployer) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cl := mocks.NewRMBMockClient(ctrl)
	sub := mocks.NewMockSubstrateExt(ctrl)
	ncPool := mocks.NewMockNodeClientGetter(ctrl)
	deployer := mocks.NewMockDeployer(ctrl)

	if mock {
		tfPluginClient.SubstrateConn = sub
		tfPluginClient.NcPool = ncPool
		tfPluginClient.RMB = cl

		tfPluginClient.State.NcPool = ncPool
		tfPluginClient.State.Substrate = sub

		tfPluginClient.DeploymentDeployer.tfPluginClient = &tfPluginClient
		tfPluginClient.DeploymentDeployer.deployer = deployer
	}

	return tfPluginClient.DeploymentDeployer, cl, sub, ncPool, deployer
}

func mustMarshal(t *testing.T, v interface{}) json.RawMessage {
	r, err := json.Marshal(v)
	assert.NoError(t, err)
	return r
}

func musUnmarshal(t *testing.T, bs json.RawMessage, v interface{}) {
	err := json.Unmarshal(bs, v)
	assert.NoError(t, err)
}

func TestDeploymentDeployerValidate(t *testing.T) {
	tfPluginClient, err := setup()
	assert.NoError(t, err)

	d, _, sub, _, _ := constructTestDeployer(t, tfPluginClient, false)

	t.Run("test validate flist checksum ", func(t *testing.T) {
		dl := constructTestDeployment()

		network := dl.NetworkName
		checksum := dl.Vms[0].FlistChecksum
		dl.NetworkName = network

		dl.Vms[0].FlistChecksum += " "
		assert.Error(t, d.Validate(context.Background(), []*workloads.Deployment{&dl}))

		dl.Vms[0].FlistChecksum = checksum
		assert.NoError(t, d.Validate(context.Background(), []*workloads.Deployment{&dl}))
	})

	t.Run("Validation failed", func(t *testing.T) {
		dl := constructTestDeployment()

		sub.EXPECT().
			GetBalance(d.tfPluginClient.Identity).
			Return(substrate.Balance{
				Free: types.U128{
					Int: big.NewInt(10),
				},
			}, nil)

		assert.Error(t, d.Deploy(context.Background(), &dl))

		// nothing should change
		assert.Empty(t, dl.NodeDeploymentID)
		assert.Empty(t, dl.ContractID)
		assert.Equal(t, d.tfPluginClient.State.CurrentNodeDeployments, map[uint32]state.ContractIDs{})
	})
}

func TestDeploymentDeployerDeploy(t *testing.T) {
	tfPluginClient, err := setup()
	assert.NoError(t, err)

	d, cl, sub, ncPool, deployer := constructTestDeployer(t, tfPluginClient, true)
	dl := constructTestDeployment()
	net := constructTestNetwork()

	t.Run("test generate deployment", func(t *testing.T) {
		gridDl, err := dl.ZosDeployment(twinID)
		assert.NoError(t, err)

		d.tfPluginClient.State.CurrentNodeDeployments[nodeID] = append(d.tfPluginClient.State.CurrentNodeDeployments[nodeID], netContractID)
		d.tfPluginClient.State.Networks = state.NetworkState{
			State: map[string]state.Network{net.Name: {
				Subnets: map[uint32]string{nodeID: net.IPRange.String()},
			}},
		}

		ncPool.EXPECT().
			GetNodeClient(sub, nodeID).
			Return(client.NewNodeClient(twinID, cl, d.tfPluginClient.RMBTimeout), nil).AnyTimes()

		cl.EXPECT().
			Call(gomock.Any(), twinID, "zos.network.list_private_ips", gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, twin uint32, fn string, data, result interface{}) error {
				var res *[]string = result.(*[]string)
				*res = []string{}
				return nil
			}).AnyTimes()

		dls, err := d.GenerateVersionlessDeployments(context.Background(), []*workloads.Deployment{&dl})
		assert.NoError(t, err)

		assert.Equal(t, len(gridDl.Workloads), len(dls[dl.NodeID][0].Workloads))
		assert.Equal(t, gridDl.Workloads, dls[dl.NodeID][0].Workloads)

		assert.Equal(t, dls[nodeID][0].Metadata, "{\"version\":3,\"type\":\"vm\",\"name\":\"test\",\"projectName\":\"vm/test\"}")
	})

	t.Run("Deploying failed", func(t *testing.T) {
		sub.EXPECT().
			GetBalance(d.tfPluginClient.Identity).
			Return(substrate.Balance{
				Free: types.U128{
					Int: big.NewInt(20000000),
				},
			}, nil)

		deployer.EXPECT().Deploy(
			gomock.Any(),
			dl.NodeDeploymentID,
			gomock.Any(),
			gomock.Any(),
		).Return(map[uint32]uint64{}, errors.New("error"))

		assert.Error(t, d.Deploy(context.Background(), &dl))

		// nothing should change
		assert.Empty(t, dl.NodeDeploymentID)
		assert.Empty(t, dl.ContractID)
		assert.Equal(t, d.tfPluginClient.State.CurrentNodeDeployments, map[uint32]state.ContractIDs{nodeID: {netContractID}})
	})

	t.Run("Deploying succeeded", func(t *testing.T) {
		dl.NodeDeploymentID = map[uint32]uint64{}
		sub.EXPECT().
			GetBalance(d.tfPluginClient.Identity).
			Return(substrate.Balance{
				Free: types.U128{
					Int: big.NewInt(20000000),
				},
			}, nil)

		deployer.EXPECT().Deploy(
			gomock.Any(),
			dl.NodeDeploymentID,
			gomock.Any(),
			gomock.Any(),
		).Return(map[uint32]uint64{nodeID: contractID}, nil)
		assert.NoError(t, d.Deploy(context.Background(), &dl))

		// should reflect on deployment and state
		assert.Equal(t, dl.NodeDeploymentID, map[uint32]uint64{nodeID: contractID})
		assert.Equal(t, dl.ContractID, contractID)
		assert.Equal(t, d.tfPluginClient.State.CurrentNodeDeployments, map[uint32]state.ContractIDs{nodeID: {netContractID, contractID}})
	})
}

func TestDeploymentDeployerDelete(t *testing.T) {
	tfPluginClient, err := setup()
	assert.NoError(t, err)

	d, _, sub, _, deployer := constructTestDeployer(t, tfPluginClient, true)
	dl := constructTestDeployment()

	t.Run("test delete", func(t *testing.T) {
		dl.ContractID = contractID
		dl.NodeDeploymentID = map[uint32]uint64{nodeID: contractID}

		d.tfPluginClient.State.CurrentNodeDeployments = map[uint32]state.ContractIDs{nodeID: {contractID}}

		t.Run("Validation failed", func(t *testing.T) {
			sub.EXPECT().
				GetBalance(d.tfPluginClient.Identity).
				Return(substrate.Balance{
					Free: types.U128{
						Int: big.NewInt(10),
					},
				}, nil)

			assert.Error(t, d.Cancel(context.Background(), &dl))

			// nothing should change
			assert.Equal(t, dl.ContractID, contractID)
			assert.Equal(t, d.tfPluginClient.State.CurrentNodeDeployments, map[uint32]state.ContractIDs{nodeID: {contractID}})
			assert.Equal(t, dl.NodeDeploymentID, map[uint32]uint64{nodeID: contractID})
		})
		t.Run("Canceling failed", func(t *testing.T) {
			sub.EXPECT().
				GetBalance(d.tfPluginClient.Identity).
				Return(substrate.Balance{
					Free: types.U128{
						Int: big.NewInt(20000000),
					},
				}, nil)

			deployer.EXPECT().
				Cancel(gomock.Any(), contractID).
				Return(errors.New("error"))

			assert.Error(t, d.Cancel(context.Background(), &dl))

			// nothing should change
			assert.Equal(t, dl.ContractID, contractID)
			assert.Equal(t, d.tfPluginClient.State.CurrentNodeDeployments, map[uint32]state.ContractIDs{nodeID: {contractID}})
			assert.Equal(t, dl.NodeDeploymentID, map[uint32]uint64{nodeID: contractID})
		})
		t.Run("Canceling succeeded", func(t *testing.T) {
			sub.EXPECT().
				GetBalance(d.tfPluginClient.Identity).
				Return(substrate.Balance{
					Free: types.U128{
						Int: big.NewInt(20000000),
					},
				}, nil)

			deployer.EXPECT().
				Cancel(gomock.Any(), contractID).
				Return(nil)
			assert.NoError(t, d.Cancel(context.Background(), &dl))

			// should reflect on state and deployment
			assert.Empty(t, dl.ContractID)
			assert.Empty(t, d.tfPluginClient.State.CurrentNodeDeployments[dl.NodeID])
			assert.Empty(t, dl.NodeDeploymentID)
		})
	})
}

func TestDeploymentDeployerSync(t *testing.T) {
	tfPluginClient, err := setup()
	assert.NoError(t, err)

	d, cl, sub, ncPool, deployer := constructTestDeployer(t, tfPluginClient, true)
	dl := constructTestDeployment()

	t.Run("test sync", func(t *testing.T) {
		dl.ContractID = contractID
		dl.NodeDeploymentID = make(map[uint32]uint64)

		net := constructTestNetwork()
		workload := net.ZosWorkload(net.NodesIPRange[nodeID], "", uint16(0), []zosTypes.Peer{}, "", nil)
		networkDl := workloads.NewGridDeployment(twinID, 0, []zosTypes.Workload{workload})

		d.tfPluginClient.State.CurrentNodeDeployments[nodeID] = append(d.tfPluginClient.State.CurrentNodeDeployments[nodeID], contractID)
		d.tfPluginClient.State.Networks = state.NetworkState{
			State: map[string]state.Network{net.Name: {
				Subnets: map[uint32]string{nodeID: net.IPRange.String()},
			}},
		}

		ncPool.EXPECT().
			GetNodeClient(sub, nodeID).
			Return(client.NewNodeClient(twinID, cl, d.tfPluginClient.RMBTimeout), nil).AnyTimes()

		cl.EXPECT().
			Call(gomock.Any(), twinID, "zos.deployment.get", gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, twin uint32, fn string, data, result interface{}) error {
				var res *zosTypes.Deployment = result.(*zosTypes.Deployment)
				*res = networkDl
				return nil
			}).AnyTimes()

		ncPool.EXPECT().
			GetNodeClient(sub, nodeID).
			Return(client.NewNodeClient(twinID, cl, d.tfPluginClient.RMBTimeout), nil)

		cl.EXPECT().
			Call(gomock.Any(), twinID, "zos.network.list_private_ips", gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, twin uint32, fn string, data, result interface{}) error {
				var res *[]string = result.(*[]string)
				*res = []string{}
				return nil
			})

		dls, err := d.GenerateVersionlessDeployments(context.Background(), []*workloads.Deployment{&dl})
		assert.NoError(t, err)

		gridDl := dls[dl.NodeID][0]
		err = json.NewEncoder(log.Writer()).Encode(gridDl.Workloads)
		assert.NoError(t, err)

		for i, w := range gridDl.Workloads {
			if w.Type == zos.ZLogsType.String() || w.Type == zos.ZMountType.String() {
				gridDl.Workloads[i] = w.WithResults(zosTypes.Result{
					State: zosTypes.StateOk,
				})
			}

			if w.Type == zos.ZMachineType.String() {
				if w.Name == dl.Vms[0].Name {
					gridDl.Workloads[i] = w.WithResults(zosTypes.Result{
						State: zosTypes.StateOk,
						Data: mustMarshal(t, zos.ZMachineResult{
							IP:          dl.Vms[0].IP,
							PlanetaryIP: dl.Vms[0].PlanetaryIP,
						}),
					})
				}

				if w.Name == dl.Vms[1].Name {
					gridDl.Workloads[i] = w.WithResults(zosTypes.Result{
						State: zosTypes.StateOk,
						Data: mustMarshal(t, zos.ZMachineResult{
							IP:          dl.Vms[1].IP,
							PlanetaryIP: dl.Vms[1].PlanetaryIP,
						}),
					})
				}
			}

			if w.Type == zos.QuantumSafeFSType.String() {
				gridDl.Workloads[i] = w.WithResults(zosTypes.Result{
					State: zosTypes.StateOk,
					Data: mustMarshal(t, zos.QuatumSafeFSResult{
						MetricsEndpoint: dl.QSFS[0].MetricsEndpoint,
					}),
				})
			}

			if w.Type == zos.ZDBType.String() {
				if w.Name == dl.Zdbs[0].Name {
					gridDl.Workloads[i] = w.WithResults(zosTypes.Result{
						State: zosTypes.StateOk,
						Data: mustMarshal(t, zos.ZDBResult{
							Namespace: dl.Zdbs[0].Namespace,
							IPs:       dl.Zdbs[0].IPs,
							Port:      uint(dl.Zdbs[0].Port),
						}),
					})
				}

				if w.Name == dl.Zdbs[1].Name {
					gridDl.Workloads[i] = w.WithResults(zosTypes.Result{
						State: zosTypes.StateOk,
						Data: mustMarshal(t, zos.ZDBResult{
							Namespace: dl.Zdbs[1].Namespace,
							IPs:       dl.Zdbs[1].IPs,
							Port:      uint(dl.Zdbs[1].Port),
						}),
					})
				}
			}

			if w.Type == zos.PublicIPType.String() {
				if w.Name == dl.Vms[0].Name+"ip" {
					gridDl.Workloads[i] = w.WithResults(zosTypes.Result{
						State: zosTypes.StateOk,
						Data: mustMarshal(t, zos.PublicIPResult{
							IP:   gridtypes.MustParseIPNet(dl.Vms[0].ComputedIP),
							IPv6: gridtypes.MustParseIPNet(dl.Vms[0].ComputedIP6),
						}),
					})
				}

				if w.Name == dl.Vms[1].Name+"ip" {
					gridDl.Workloads[i] = w.WithResults(zosTypes.Result{
						State: zosTypes.StateOk,
						Data: mustMarshal(t, zos.PublicIPResult{
							IP:   gridtypes.MustParseIPNet(dl.Vms[1].ComputedIP),
							IPv6: gridtypes.MustParseIPNet(dl.Vms[1].ComputedIP6),
						}),
					})
				}
			}
		}

		for i := 0; 2*i < len(gridDl.Workloads); i++ {
			gridDl.Workloads[i], gridDl.Workloads[len(gridDl.Workloads)-1-i] = gridDl.Workloads[len(gridDl.Workloads)-1-i], gridDl.Workloads[i]
		}

		sub.EXPECT().IsValidContract(contractID).Return(true, nil)

		var cp workloads.Deployment
		musUnmarshal(t, mustMarshal(t, dl), &cp)

		_, err = workloads.GetUsedIPs(gridDl, nodeID)
		assert.NoError(t, err)

		deployer.EXPECT().
			GetDeployments(gomock.Any(), map[uint32]uint64{}).
			Return(map[uint32]zosTypes.Deployment{nodeID: gridDl}, nil)
		// manager.EXPECT().Commit(context.Background()).AnyTimes()
		assert.NoError(t, d.Sync(context.Background(), &dl))
		assert.Equal(t, dl.Vms, cp.Vms)
		assert.Equal(t, dl.Disks, cp.Disks)
		assert.Equal(t, dl.QSFS, cp.QSFS)
		assert.Equal(t, dl.Zdbs, cp.Zdbs)
		assert.Equal(t, dl.ContractID, cp.ContractID)
		assert.Equal(t, dl.NodeID, cp.NodeID)
	})

	t.Run("test sync deleted contract", func(t *testing.T) {
		dl.ContractID = contractID
		sub.EXPECT().IsValidContract(dl.ContractID).Return(false, nil).AnyTimes()

		err := d.syncContract(&dl)
		assert.NoError(t, err)
		assert.Equal(t, dl.ContractID, uint64(0))
		dl.ContractID = contractID
		dl.NodeDeploymentID = map[uint32]uint64{dl.NodeID: dl.ContractID}

		deployer.EXPECT().
			GetDeployments(gomock.Any(), map[uint32]uint64{nodeID: contractID}).
			Return(map[uint32]zosTypes.Deployment{}, nil)

		assert.NoError(t, d.Sync(context.Background(), &dl))

		assert.Equal(t, dl.ContractID, uint64(0))
		assert.Empty(t, dl.Vms)
		assert.Empty(t, dl.Disks)
		assert.Empty(t, dl.QSFS)
		assert.Empty(t, dl.Zdbs)
	})
}

func ExampleDeploymentDeployer_Deploy() {
	const mnemonic = "<mnemonics goes here>"
	const network = "<dev, test, qa, main>"
	const nodeID = 11 // use any node with status up, use ExampleFilterNodes to get valid nodeID

	tfPluginClient, err := NewTFPluginClient(mnemonic, WithNetwork(network))
	if err != nil {
		fmt.Println(err)
		return
	}

	n := workloads.ZNet{
		Name:        "network",
		Description: "network for testing",
		Nodes:       []uint32{nodeID},
		IPRange: zosTypes.IPNet{IPNet: net.IPNet{
			IP:   net.IPv4(10, 1, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}},
		AddWGAccess: false,
	}

	vm := workloads.VM{
		Name:        "vm",
		NodeID:      nodeID,
		Flist:       "https://hub.grid.tf/tf-official-apps/base:latest.flist",
		CPU:         2,
		Planetary:   true,
		MemoryMB:    1024,
		Entrypoint:  "/sbin/zinit init",
		EnvVars:     map[string]string{"SSH_KEY": "<ssh key goes here>"},
		NetworkName: n.Name,
	}

	err = tfPluginClient.NetworkDeployer.Deploy(context.Background(), &n)
	if err != nil {
		fmt.Println(err)
		return
	}

	dl := workloads.NewDeployment("vmdeployment", nodeID, "", nil, n.Name, nil, nil, []workloads.VM{vm}, nil, nil, nil)
	err = tfPluginClient.DeploymentDeployer.Deploy(context.Background(), &dl)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("deployment is done successfully")
}

func ExampleDeploymentDeployer_BatchDeploy() {
	const mnemonic = "<mnemonics goes here>"
	const network = "<dev, test, qa, main>"
	const nodeID = 11 // use any node with status up, use ExampleFilterNodes to get valid nodeID

	tfPluginClient, err := NewTFPluginClient(mnemonic, WithNetwork(network))
	if err != nil {
		fmt.Println(err)
		return
	}

	n := workloads.ZNet{
		Name:        "network",
		Description: "network for testing",
		Nodes:       []uint32{nodeID},
		IPRange: zosTypes.IPNet{IPNet: net.IPNet{
			IP:   net.IPv4(10, 1, 0, 0),
			Mask: net.CIDRMask(16, 32),
		}},
		AddWGAccess: false,
	}

	vm1 := workloads.VM{
		Name:        "vm1",
		NodeID:      nodeID,
		Flist:       "https://hub.grid.tf/tf-official-apps/base:latest.flist",
		CPU:         2,
		Planetary:   true,
		MemoryMB:    1024,
		Entrypoint:  "/sbin/zinit init",
		EnvVars:     map[string]string{"SSH_KEY": "<ssh key goes here>"},
		NetworkName: n.Name,
	}
	vm2 := workloads.VM{
		Name:        "vm2",
		NodeID:      nodeID,
		Flist:       "https://hub.grid.tf/tf-official-apps/base:latest.flist",
		CPU:         2,
		Planetary:   true,
		MemoryMB:    1024,
		Entrypoint:  "/sbin/zinit init",
		EnvVars:     map[string]string{"SSH_KEY": "<ssh key goes here>"},
		NetworkName: n.Name,
	}

	err = tfPluginClient.NetworkDeployer.Deploy(context.Background(), &n)
	if err != nil {
		fmt.Println(err)
		return
	}

	d1 := workloads.NewDeployment("vm1deployment", nodeID, "", nil, n.Name, nil, nil, []workloads.VM{vm1}, nil, nil, nil)
	d2 := workloads.NewDeployment("vm2deployment", nodeID, "", nil, n.Name, nil, nil, []workloads.VM{vm2}, nil, nil, nil)
	err = tfPluginClient.DeploymentDeployer.BatchDeploy(context.Background(), []*workloads.Deployment{&d1, &d2})
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("batch deployment is done successfully")
}

func ExampleDeploymentDeployer_Cancel() {
	const mnemonic = "<mnemonics goes here>"
	const network = "<dev, test, qa, main>"
	const nodeID = 11 // use any node with status up, use ExampleFilterNodes to get valid nodeID

	tfPluginClient, err := NewTFPluginClient(mnemonic, WithNetwork(network))
	if err != nil {
		fmt.Println(err)
		return
	}

	// dl should be a valid and existing deployment name
	deploymentName := "vmdeployment"
	dl, err := tfPluginClient.State.LoadDeploymentFromGrid(context.Background(), nodeID, deploymentName)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = tfPluginClient.DeploymentDeployer.Cancel(context.Background(), &dl)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("deployment is canceled successfully")
}
