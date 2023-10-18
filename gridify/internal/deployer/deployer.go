// Package deployer for project deployment
package deployer

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
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
func (d *Deployer) Deploy(ctx context.Context, vmSpec VMSpec, ports []uint, generator rand.Rand) (map[uint]string, error) {

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

	node, err := d.tfPluginClient.GetAvailableNode(ctx, buildNodeFilter(vmSpec), uint64(vmSpec.Storage))
	if err != nil {
		return map[uint]string{}, errors.Wrapf(
			err,
			"failed to get a node with enough resources on network %s",
			d.tfPluginClient.GetGridNetwork(),
		)
	}

	network := buildNetwork(d.projectName, node, generator)
	dl := buildDeployment(vmSpec, network.Name, d.projectName, d.repoURL, node, generator)

	d.logger.Info().Msg("deploying a network")
	err = d.tfPluginClient.DeployNetwork(ctx, &network)
	if err != nil {
		return map[uint]string{}, errors.Wrapf(err, "could not deploy network %s on node %d", network.Name, node)
	}

	d.logger.Info().Msg("deploying a vm")
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
		gateway := buildGateway(backend, d.projectName, node, generator)
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

	d.logger.Info().Msg("project deployed")

	return FQDNs, nil
}

// Destroy destroys all the contracts of a project
func (d *Deployer) Destroy() error {
	return d.tfPluginClient.CancelByProjectName(d.projectName)
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
		var deploymentData workloads.DeploymentData
		err = json.Unmarshal([]byte(contract.DeploymentData), &deploymentData)
		if err != nil {
			return map[string]string{}, errors.Wrapf(err, "failed to unmarshal deployment data %s", contract.DeploymentData)
		}
		if deploymentData.Type != "Gateway Name" {
			continue
		}
		contractID, err := strconv.ParseUint(contract.ContractID, 0, 64)
		if err != nil {
			return map[string]string{}, errors.Wrapf(err, "could not parse contract %s into uint64", contract.ContractID)
		}
		d.tfPluginClient.SetState(contract.NodeID, []uint64{contractID})
		gateway, err := d.tfPluginClient.LoadGatewayNameFromGrid(contract.NodeID, deploymentData.Name, deploymentData.Name)
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
