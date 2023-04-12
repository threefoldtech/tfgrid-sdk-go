package deployer

import (
	"context"
	"errors"
	"math/rand"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/threefoldtech/grid_proxy_server/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/grid3-go/graphql"
	"github.com/threefoldtech/tfgrid-sdk-go/grid3-go/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/gridify/internal/mocks"
	"github.com/threefoldtech/zos/pkg/gridtypes"
)

func TestDeploy(t *testing.T) {
	rand.Seed(1)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoURL := "https://github.com/threefoldtech/tfgrid-sdk-go/gridify.git"
	projectName := "gridify"
	filter := buildNodeFilter()
	network := buildNetwork(projectName, 1)
	deployment := buildDeployment(network.Name, projectName, repoURL, 1)
	vmIP := "10.10.10.10/24"
	gateway1 := buildGateway("http://10.10.10.10:80", projectName, 1)
	gateway2 := buildGateway("http://10.10.10.10:8080", projectName, 1)

	clientMock := mocks.NewMockTFPluginClientInterface(ctrl)

	deployer, err := NewDeployer(clientMock, repoURL, log.Logger.Level(zerolog.Disabled))
	assert.NoError(t, err)

	t.Run("error listing contracts of a project", func(t *testing.T) {
		clientMock.
			EXPECT().
			ListContractsOfProjectName(projectName).
			Return(graphql.Contracts{}, errors.New("error"))

		_, err := deployer.Deploy(context.Background(), []uint{80})
		assert.Error(t, err)
	})
	t.Run("deployment for same project already exists", func(t *testing.T) {
		clientMock.
			EXPECT().
			ListContractsOfProjectName(projectName).
			Return(graphql.Contracts{NameContracts: []graphql.Contract{{ContractID: "10"}}}, nil)

		_, err := deployer.Deploy(context.Background(), []uint{80})
		assert.Error(t, err)
	})
	t.Run("no nodes available", func(t *testing.T) {
		clientMock.
			EXPECT().
			ListContractsOfProjectName(projectName).
			Return(graphql.Contracts{}, nil)

		clientMock.
			EXPECT().
			FilterNodes(filter, gomock.Any()).
			Return([]types.Node{}, 0, nil)

		clientMock.
			EXPECT().
			GetGridNetwork().
			Return("dev")

		_, err := deployer.Deploy(context.Background(), []uint{80})
		assert.Error(t, err)
	})
	t.Run("error finding available nodes", func(t *testing.T) {
		clientMock.
			EXPECT().
			ListContractsOfProjectName(projectName).
			Return(graphql.Contracts{}, nil)

		clientMock.
			EXPECT().
			FilterNodes(filter, gomock.Any()).
			Return([]types.Node{}, 0, errors.New("error"))

		clientMock.
			EXPECT().
			GetGridNetwork().
			Return("dev")

		_, err := deployer.Deploy(context.Background(), []uint{80})
		assert.Error(t, err)
	})
	t.Run("network deployment failed", func(t *testing.T) {
		rand.Seed(1)
		clientMock.
			EXPECT().
			ListContractsOfProjectName(projectName).
			Return(graphql.Contracts{}, nil)

		clientMock.
			EXPECT().
			FilterNodes(filter, gomock.Any()).
			Return([]types.Node{{NodeID: 1}}, 0, nil)

		clientMock.
			EXPECT().
			DeployNetwork(gomock.Any(), &network).
			Return(errors.New("error"))

		_, err := deployer.Deploy(context.Background(), []uint{80})
		assert.Error(t, err)
	})
	t.Run("vm deployment failed", func(t *testing.T) {
		rand.Seed(1)
		clientMock.
			EXPECT().
			ListContractsOfProjectName(projectName).
			Return(graphql.Contracts{}, nil)

		clientMock.
			EXPECT().
			FilterNodes(filter, gomock.Any()).
			Return([]types.Node{{NodeID: 1}}, 0, nil)

		clientMock.
			EXPECT().
			DeployNetwork(gomock.Any(), &network).
			Return(nil)

		clientMock.
			EXPECT().
			DeployDeployment(gomock.Any(), &deployment).
			Return(errors.New("error"))

		_, err := deployer.Deploy(context.Background(), []uint{80})
		assert.Error(t, err)
	})
	t.Run("loading vm failed", func(t *testing.T) {
		rand.Seed(1)
		clientMock.
			EXPECT().
			ListContractsOfProjectName(projectName).
			Return(graphql.Contracts{}, nil)

		clientMock.
			EXPECT().
			FilterNodes(filter, gomock.Any()).
			Return([]types.Node{{NodeID: 1}}, 0, nil)

		clientMock.
			EXPECT().
			DeployNetwork(gomock.Any(), &network).
			Return(nil)

		clientMock.
			EXPECT().
			DeployDeployment(gomock.Any(), &deployment).
			Return(nil)

		clientMock.
			EXPECT().
			LoadVMFromGrid(gomock.Any(), deployment.Name, deployment.Name).
			Return(workloads.VM{}, errors.New("error"))

		_, err := deployer.Deploy(context.Background(), []uint{80})
		assert.Error(t, err)
	})
	t.Run("gateway deployment failed", func(t *testing.T) {
		rand.Seed(1)
		clientMock.
			EXPECT().
			ListContractsOfProjectName(projectName).
			Return(graphql.Contracts{}, nil)

		clientMock.
			EXPECT().
			FilterNodes(filter, gomock.Any()).
			Return([]types.Node{{NodeID: 1}}, 0, nil)

		clientMock.
			EXPECT().
			DeployNetwork(gomock.Any(), &network).
			Return(nil)

		clientMock.
			EXPECT().
			DeployDeployment(gomock.Any(), &deployment).
			Return(nil)

		clientMock.
			EXPECT().
			LoadVMFromGrid(gomock.Any(), deployment.Name, deployment.Name).
			Return(workloads.VM{ComputedIP: vmIP}, nil)

		clientMock.
			EXPECT().
			DeployGatewayName(gomock.Any(), &gateway1).
			Return(errors.New("error"))

		_, err := deployer.Deploy(context.Background(), []uint{80})
		assert.Error(t, err)
	})
	t.Run("loading gateway failed", func(t *testing.T) {
		rand.Seed(1)
		clientMock.
			EXPECT().
			ListContractsOfProjectName(projectName).
			Return(graphql.Contracts{}, nil)

		clientMock.
			EXPECT().
			FilterNodes(filter, gomock.Any()).
			Return([]types.Node{{NodeID: 1}}, 0, nil)

		clientMock.
			EXPECT().
			DeployNetwork(gomock.Any(), &network).
			Return(nil)

		clientMock.
			EXPECT().
			DeployDeployment(gomock.Any(), &deployment).
			Return(nil)

		clientMock.
			EXPECT().
			LoadVMFromGrid(gomock.Any(), deployment.Name, deployment.Name).
			Return(workloads.VM{ComputedIP: vmIP}, nil)

		clientMock.
			EXPECT().
			DeployGatewayName(gomock.Any(), &gateway1).
			Return(nil)

		clientMock.
			EXPECT().
			LoadGatewayNameFromGrid(gomock.Any(), gateway1.Name, gateway1.Name).
			Return(workloads.GatewayNameProxy{}, errors.New("error"))

		_, err := deployer.Deploy(context.Background(), []uint{80})
		assert.Error(t, err)
	})
	t.Run("deploying using one port", func(t *testing.T) {
		rand.Seed(1)
		clientMock.
			EXPECT().
			ListContractsOfProjectName(projectName).
			Return(graphql.Contracts{}, nil)

		clientMock.
			EXPECT().
			FilterNodes(filter, gomock.Any()).
			Return([]types.Node{{NodeID: 1}}, 0, nil)

		clientMock.
			EXPECT().
			DeployNetwork(gomock.Any(), &network).
			Return(nil)

		clientMock.
			EXPECT().
			DeployDeployment(gomock.Any(), &deployment).
			Return(nil)

		clientMock.
			EXPECT().
			LoadVMFromGrid(gomock.Any(), deployment.Name, deployment.Name).
			Return(workloads.VM{ComputedIP: vmIP}, nil)

		clientMock.
			EXPECT().
			DeployGatewayName(gomock.Any(), &gateway1).
			Return(nil)

		clientMock.
			EXPECT().
			LoadGatewayNameFromGrid(gomock.Any(), gateway1.Name, gateway1.Name).
			Return(workloads.GatewayNameProxy{FQDN: "domain1"}, nil)

		fqdns, err := deployer.Deploy(context.Background(), []uint{80})
		assert.NoError(t, err)
		assert.Equal(t, fqdns, map[uint]string{80: "domain1"})
	})
	t.Run("deploying using multiple ports", func(t *testing.T) {
		rand.Seed(1)
		clientMock.
			EXPECT().
			ListContractsOfProjectName(projectName).
			Return(graphql.Contracts{}, nil)

		clientMock.
			EXPECT().
			FilterNodes(filter, gomock.Any()).
			Return([]types.Node{{NodeID: 1}}, 0, nil)

		clientMock.
			EXPECT().
			DeployNetwork(gomock.Any(), &network).
			Return(nil)

		clientMock.
			EXPECT().
			DeployDeployment(gomock.Any(), &deployment).
			Return(nil)

		clientMock.
			EXPECT().
			LoadVMFromGrid(gomock.Any(), deployment.Name, deployment.Name).
			Return(workloads.VM{ComputedIP: vmIP}, nil)

		clientMock.
			EXPECT().
			DeployGatewayName(gomock.Any(), &gateway1).
			Return(nil)

		clientMock.
			EXPECT().
			LoadGatewayNameFromGrid(gomock.Any(), gateway1.Name, gateway1.Name).
			Return(workloads.GatewayNameProxy{FQDN: "domain1"}, nil)

		clientMock.
			EXPECT().
			DeployGatewayName(gomock.Any(), &gateway2).
			Return(nil)

		clientMock.
			EXPECT().
			LoadGatewayNameFromGrid(gomock.Any(), gateway2.Name, gateway2.Name).
			Return(workloads.GatewayNameProxy{FQDN: "domain2"}, nil)

		fqdns, err := deployer.Deploy(context.Background(), []uint{80, 8080})
		assert.NoError(t, err)

		assert.Equal(t, fqdns, map[uint]string{80: "domain1", 8080: "domain2"})
	})

}

func TestDestroy(t *testing.T) {

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoURL := "https://github.com/threefoldtech/tfgrid-sdk-go/gridify.git"
	projectName := "gridify"
	contracts := graphql.Contracts{
		NameContracts: []graphql.Contract{{ContractID: "10"}, {ContractID: "11"}},
		NodeContracts: []graphql.Contract{{ContractID: "20"}, {ContractID: "21"}},
	}

	clientMock := mocks.NewMockTFPluginClientInterface(ctrl)

	deployer, err := NewDeployer(clientMock, repoURL, log.Logger.Level(zerolog.Disabled))
	assert.NoError(t, err)

	t.Run("loading contracts failed", func(t *testing.T) {
		clientMock.
			EXPECT().
			ListContractsOfProjectName(projectName).
			Return(graphql.Contracts{}, errors.New("error"))

		err := deployer.Destroy()
		assert.Error(t, err)
	})
	t.Run("no contracts found", func(t *testing.T) {
		clientMock.
			EXPECT().
			ListContractsOfProjectName(projectName).
			Return(graphql.Contracts{}, nil)

		err := deployer.Destroy()
		assert.NoError(t, err)
	})
	t.Run("contracts with non-numeric ids", func(t *testing.T) {
		clientMock.
			EXPECT().
			ListContractsOfProjectName(projectName).
			Return(graphql.Contracts{NameContracts: []graphql.Contract{{ContractID: "id"}}}, nil)

		err := deployer.Destroy()
		assert.Error(t, err)
	})
	t.Run("canceling contracts failed", func(t *testing.T) {
		clientMock.
			EXPECT().
			ListContractsOfProjectName(projectName).
			Return(contracts, nil)

		clientMock.
			EXPECT().
			CancelContract(uint64(10)).
			Return(errors.New("error"))
		err := deployer.Destroy()
		assert.Error(t, err)
	})
	t.Run("canceling contracts succeeded", func(t *testing.T) {
		clientMock.
			EXPECT().
			ListContractsOfProjectName(projectName).
			Return(contracts, nil)

		clientMock.
			EXPECT().
			CancelContract(uint64(10)).
			Return(nil)

		clientMock.
			EXPECT().
			CancelContract(uint64(11)).
			Return(nil)

		clientMock.
			EXPECT().
			CancelContract(uint64(20)).
			Return(nil)

		clientMock.
			EXPECT().
			CancelContract(uint64(21)).
			Return(nil)

		err := deployer.Destroy()
		assert.NoError(t, err)
	})
}

func TestGet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repoURL := "https://github.com/threefoldtech/tfgrid-sdk-go/gridify.git"
	projectName := "gridify"
	contracts := graphql.Contracts{
		NameContracts: []graphql.Contract{{ContractID: "10"}, {ContractID: "11"}},
		NodeContracts: []graphql.Contract{{ContractID: "20", NodeID: 14, DeploymentData: "{\"type\":\"Gateway Name\"}"}, {ContractID: "21", NodeID: 14, DeploymentData: "{}"}},
	}

	gatewayWorkload := gridtypes.Workload{
		Type: gridtypes.WorkloadType("gateway-name-proxy"),
		Data: []byte(`{"backends":["http://10.10.10.10:8080"]}`),
		Result: gridtypes.Result{
			Data: []byte(`{"fqdn":"example.com"}`),
		},
	}
	badBackendGateway := gridtypes.Workload{
		Type: gridtypes.WorkloadType("gateway-name-proxy"),
		Data: []byte(`{"backends":["\t"]}`),
		Result: gridtypes.Result{
			Data: []byte(`{"fqdn":"example.com"}`),
		},
	}

	clientMock := mocks.NewMockTFPluginClientInterface(ctrl)

	deployer, err := NewDeployer(clientMock, repoURL, log.Logger.Level(zerolog.Disabled))
	assert.NoError(t, err)
	t.Run("loading contracts failed", func(t *testing.T) {
		clientMock.
			EXPECT().
			ListContractsOfProjectName(projectName).
			Return(graphql.Contracts{}, errors.New("error"))

		_, err := deployer.Get()
		assert.Error(t, err)
	})
	t.Run("no contracts found", func(t *testing.T) {
		clientMock.
			EXPECT().
			ListContractsOfProjectName(projectName).
			Return(graphql.Contracts{}, nil)

		_, err := deployer.Get()
		assert.NoError(t, err)
	})
	t.Run("contracts with non-numeric ids", func(t *testing.T) {
		clientMock.
			EXPECT().
			ListContractsOfProjectName(projectName).
			Return(graphql.Contracts{NodeContracts: []graphql.Contract{{ContractID: "id"}}}, nil)

		_, err := deployer.Get()
		assert.Error(t, err)
	})
	t.Run("parsing deployment data failed", func(t *testing.T) {
		clientMock.
			EXPECT().
			ListContractsOfProjectName(projectName).
			Return(graphql.Contracts{NodeContracts: []graphql.Contract{{ContractID: "1"}}}, nil)

		_, err := deployer.Get()
		assert.Error(t, err)
	})
	t.Run("loading deployment failed", func(t *testing.T) {
		clientMock.
			EXPECT().
			ListContractsOfProjectName(projectName).
			Return(contracts, nil)

		clientMock.
			EXPECT().
			GetDeployment(uint32(14), uint64(20)).
			Return(gridtypes.Deployment{}, errors.New("error"))

		_, err := deployer.Get()
		assert.Error(t, err)
	})
	t.Run("generate gateway from workload failed", func(t *testing.T) {
		clientMock.
			EXPECT().
			ListContractsOfProjectName(projectName).
			Return(contracts, nil)

		clientMock.
			EXPECT().
			GetDeployment(uint32(14), uint64(20)).
			Return(gridtypes.Deployment{Workloads: []gridtypes.Workload{{}}}, nil)

		_, err := deployer.Get()
		assert.Error(t, err)
	})
	t.Run("parsing backend failed", func(t *testing.T) {
		clientMock.
			EXPECT().
			ListContractsOfProjectName(projectName).
			Return(contracts, nil)

		clientMock.
			EXPECT().
			GetDeployment(uint32(14), uint64(20)).
			Return(gridtypes.Deployment{Workloads: []gridtypes.Workload{badBackendGateway}}, nil)

		_, err := deployer.Get()
		assert.Error(t, err)
	})
	t.Run("fetching succeeded", func(t *testing.T) {
		clientMock.
			EXPECT().
			ListContractsOfProjectName(projectName).
			Return(contracts, nil)

		clientMock.
			EXPECT().
			GetDeployment(uint32(14), uint64(20)).
			Return(gridtypes.Deployment{Workloads: []gridtypes.Workload{gatewayWorkload}}, nil)

		fqdns, err := deployer.Get()
		assert.Equal(t, fqdns, map[string]string{"8080": "example.com"})
		assert.NoError(t, err)
	})

}
