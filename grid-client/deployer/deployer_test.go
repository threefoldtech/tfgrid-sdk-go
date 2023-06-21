// Package deployer for grid deployer
package deployer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/mocks"
	client "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/subi"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	proxyTypes "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
	"golang.org/x/exp/slices"
)

var backendURLWithTLSPassthrough = "1.1.1.1:10"
var backendURLWithoutTLSPassthrough = "http://1.1.1.1:10"

func setup() (TFPluginClient, error) {
	mnemonics := os.Getenv("MNEMONICS")
	log.Debug().Msgf("mnemonics: %s", mnemonics)

	network := os.Getenv("NETWORK")
	log.Debug().Msgf("network: %s", network)

	return NewTFPluginClient(mnemonics, "sr25519", network, "", "", "", 0, true)
}

type gatewayWorkloadGenerator interface {
	ZosWorkload() gridtypes.Workload
}

func newDeploymentWithGateway(identity substrate.Identity, twinID uint32, version uint32, gw gatewayWorkloadGenerator) (gridtypes.Deployment, error) {
	dl := workloads.NewGridDeployment(twinID, []gridtypes.Workload{})
	dl.Version = version

	dl.Workloads = append(dl.Workloads, gw.ZosWorkload())
	dl.Workloads[0].Version = version

	err := dl.Sign(twinID, identity)
	if err != nil {
		return gridtypes.Deployment{}, err
	}

	return dl, nil
}

func deploymentWithNameGateway(identity substrate.Identity, twinID uint32, TLSPassthrough bool, version uint32, backendURL string) (gridtypes.Deployment, error) {
	gw := workloads.GatewayNameProxy{
		Name:           "name",
		TLSPassthrough: TLSPassthrough,
		Backends:       []zos.Backend{zos.Backend(backendURL)},
	}

	return newDeploymentWithGateway(identity, twinID, version, &gw)
}

func deploymentWithFQDN(identity substrate.Identity, twinID uint32, version uint32) (gridtypes.Deployment, error) {
	gw := workloads.GatewayFQDNProxy{
		Name:     "fqdn",
		FQDN:     "a.b.com",
		Backends: []zos.Backend{zos.Backend(backendURLWithoutTLSPassthrough)},
	}

	return newDeploymentWithGateway(identity, twinID, version, &gw)
}

func hash(dl *gridtypes.Deployment) (string, error) {
	hash, err := dl.ChallengeHash()
	if err != nil {
		return "", err
	}
	hashHex := hex.EncodeToString(hash)
	return hashHex, nil
}

func mockDeployerValidator(d *Deployer, ctrl *gomock.Controller, nodes []uint32) {
	proxyCl := mocks.NewMockClient(ctrl)
	d.gridProxyClient = proxyCl

	for _, nodeID := range nodes {
		proxyCl.EXPECT().
			Node(nodeID).
			Return(proxyTypes.NodeWithNestedCapacity{
				FarmID: 1,
				PublicConfig: proxyTypes.PublicConfig{
					Ipv4:   "1.1.1.1",
					Domain: "test",
				},
			}, nil)

		proxyCl.EXPECT().Farms(gomock.Any(), gomock.Any()).Return([]proxyTypes.Farm{{FarmID: 1}}, 1, nil).AnyTimes()
	}
}

func TestDeployer(t *testing.T) {
	tfPluginClient, err := setup()
	assert.NoError(t, err)

	identity := tfPluginClient.Identity
	twinID := tfPluginClient.TwinID

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("test create", func(t *testing.T) {
		cl := mocks.NewRMBMockClient(ctrl)
		sub := mocks.NewMockSubstrateExt(ctrl)
		ncPool := mocks.NewMockNodeClientGetter(ctrl)

		tfPluginClient.SubstrateConn = sub
		tfPluginClient.NcPool = ncPool
		tfPluginClient.RMB = cl

		deployer := NewDeployer(
			tfPluginClient,
			true,
		)

		dl1, err := deploymentWithNameGateway(identity, twinID, true, 0, backendURLWithTLSPassthrough)
		assert.NoError(t, err)
		dl2, err := deploymentWithFQDN(identity, twinID, 0)
		assert.NoError(t, err)

		newDls := map[uint32]gridtypes.Deployment{
			10: dl1,
			20: dl2,
		}

		newDlsSolProvider := map[uint32]*uint64{
			10: nil,
			20: nil,
		}

		dl1.ContractID = 100
		dl2.ContractID = 200

		dl1Hash, err := hash(&dl1)
		assert.NoError(t, err)
		dl2Hash, err := hash(&dl2)
		assert.NoError(t, err)

		mockDeployerValidator(&deployer, ctrl, []uint32{10, 20})

		contractsData := []substrate.BatchCreateContractData{
			{
				Node:               10,
				Body:               "",
				Hash:               dl1Hash,
				PublicIPs:          0,
				SolutionProviderID: nil,
			},
			{
				Node:               20,
				Body:               "",
				Hash:               dl2Hash,
				PublicIPs:          0,
				SolutionProviderID: nil,
			},
		}

		sub.EXPECT().BatchCreateContract(identity, gomock.Any()).
			DoAndReturn(func(identity substrate.Identity, data []substrate.BatchCreateContractData) ([]uint64, *int, error) {
				if !slices.Contains(data, contractsData[0]) || !slices.Contains(data, contractsData[1]) {
					return nil, nil, fmt.Errorf("unexpected call to BatchCreateContract with contracts data %+v", data)
				}

				if data[0] == contractsData[0] {
					return []uint64{100, 200}, nil, nil
				}

				return []uint64{200, 100}, nil, nil
			})

		ncPool.EXPECT().
			GetNodeClient(sub, gomock.Any()).
			DoAndReturn(func(sub subi.SubstrateExt, nodeID uint32) (*client.NodeClient, error) {
				if nodeID == 10 {
					return client.NewNodeClient(13, cl, tfPluginClient.RMBTimeout), nil
				}

				return client.NewNodeClient(23, cl, tfPluginClient.RMBTimeout), nil
			}).Times(4)

		cl.EXPECT().
			Call(gomock.Any(), gomock.Any(), "zos.deployment.deploy", gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, nodeTwinID uint32, fn string, data, result interface{}) error {
				switch nodeTwinID {
				case 13:
					dl1.Workloads[0].Result.State = gridtypes.StateOk
				case 23:
					dl2.Workloads[0].Result.State = gridtypes.StateOk
				default:
					return fmt.Errorf("unexpected rmb call with node id %d", nodeID)
				}

				return nil
			}).Times(2)

		cl.EXPECT().
			Call(gomock.Any(), gomock.Any(), "zos.deployment.changes", gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, nodeTwinID uint32, fn string, data, result interface{}) error {
				var res *[]gridtypes.Workload = result.(*[]gridtypes.Workload)
				switch nodeTwinID {
				case 13:
					*res = dl1.Workloads
				case 23:
					*res = dl2.Workloads
				default:
					return fmt.Errorf("unexpected rmb call with node id %d", nodeID)
				}

				return nil
			}).AnyTimes()

		contracts, err := deployer.Deploy(context.Background(), nil, newDls, newDlsSolProvider)
		assert.NoError(t, err)
		assert.Equal(t, contracts, map[uint32]uint64{10: 100, 20: 200})
	})

	t.Run("test update", func(t *testing.T) {
		cl := mocks.NewRMBMockClient(ctrl)
		sub := mocks.NewMockSubstrateExt(ctrl)
		ncPool := mocks.NewMockNodeClientGetter(ctrl)

		tfPluginClient.SubstrateConn = sub
		tfPluginClient.NcPool = ncPool
		tfPluginClient.RMB = cl

		deployer := NewDeployer(
			tfPluginClient,
			true,
		)

		oldDl, err := deploymentWithNameGateway(identity, twinID, true, 0, backendURLWithTLSPassthrough)
		assert.NoError(t, err)
		oldDl.ContractID = 100

		newVersionlessDl, err := deploymentWithNameGateway(identity, twinID, true, 0, "2.2.2.2:10")
		assert.NoError(t, err)
		newVersionlessDl.ContractID = 100

		versionedDl, err := deploymentWithNameGateway(identity, twinID, true, 1, "2.2.2.2:10")
		assert.NoError(t, err)
		versionedDl.SignatureRequirement = newVersionlessDl.SignatureRequirement
		versionedDl.ContractID = 100

		newDls := map[uint32]gridtypes.Deployment{
			10: newVersionlessDl,
		}

		newDlsSolProvider := map[uint32]*uint64{
			10: nil,
		}

		mockDeployerValidator(&deployer, ctrl, []uint32{10})
		sub.EXPECT().GetContract(uint64(100)).Return(subi.Contract{
			Contract: &substrate.Contract{ContractType: substrate.ContractType{
				NodeContract: substrate.NodeContract{
					PublicIPsCount: 0,
				},
			}},
		}, nil)

		ncPool.EXPECT().
			GetNodeClient(sub, uint32(10)).
			Return(client.NewNodeClient(13, cl, tfPluginClient.RMBTimeout), nil).AnyTimes()

		cl.EXPECT().
			Call(gomock.Any(), uint32(13), "zos.deployment.get", gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, twin uint32, fn string, data, result interface{}) error {
				var res *gridtypes.Deployment = result.(*gridtypes.Deployment)
				*res = oldDl
				return nil
			}).AnyTimes()

		newDlHash, err := hash(&versionedDl)
		assert.NoError(t, err)

		sub.EXPECT().
			UpdateNodeContract(
				identity,
				uint64(100),
				"",
				newDlHash,
			).Return(uint64(100), nil)

		cl.EXPECT().
			Call(gomock.Any(), uint32(13), "zos.deployment.update", versionedDl, gomock.Any()).
			DoAndReturn(func(ctx context.Context, twin uint32, fn string, data, result interface{}) error {
				versionedDl.Workloads[0].Result.State = gridtypes.StateOk
				versionedDl.Workloads[0].Result.Data, _ = json.Marshal(zos.GatewayProxyResult{})
				return nil
			})

		cl.EXPECT().
			Call(gomock.Any(), uint32(13), "zos.deployment.changes", gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, twin uint32, fn string, data, result interface{}) error {
				var res *[]gridtypes.Workload = result.(*[]gridtypes.Workload)
				*res = versionedDl.Workloads
				return nil
			}).AnyTimes()

		contracts, err := deployer.Deploy(context.Background(), map[uint32]uint64{10: 100}, newDls, newDlsSolProvider)

		assert.NoError(t, err)
		assert.Equal(t, contracts, map[uint32]uint64{10: 100})
	})

	t.Run("test cancel", func(t *testing.T) {
		cl := mocks.NewRMBMockClient(ctrl)
		sub := mocks.NewMockSubstrateExt(ctrl)
		ncPool := mocks.NewMockNodeClientGetter(ctrl)

		tfPluginClient.SubstrateConn = sub
		tfPluginClient.NcPool = ncPool
		tfPluginClient.RMB = cl

		deployer := NewDeployer(
			tfPluginClient,
			true,
		)

		dl1, err := deploymentWithNameGateway(identity, twinID, true, 0, backendURLWithTLSPassthrough)
		assert.NoError(t, err)

		dl1.ContractID = 100

		sub.EXPECT().
			EnsureContractCanceled(
				identity,
				uint64(100),
			).Return(nil)

		ncPool.EXPECT().
			GetNodeClient(sub, uint32(10)).
			Return(client.NewNodeClient(13, cl, tfPluginClient.RMBTimeout), nil).AnyTimes()

		err = deployer.Cancel(context.Background(), 100)
		assert.NoError(t, err)
	})

	t.Run("test cocktail", func(t *testing.T) {
		cl := mocks.NewRMBMockClient(ctrl)
		sub := mocks.NewMockSubstrateExt(ctrl)
		ncPool := mocks.NewMockNodeClientGetter(ctrl)

		tfPluginClient.SubstrateConn = sub
		tfPluginClient.NcPool = ncPool
		tfPluginClient.RMB = cl

		deployer := NewDeployer(
			tfPluginClient,
			true,
		)

		g := workloads.GatewayFQDNProxy{Name: "f", FQDN: "test.com", Backends: []zos.Backend{zos.Backend(backendURLWithoutTLSPassthrough)}}

		dl1, err := deploymentWithNameGateway(identity, twinID, false, 0, backendURLWithoutTLSPassthrough)
		assert.NoError(t, err)
		dl2, err := deploymentWithNameGateway(identity, twinID, false, 0, backendURLWithoutTLSPassthrough)
		assert.NoError(t, err)
		dl3, err := deploymentWithNameGateway(identity, twinID, true, 1, backendURLWithTLSPassthrough)
		assert.NoError(t, err)
		dl4, err := deploymentWithNameGateway(identity, twinID, false, 0, backendURLWithoutTLSPassthrough)
		assert.NoError(t, err)
		dl5, err := deploymentWithNameGateway(identity, twinID, true, 0, backendURLWithTLSPassthrough)
		assert.NoError(t, err)
		dl6, err := deploymentWithNameGateway(identity, twinID, true, 1, backendURLWithTLSPassthrough)
		assert.NoError(t, err)

		dl2.Workloads = append(dl2.Workloads, g.ZosWorkload())
		dl3.Workloads = append(dl3.Workloads, g.ZosWorkload())
		assert.NoError(t, dl2.Sign(twinID, identity))
		assert.NoError(t, dl3.Sign(twinID, identity))

		dl1.ContractID = 100
		dl2.ContractID = 200
		dl3.ContractID = 200
		dl4.ContractID = 300
		dl5.ContractID = 400
		dl6.ContractID = 400

		dl3Hash, err := hash(&dl3)
		assert.NoError(t, err)
		dl4Hash, err := hash(&dl4)
		assert.NoError(t, err)
		dl6Hash, err := hash(&dl6)
		assert.NoError(t, err)

		oldDls := map[uint32]uint64{
			10: 100,
			20: 200,
			40: 400,
		}
		newDls := map[uint32]gridtypes.Deployment{
			20: dl3,
			30: dl4,
			40: dl6,
		}

		newDlsSolProvider := map[uint32]*uint64{
			20: nil,
			30: nil,
			40: nil,
		}

		mockDeployerValidator(&deployer, ctrl, []uint32{10, 20, 40, 30})
		sub.EXPECT().GetContract(gomock.Any()).Return(subi.Contract{
			Contract: &substrate.Contract{ContractType: substrate.ContractType{
				NodeContract: substrate.NodeContract{
					PublicIPsCount: 0,
				},
			}},
		}, nil).AnyTimes()

		contractsData := []substrate.BatchCreateContractData{
			{
				Node:               30,
				Body:               "",
				Hash:               dl4Hash,
				PublicIPs:          0,
				SolutionProviderID: nil,
			},
		}

		sub.EXPECT().BatchCreateContract(identity, contractsData).Return([]uint64{300}, nil, nil)

		sub.EXPECT().
			UpdateNodeContract(
				identity,
				gomock.Any(),
				"",
				gomock.Any(),
			).DoAndReturn(func(identity substrate.Identity, contractID uint64, body string, hash string) (uint64, error) {
			if !slices.Contains([]string{dl3Hash, dl6Hash}, hash) {
				return 0, fmt.Errorf("unexpected call to UpdateNodeContract with hash %s", hash)
			}

			return contractID, nil
		}).Times(2)

		sub.EXPECT().BatchCancelContract(identity, []uint64{100}).Return(nil)

		ncPool.EXPECT().GetNodeClient(sub, gomock.Any()).DoAndReturn(func(sub subi.SubstrateExt, nodeID uint32) (*client.NodeClient, error) {
			switch nodeID {
			case 10:
				return client.NewNodeClient(13, cl, tfPluginClient.RMBTimeout), nil
			case 20:
				return client.NewNodeClient(23, cl, tfPluginClient.RMBTimeout), nil
			case 30:
				return client.NewNodeClient(33, cl, tfPluginClient.RMBTimeout), nil
			case 40:
				return client.NewNodeClient(43, cl, tfPluginClient.RMBTimeout), nil
			default:
				return nil, fmt.Errorf("unexpected call to GetNodeClient with node id %d", nodeID)
			}
		}).AnyTimes()

		cl.EXPECT().
			Call(gomock.Any(), gomock.Any(), "zos.deployment.changes", gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, nodeTwinID uint32, fn string, data, result interface{}) error {
				var res *[]gridtypes.Workload = result.(*[]gridtypes.Workload)
				switch nodeTwinID {
				case 23:
					*res = dl2.Workloads
				case 33:
					*res = dl4.Workloads
				case 43:
					*res = dl5.Workloads
				default:
					return fmt.Errorf("unexpected rmb call to node with twin %d", nodeTwinID)
				}

				return nil
			}).Times(3) // for each wait call

		cl.EXPECT().
			Call(gomock.Any(), gomock.Any(), "zos.deployment.get", gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, nodeTwinID uint32, fn string, data, result interface{}) error {
				var res *gridtypes.Deployment = result.(*gridtypes.Deployment)
				switch nodeTwinID {
				case 13:
					*res = dl1
				case 23:
					*res = dl2
				case 33:
					*res = dl4
				case 43:
					*res = dl5
				default:
					return fmt.Errorf("unexpected rmb call to node with twin %d", nodeTwinID)
				}

				return nil
			}).Times(5) // 3 calls for getting old deployments, 2 calls for getting hashes of to-be-updated deployments

		cl.EXPECT().
			Call(gomock.Any(), gomock.Any(), "zos.deployment.update", gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, nodeTwinID uint32, fn string, data, result interface{}) error {
				switch nodeTwinID {
				case 23:
					dl2.Workloads = dl3.Workloads
					dl2.Version = 1
					dl2.Workloads[0].Version = 1
					dl2.Workloads[0].Result.State = gridtypes.StateOk
					dl2.Workloads[1].Result.State = gridtypes.StateOk
				case 43:
					dl5.Workloads = dl6.Workloads
					dl5.Version = 1
					dl5.Workloads[0].Version = 1
					dl5.Workloads[0].Result.State = gridtypes.StateOk
				default:
					return fmt.Errorf("undexpected rmb call with node twin id %d", nodeTwinID)
				}

				return nil
			}).Times(2)

		cl.EXPECT().
			Call(gomock.Any(), uint32(33), "zos.deployment.deploy", gomock.Any(), gomock.Any()).
			DoAndReturn(func(ctx context.Context, twin uint32, fn string, data, result interface{}) error {
				dl4.Workloads[0].Result.State = gridtypes.StateOk
				return nil
			})

		contracts, err := deployer.Deploy(context.Background(), oldDls, newDls, newDlsSolProvider)
		assert.NoError(t, err)
		assert.Equal(t, contracts, map[uint32]uint64{
			20: 200,
			30: 300,
			40: 400,
		})
	})
}
