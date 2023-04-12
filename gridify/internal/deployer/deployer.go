// Package deployer for project deployment
package deployer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/workloads"
	"github.com/threefoldtech/tfgrid-sdk-go/gridify/internal/tfplugin"
)

// Deployer struct manages project deployment
type Deployer struct {
	tfPluginClient tfplugin.TFPluginClientInterface

	repoURL     string
	projectName string

	logger zerolog.Logger
}

// NewDeployer return new project deployer
func NewDeployer(tfPluginClient tfplugin.TFPluginClientInterface, repoURL string, logger zerolog.Logger) (Deployer, error) {

	deployer := Deployer{
		tfPluginClient: tfPluginClient,
		logger:         logger,
		repoURL:        repoURL,
	}

	projectName, err := deployer.getProjectName()
	if err != nil {
		return Deployer{}, err
	}
	deployer.projectName = projectName

	return deployer, nil
}

// Deploy deploys a project and map each port to a domain
func (d *Deployer) Deploy(ctx context.Context, ports []uint) (map[uint]string, error) {

	contracts, err := d.tfPluginClient.ListContractsOfProjectName(d.projectName)
	if err != nil {
		return map[uint]string{}, errors.Wrapf(err, "could not check existing contracts for project %s", d.projectName)
	}
	if len(contracts.NameContracts) != 0 || len(contracts.NodeContracts) != 0 {
		return map[uint]string{}, fmt.Errorf(
			"project %s already deployed please destroy project deployment first using gridify destroy",
			d.projectName,
		)
	}

	d.logger.Debug().Msg("getting nodes with free resources")

	node, err := findNode(d.tfPluginClient)
	if err != nil {
		return map[uint]string{}, errors.Wrapf(
			err,
			"failed to get a node with enough resources on network %s",
			d.tfPluginClient.GetGridNetwork(),
		)
	}

	d.logger.Info().Msg("deploying a network")
	network := buildNetwork(d.projectName, node)
	err = d.tfPluginClient.DeployNetwork(ctx, &network)
	if err != nil {
		return map[uint]string{}, errors.Wrapf(err, "could not deploy network %s on node %d", network.Name, node)
	}

	d.logger.Info().Msg("deploying a vm")
	dl := buildDeployment(network.Name, d.projectName, d.repoURL, node)
	err = d.tfPluginClient.DeployDeployment(ctx, &dl)
	if err != nil {
		return map[uint]string{}, errors.Wrapf(err, "could not deploy vm %s on node %d", dl.Name, node)
	}

	resVM, err := d.tfPluginClient.LoadVMFromGrid(node, dl.Name, dl.Name)
	if err != nil {
		return map[uint]string{}, errors.Wrapf(err, "could not load vm %s on node %d", dl.Name, node)
	}

	portlessBackend := buildPortlessBackend(resVM.ComputedIP)

	FQDNs := make(map[uint]string)
	// TODO: deploy each gateway in a separate goroutine
	for _, port := range ports {
		backend := fmt.Sprintf("%s:%d", portlessBackend, port)
		d.logger.Info().Msgf("deploying a gateway for port %d", port)
		gateway := buildGateway(backend, d.projectName, node)
		err := d.tfPluginClient.DeployGatewayName(ctx, &gateway)
		if err != nil {
			return map[uint]string{}, errors.Wrapf(err, "could not deploy gateway %s on node %d", gateway.Name, node)
		}
		resGateway, err := d.tfPluginClient.LoadGatewayNameFromGrid(node, gateway.Name, gateway.Name)
		if err != nil {
			return map[uint]string{}, errors.Wrapf(err, "could not load gateway %s on node %d", gateway.Name, node)
		}
		FQDNs[port] = resGateway.FQDN
	}

	d.logger.Info().Msg("Project Deployed!")

	return FQDNs, nil
}

// Destroy destroys all the contracts of a project
func (d *Deployer) Destroy() error {
	d.logger.Info().Msgf("canceling contracts for project %s", d.projectName)
	contracts, err := d.tfPluginClient.ListContractsOfProjectName(d.projectName)
	if err != nil {
		return errors.Wrapf(err, "could not load contracts for project %s", d.projectName)
	}
	contractsSlice := append(contracts.NameContracts, contracts.NodeContracts...)
	for _, contract := range contractsSlice {
		contractID, err := strconv.ParseUint(contract.ContractID, 0, 64)
		if err != nil {
			return errors.Wrapf(err, "could not parse contract %s into uint64", contract.ContractID)
		}
		d.logger.Debug().Msgf("canceling contract %d", contractID)

		err = d.tfPluginClient.CancelContract(contractID)
		if err != nil {
			return errors.Wrapf(err, "could not cancel contract %d", contractID)
		}
	}
	d.logger.Info().Msg("Project Destroyed!")
	return nil
}

// Get returns deployed project domains
func (d *Deployer) Get() (map[string]string, error) {
	d.logger.Info().Msgf("getting contracts for project %s", d.projectName)
	contracts, err := d.tfPluginClient.ListContractsOfProjectName(d.projectName)
	if err != nil {
		return map[string]string{}, errors.Wrapf(err, "could not load contracts for project %s", d.projectName)
	}
	fqdns := make(map[string]string)
	for _, contract := range contracts.NodeContracts {
		contractID, err := strconv.ParseUint(contract.ContractID, 0, 64)
		if err != nil {
			return map[string]string{}, errors.Wrapf(err, "could not parse contract %s into uint64", contract.ContractID)
		}
		var deploymentData workloads.DeploymentData
		err = json.Unmarshal([]byte(contract.DeploymentData), &deploymentData)
		if err != nil {
			return map[string]string{}, errors.Wrapf(err, "failed to unmarshal deployment data %s", contract.DeploymentData)
		}
		if deploymentData.Type != "Gateway Name" {
			continue
		}
		dl, err := d.tfPluginClient.GetDeployment(contract.NodeID, contractID)
		if err != nil {
			return map[string]string{}, err
		}
		if len(dl.Workloads) == 0 {
			d.logger.Debug().Msgf("no workloads found in deployment with contract %d", contractID)
			continue
		}
		gateway, err := workloads.NewGatewayNameProxyFromZosWorkload(dl.Workloads[0])
		if err != nil {
			return map[string]string{}, err
		}
		if len(gateway.Backends) == 0 {
			d.logger.Debug().Msgf("no backends found in gateway %s", gateway.Name)
			continue
		}
		u, err := url.Parse(string(gateway.Backends[0]))
		if err != nil {
			return map[string]string{}, errors.Wrapf(err, "failed parsing the domain %s", gateway.FQDN)
		}
		fqdns[u.Port()] = gateway.FQDN
	}
	return fqdns, nil
}

func (d *Deployer) getProjectName() (string, error) {

	splitURL := strings.Split(string(d.repoURL), "/")
	projectName, _, found := strings.Cut(splitURL[len(splitURL)-1], ".git")
	if !found {
		return "", fmt.Errorf("couldn't get project name")
	}
	return projectName, nil
}
