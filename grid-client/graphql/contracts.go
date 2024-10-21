// Package graphql for grid graphql support
package graphql

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	client "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/subi"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
)

// ContractsGetter for contracts getter from graphql
type ContractsGetter struct {
	twinID        uint32
	graphql       GraphQl
	substrateConn subi.SubstrateExt
	ncPool        client.NodeClientGetter
}

// Contracts from graphql
type Contracts struct {
	NameContracts []Contract `json:"nameContracts"`
	NodeContracts []Contract `json:"nodeContracts"`
	RentContracts []Contract `json:"rentContracts"`
}

// Contract from graphql
type Contract struct {
	ContractID     string `json:"contractID"`
	State          string `json:"state"`
	DeploymentData string `json:"deploymentData"`

	// for node and rent contracts
	NodeID uint32 `json:"nodeID"`
	// for name contracts
	Name string `json:"name"`
}

var ErrorContractsNotFound = fmt.Errorf("could not find any contracts")

// NewContractsGetter return a new Getter for contracts
func NewContractsGetter(twinID uint32, graphql GraphQl, substrateConn subi.SubstrateExt, ncPool client.NodeClientGetter) ContractsGetter {
	return ContractsGetter{
		twinID:        twinID,
		graphql:       graphql,
		substrateConn: substrateConn,
		ncPool:        ncPool,
	}
}

// ListContractsByTwinID returns contracts for a twinID
func (c *ContractsGetter) ListContractsByTwinID(states []string) (Contracts, error) {
	state := fmt.Sprintf(`[%v]`, strings.Join(states, ", "))
	options := fmt.Sprintf(`(where: {twinID_eq: %v, state_in: %v}, orderBy: twinID_ASC)`, c.twinID, state)

	nameContractsCount, err := c.graphql.GetItemTotalCount("nameContracts", options)
	if err != nil {
		return Contracts{}, err
	}

	nodeContractsCount, err := c.graphql.GetItemTotalCount("nodeContracts", options)
	if err != nil {
		return Contracts{}, err
	}

	rentContractsCount, err := c.graphql.GetItemTotalCount("rentContracts", options)
	if err != nil {
		return Contracts{}, err
	}

	contractsData, err := c.graphql.Query(fmt.Sprintf(`query getContracts($nameContractsCount: Int!, $nodeContractsCount: Int!, $rentContractsCount: Int!){
            nameContracts(where: {twinID_eq: %v, state_in: %v}, limit: $nameContractsCount) {
              contractID
              state
              name
            }
            nodeContracts(where: {twinID_eq: %v, state_in: %v}, limit: $nodeContractsCount) {
              contractID
              deploymentData
              state
              nodeID
            }
            rentContracts(where: {twinID_eq: %v, state_in: %v}, limit: $rentContractsCount) {
              contractID
              state
              nodeID
            }
          }`, c.twinID, state, c.twinID, state, c.twinID, state),
		map[string]interface{}{
			"nodeContractsCount": nodeContractsCount,
			"nameContractsCount": nameContractsCount,
			"rentContractsCount": rentContractsCount,
		})
	if err != nil {
		return Contracts{}, err
	}

	contractsJSONData, err := json.Marshal(contractsData)
	if err != nil {
		return Contracts{}, err
	}

	var listContracts Contracts
	err = json.Unmarshal(contractsJSONData, &listContracts)
	if err != nil {
		return Contracts{}, err
	}

	return listContracts, nil
}

// ListContractsOfProjectName returns contracts for a project name
func (c *ContractsGetter) ListContractsOfProjectName(projectName string, noGateways ...bool) (Contracts, error) {
	contracts := Contracts{
		NodeContracts: make([]Contract, 0),
		NameContracts: make([]Contract, 0),
	}
	contractsList, err := c.ListContractsByTwinID([]string{"Created, GracePeriod"})
	if err != nil {
		return Contracts{}, err
	}

	for _, contract := range contractsList.NodeContracts {
		deploymentData, err := workloads.ParseDeploymentData(contract.DeploymentData)
		if err != nil {
			log.Warn().Err(err).Str("metadata", contract.DeploymentData).Str("id", contract.ContractID).Msg("got contract with invalid metadata")
			continue
		}

		if deploymentData.ProjectName == projectName {
			contracts.NodeContracts = append(contracts.NodeContracts, contract)
		}
	}

	if len(noGateways) > 0 && noGateways[0] {
		return contracts, nil
	}

	nameGatewaysWorkloads, err := c.filterNameGatewaysWithinNodeContracts(contracts.NodeContracts)
	if err != nil {
		return Contracts{}, err
	}

	contracts.NameContracts = c.filterNameContracts(contractsList.NameContracts, nameGatewaysWorkloads)
	return contracts, nil
}

// GetNodeContractsByTypeAndName list node contracts for a given type, project name and deployment name
func (c *ContractsGetter) GetNodeContractsByTypeAndName(projectName, deploymentType, deploymentName string) (map[uint32]uint64, error) {
	contracts, err := c.ListContractsOfProjectName(projectName)
	if err != nil {
		return map[uint32]uint64{}, err
	}
	nodeContractIDs := make(map[uint32]uint64)
	for _, contract := range contracts.NodeContracts {
		deploymentData, err := workloads.ParseDeploymentData(contract.DeploymentData)
		if err != nil {
			log.Warn().Err(err).Str("metadata", contract.DeploymentData).Str("id", contract.ContractID).Msg("got contract with invalid metadata")
			continue
		}
		if deploymentData.Type != deploymentType || deploymentData.Name != deploymentName {
			continue
		}
		contractID, err := strconv.ParseUint(contract.ContractID, 0, 64)
		if err != nil {
			return map[uint32]uint64{}, err
		}
		nodeContractIDs[contract.NodeID] = contractID
		// only k8s and network have multiple contracts
		if deploymentType == workloads.VMType ||
			deploymentType == workloads.GatewayFQDNType ||
			deploymentType == workloads.GatewayNameType {
			break
		}
	}
	if len(nodeContractIDs) == 0 {
		return map[uint32]uint64{}, errors.Wrapf(ErrorContractsNotFound, "no %s with name %s found", deploymentType, deploymentName)
	}
	return nodeContractIDs, nil
}

// filterNameContracts returns the name contracts of the given name gateways
func (c *ContractsGetter) filterNameContracts(nameContracts []Contract, nameGatewayWorkloads []zos.Workload) []Contract {
	filteredNameContracts := make([]Contract, 0)
	for _, contract := range nameContracts {
		for _, w := range nameGatewayWorkloads {
			if w.Name == contract.Name {
				filteredNameContracts = append(filteredNameContracts, contract)
			}
		}
	}

	return filteredNameContracts
}

func (c *ContractsGetter) filterNameGatewaysWithinNodeContracts(nodeContracts []Contract) ([]zos.Workload, error) {
	nameGatewayWorkloads := make([]zos.Workload, 0)
	for _, contract := range nodeContracts {
		nodeClient, err := c.ncPool.GetNodeClient(c.substrateConn, contract.NodeID)
		if err != nil {
			return []zos.Workload{}, errors.Wrapf(err, "could not get node client: %d", contract.NodeID)
		}

		contractID, err := strconv.Atoi(contract.ContractID)
		if err != nil {
			return []zos.Workload{}, errors.Wrapf(err, "could not parse contract id: %s", contract.ContractID)
		}

		dl, err := nodeClient.DeploymentGet(context.Background(), uint64(contractID))
		if err != nil {
			return []zos.Workload{}, errors.Wrapf(err, "could not get deployment %d from node %d", contractID, contract.NodeID)
		}

		for _, workload := range dl.Workloads {
			if workload.Type == zos.GatewayNameProxyType {
				nameGatewayWorkloads = append(nameGatewayWorkloads, workload)
			}
		}
	}

	return nameGatewayWorkloads, nil
}
