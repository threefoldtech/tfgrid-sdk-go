package explorer

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
	httpSwagger "github.com/swaggo/http-swagger"

	// swagger configuration
	_ "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/docs"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/mw"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	rmb "github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
)

const (
	// SSDOverProvisionFactor factor by which the ssd are allowed to be overprovisioned
	SSDOverProvisionFactor = 2
)

// listFarms godoc
// @Summary Show farms on the grid
// @Description Get all farms on the grid, It has pagination
// @Tags GridProxy
// @Accept  json
// @Produce  json
// @Param page query int false "Page number"
// @Param size query int false "Max result per page"
// @Param ret_count query bool false "Set farms' count on headers based on filter"
// @Param randomize query bool false "Get random patch of farms"
// @Param sort_by query string false "Sort by specific farm filed" Enums(name, farm_id, twin_id, public_ips, dedicated)
// @Param sort_order query string false "The sorting order, default is 'asc'" Enums(desc, asc)
// @Param free_ips query int false "Min number of free ips in the farm"
// @Param total_ips query int false "Min number of total ips in the farm"
// @Param pricing_policy_id query int false "Pricing policy id"
// @Param version query int false "farm version"
// @Param farm_id query int false "farm id"
// @Param twin_id query int false "twin id associated with the farm"
// @Param name query string false "farm name"
// @Param name_contains query string false "farm name contains"
// @Param certification_type query string false "certificate type NotCertified, Silver or Gold" Enums(NotCertified, Silver, Gold)
// @Param dedicated query bool false "farm is dedicated"
// @Param stellar_address query string false "farm stellar_address"
// @Param node_free_mru query int false "Min free reservable mru for at least a single node that belongs to the farm, in bytes"
// @Param node_free_hru query int false "Min free reservable hru for at least a single node that belongs to the farm, in bytes"
// @Param node_free_sru query int false "Min free reservable sru for at least a single node that belongs to the farm, in bytes"
// @Param node_total_cru query int false "Min total cpu cores for at least a single node that belongs to the farm"
// @Param node_status query string false "Node status for at least a single node that belongs to the farm"
// @Param node_rented_by query int false "Twin ID of user who has at least one rented node in the farm"
// @Param node_available_for query int false "Twin ID of user for whom there is at least one node that is available to be deployed to in the farm"
// @Param node_has_gpu query bool false "True for farms who have at least one node with a GPU"
// @Param node_certified query bool false "True for farms who have at least one certified node"
// @Param country query string false "farm country"
// @Param region query string false "farm region"
// @Success 200 {object} []types.Farm
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /farms [get]
func (a *App) listFarms(r *http.Request) (interface{}, mw.Response) {
	filter := types.FarmFilter{}
	limit := types.DefaultLimit()
	if err := parseQueryParams(r, &filter, &limit); err != nil {
		return nil, mw.BadRequest(err)
	}
	if err := limit.Valid(types.Farm{}); err != nil {
		return nil, mw.BadRequest(err)
	}

	dbFarms, farmsCount, err := a.cl.Farms(r.Context(), filter, limit)
	if err != nil {
		log.Error().Err(err).Msg("failed to query farm")
		return nil, mw.Error(err)
	}

	// return the number of pages and totalCount in the response headers
	resp := createResponse(uint(farmsCount), limit)

	return dbFarms, resp
}

// getStats godoc
// @Summary Show stats about the grid
// @Description Get statistics about the grid
// @Tags GridProxy
// @Accept  json
// @Produce  json
// @Param status query string false "Node status filter, 'up': for only up nodes, 'down': for only down nodes & 'standby' for powered-off nodes by farmerbot."
// @Success 200 {object} []types.Stats
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /stats [get]
func (a *App) getStats(r *http.Request) (interface{}, mw.Response) {
	filter := types.StatsFilter{}
	limit := types.DefaultLimit()
	if err := parseQueryParams(r, &filter, &limit); err != nil {
		return nil, mw.BadRequest(err)
	}

	stats, err := a.cl.Stats(r.Context(), filter)
	if err != nil {
		return nil, mw.Error(err)
	}

	return stats, nil
}

// getNodes godoc
// @Summary Show nodes on the grid
// @Description Get all nodes on the grid, It has pagination
// @Tags GridProxy
// @Accept  json
// @Produce  json
// @Param page query int false "Page number"
// @Param size query int false "Max result per page"
// @Param ret_count query bool false "Set nodes' count on headers based on filter"
// @Param randomize query bool false "Get random patch of nodes"
// @Param sort_by query string false "Sort by specific node filed" Enums(status, node_id, farm_id, twin_id, uptime, created, updated_at, country, city, dedicated_farm, rent_contract_id, total_cru, total_mru, total_hru, total_sru, used_cru, used_mru, used_hru, used_sru, num_gpu, extra_fee)
// @Param sort_order query string false "The sorting order, default is 'asc'" Enums(desc, asc)
// @Param free_mru query int false "Min free reservable mru in bytes"
// @Param free_hru query int false "Min free reservable hru in bytes"
// @Param free_sru query int false "Min free reservable sru in bytes"
// @Param total_mru query int false "Total mru in bytes"
// @Param total_cru query int false "Total cru number"
// @Param total_sru query int false "Total sru in bytes"
// @Param total_hru query int false "Total hru in bytes"
// @Param free_ips query int false "Min number of free ips in the farm of the node"
// @Param status query string false "Node status filter, 'up': for only up nodes, 'down': for only down nodes & 'standby' for powered-off nodes by farmerbot."
// @Param city query string false "Node city filter"
// @Param country query string false "Node country filter"
// @Param region query string false "Node region"
// @Param farm_name query string false "Get nodes for specific farm"
// @Param ipv4 query bool false "Set to true to filter nodes with ipv4"
// @Param ipv6 query bool false "Set to true to filter nodes with ipv6"
// @Param domain query bool false "Set to true to filter nodes with domain"
// @Param dedicated query bool false "Set to true to get the dedicated nodes only"
// @Param in_dedicated_farm query bool false "Set to true to get the nodes belongs to dedicated farms"
// @Param rentable query bool false "Set to true to filter the available nodes for renting"
// @Param rented query bool false "Set to true to filter rented nodes"
// @Param rented_by query int false "rented by twin id"
// @Param available_for query int false "available for twin id"
// @Param farm_ids query string false "List of farms separated by comma to fetch nodes from (e.g. '1,2,3')"
// @Param certification_type query string false "certificate type" Enums(Certified, DIY)
// @Param has_gpu query bool false "filter nodes on whether they have GPU support or not"
// @Param gpu_device_id query string false "filter nodes based on GPU device ID"
// @Param gpu_device_name query string false "filter nodes based on GPU device partial name"
// @Param gpu_vendor_id query string false "filter nodes based on GPU vendor ID"
// @Param gpu_vendor_name query string false "filter nodes based on GPU vendor partial name"
// @Param gpu_available query bool false "filter nodes that have available GPU"
// @Param owned_by query int false "get nodes owned by twin id"
// @Success 200 {object} []types.Node
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /nodes [get]
func (a *App) getNodes(r *http.Request) (interface{}, mw.Response) {
	return a.listNodes(r)
}

// getGateways godoc
// @Summary Show gateways on the grid
// @Description Get all gateways on the grid, It has pagination
// @Tags GridProxy
// @Accept  json
// @Produce  json
// @Param page query int false "Page number"
// @Param size query int false "Max result per page"
// @Param ret_count query bool false "Set nodes' count on headers based on filter"
// @Param randomize query bool false "Get random patch of gateways"
// @Param sort_by query string false "Sort by specific gateway filed" Enums(node_id, farm_id, twin_id, uptime, created, updated_at, country, city, dedicated_farm, rent_contract_id, total_cru, total_mru, total_hru, total_sru, used_cru, used_mru, used_hru, used_sru, num_gpu, extra_fee)
// @Param sort_order query string false "The sorting order, default is 'asc'" Enums(desc, asc)
// @Param free_mru query int false "Min free reservable mru in bytes"
// @Param free_hru query int false "Min free reservable hru in bytes"
// @Param free_sru query int false "Min free reservable sru in bytes"
// @Param free_ips query int false "Min number of free ips in the farm of the node"
// @Param status query string false "Node status filter, 'up': for only up nodes, 'down': for only down nodes & 'standby' for powered-off nodes by farmerbot."
// @Param city query string false "Node city filter"
// @Param country query string false "Node country filter"
// @Param region query string false "node region"
// @Param farm_name query string false "Get nodes for specific farm"
// @Param ipv4 query bool false "Set to true to filter nodes with ipv4"
// @Param ipv6 query bool false "Set to true to filter nodes with ipv6"
// @Param domain query bool false "Set to true to filter nodes with domain"
// @Param dedicated query bool false "Set to true to get the dedicated nodes only"
// @Param in_dedicated_farm query bool false "Set to true to get the nodes belongs to dedicated farms"
// @Param rentable query bool false "Set to true to filter the available nodes for renting"
// @Param rented query bool false "Set to true to filter rented nodes"
// @Param rented_by query int false "rented by twin id"
// @Param available_for query int false "available for twin id"
// @Param farm_ids query string false "List of farms separated by comma to fetch nodes from (e.g. '1,2,3')"
// @Param certification_type query string false "certificate type" Enums(Certified, DIY)
// @Param owned_by query int false "get nodes owned by twin id"
// @Success 200 {object} []types.Node
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /gateways [get]
func (a *App) getGateways(r *http.Request) (interface{}, mw.Response) {
	return a.listNodes(r)
}

func (a *App) listNodes(r *http.Request) (interface{}, mw.Response) {
	filter := types.NodeFilter{}
	limit := types.DefaultLimit()
	if err := parseQueryParams(r, &filter, &limit); err != nil {
		return nil, mw.BadRequest(err)
	}
	if err := limit.Valid(types.Node{}); err != nil {
		return nil, mw.BadRequest(err)
	}

	dbNodes, nodesCount, err := a.cl.Nodes(r.Context(), filter, limit)
	if err != nil {
		return nil, mw.Error(err)
	}

	resp := createResponse(uint(nodesCount), limit)
	return dbNodes, resp
}

// getNode godoc
// @Summary Show the details for specific node
// @Description Get all details for specific node hardware, capacity, DMI, hypervisor
// @Tags GridProxy
// @Param node_id path int false "Node ID"
// @Accept  json
// @Produce  json
// @Success 200 {object} types.NodeWithNestedCapacity
// @Failure 400 {object} string
// @Failure 404 {object} string
// @Failure 500 {object} string
// @Router /nodes/{node_id} [get]
func (a *App) getNode(r *http.Request) (interface{}, mw.Response) {
	return a._getNode(r)
}

// getGateway godoc
// @Summary Show the details for specific gateway
// @Description Get all details for specific gateway hardware, capacity, DMI, hypervisor
// @Tags GridProxy
// @Param node_id path int false "Node ID"
// @Accept  json
// @Produce  json
// @Success 200 {object} types.NodeWithNestedCapacity
// @Failure 400 {object} string
// @Failure 404 {object} string
// @Failure 500 {object} string
// @Router /gateways/{node_id} [get]
func (a *App) getGateway(r *http.Request) (interface{}, mw.Response) {
	node, err := a._getNode(r)

	if err != nil {
		return nil, err
	} else if node.(types.NodeWithNestedCapacity).PublicConfig.Domain == "" {
		return nil, errorReply(ErrGatewayNotFound)
	} else {
		return node, nil
	}
}

func (a *App) _getNode(r *http.Request) (interface{}, mw.Response) {
	nodeID := mux.Vars(r)["node_id"]
	nodeData, err := a.getNodeData(r.Context(), nodeID)
	if err != nil {
		return nil, errorReply(err)
	}
	return nodeData, nil
}

func (a *App) getNodeStatus(r *http.Request) (interface{}, mw.Response) {
	nodeIDStr := mux.Vars(r)["node_id"]

	nodeID, err := strconv.Atoi(nodeIDStr)
	if err != nil {
		return types.NodeWithNestedCapacity{}, mw.BadRequest(err)
	}

	status, err := a.cl.NodeStatus(r.Context(), uint32(nodeID))
	if err != nil {
		return nil, errorReply(err)
	}

	return status, nil
}

// listTwins godoc
// @Summary Show twins on the grid
// @Description Get all twins on the grid, It has pagination
// @Tags GridProxy
// @Accept  json
// @Produce  json
// @Param page query int false "Page number"
// @Param size query int false "Max result per page"
// @Param ret_count query bool false "Set twins' count on headers based on filter"
// @Param randomize query bool false "Get random patch of twins"
// @Param sort_by query string false "Sort by specific twin filed" Enums(relay, public_key, account_id, twin_id)
// @Param sort_order query string false "The sorting order, default is 'asc'" Enums(desc, asc)
// @Param twin_id query int false "twin id"
// @Param account_id query string false "Account address"
// @Param relay query string false "Relay address"
// @Param public_key query string false "Twin public key"
// @Success 200 {object} []types.Twin
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /twins [get]
func (a *App) listTwins(r *http.Request) (interface{}, mw.Response) {
	filter := types.TwinFilter{}
	limit := types.DefaultLimit()
	if err := parseQueryParams(r, &filter, &limit); err != nil {
		return nil, mw.BadRequest(err)
	}
	if err := limit.Valid(types.Twin{}); err != nil {
		return nil, mw.BadRequest(err)
	}

	twins, twinsCount, err := a.cl.Twins(r.Context(), filter, limit)
	if err != nil {
		log.Error().Err(err).Msg("failed to query twin")
		return nil, mw.Error(err)
	}

	resp := createResponse(uint(twinsCount), limit)
	return twins, resp
}

// listContracts godoc
// @Summary Show contracts on the grid
// @Description Get all contracts on the grid, It has pagination
// @Tags GridProxy
// @Accept  json
// @Produce  json
// @Param page query int false "Page number"
// @Param size query int false "Max result per page"
// @Param ret_count query bool false "Set contracts' count on headers based on filter"
// @Param randomize query bool false "Get random patch of contracts"
// @Param sort_by query string false "Sort by specific contract filed" Enums(twin_id, contract_id, type, state, created_at)
// @Param sort_order query string false "The sorting order, default is 'asc'" Enums(desc, asc)
// @Param contract_id query int false "contract id"
// @Param twin_id query int false "twin id"
// @Param node_id query int false "node id which contract is deployed on in case of ('rent' or 'node' contracts)"
// @Param name query string false "contract name in case of 'name' contracts"
// @Param type query string false "contract type 'node', 'name', or 'rent'"
// @Param state query string false "contract state 'Created', 'GracePeriod', or 'Deleted'"
// @Param deployment_data query string false "contract deployment data in case of 'node' contracts"
// @Param deployment_hash query string false "contract deployment hash in case of 'node' contracts"
// @Param number_of_public_ips query int false "Min number of public ips in the 'node' contract"
// @Success 200 {object} []types.Contract
// @Failure 400 {object} string
// @Failure 500 {object} string
// @Router /contracts [get]
func (a *App) listContracts(r *http.Request) (interface{}, mw.Response) {
	filter := types.ContractFilter{}
	limit := types.DefaultLimit()
	if err := parseQueryParams(r, &filter, &limit); err != nil {
		return nil, mw.BadRequest(err)
	}
	if err := limit.Valid(types.Contract{}); err != nil {
		return nil, mw.BadRequest(err)
	}

	dbContracts, contractsCount, err := a.cl.Contracts(r.Context(), filter, limit)
	if err != nil {
		log.Error().Err(err).Msg("failed to query contract")
		return nil, mw.Error(err)
	}

	resp := createResponse(uint(contractsCount), limit)
	return dbContracts, resp
}

// ping godoc
// @Summary ping the server
// @Description ping the server to check if it is running
// @Tags ping
// @Accept  json
// @Produce  json
// @Success 200 {object} PingMessage
// @Router /ping [get]
func (a *App) ping(r *http.Request) (interface{}, mw.Response) {
	return PingMessage{Ping: "pong"}, mw.Ok()
}

func (a *App) indexPage(m *mux.Router) mw.Action {
	return func(r *http.Request) (interface{}, mw.Response) {
		response := mw.Ok()
		var sb strings.Builder
		sb.WriteString("Welcome to threefold grid proxy server, available endpoints ")

		_ = m.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
			path, err := route.GetPathTemplate()
			if err != nil {
				return nil
			}

			sb.WriteString("[" + path + "] ")
			return nil
		})
		return sb.String(), response
	}
}

func (a *App) version(r *http.Request) (interface{}, mw.Response) {
	response := mw.Ok()
	return types.Version{
		Version: a.releaseVersion,
	}, response
}

// getNodeStatistics godoc
// @Summary Show node statistics
// @Description Get node statistics for more information about each node through the RMB relay
// @Tags NodeStatistics
// @Param node_id path int yes "Node ID"
// @Accept  json
// @Produce  json
// @Success 200 {object} types.NodeStatistics
// @Failure 400 {object} string
// @Failure 404 {object} string
// @Failure 500 {object} string
// @Router /nodes/{node_id}/statistics  [get]
func (a *App) getNodeStatistics(r *http.Request) (interface{}, mw.Response) {
	nodeID := mux.Vars(r)["node_id"]
	node, err := a.getNodeData(r.Context(), nodeID)
	if err != nil {
		return nil, errorReply(err)
	}

	if node.Status == "down" || node.Status == "standby" {
		return nil, mw.Error(fmt.Errorf("cannot fetch statistics from node %d with status: %s", node.NodeID, node.Status))
	}

	var res types.NodeStatistics
	err = a.relayClient.Call(r.Context(), uint32(node.TwinID), "zos.statistics.get", nil, &res)
	if err != nil {
		return nil, mw.Error(fmt.Errorf("failed to get get node statistics from relay: %w", err))
	}
	return res, mw.Ok()
}

// getNodeGpus godoc
// @Summary Show node GPUs information
// @Description Get node GPUs through the RMB relay
// @Tags NodeGPUs
// @Param node_id path int yes "Node ID"
// @Accept  json
// @Produce  json
// @Success 200 {object} []types.NodeGPU
// @Failure 400 {object} string
// @Failure 404 {object} string
// @Failure 500 {object} string
// @Router /nodes/{node_id}/gpu  [get]
func (a *App) getNodeGpus(r *http.Request) (interface{}, mw.Response) {
	nodeID := mux.Vars(r)["node_id"]
	node, err := a.getNodeData(r.Context(), nodeID)
	if err != nil {
		return nil, errorReply(err)
	}

	if node.Status == "down" || node.Status == "standby" {
		return nil, mw.Error(fmt.Errorf("cannot fetch GPU information from node %d with status: %s", node.NodeID, node.Status))
	}

	var res []types.NodeGPU
	err = a.relayClient.Call(r.Context(), uint32(node.TwinID), "zos.gpu.list", nil, &res)
	if err != nil {
		return nil, mw.Error(fmt.Errorf("failed to get get node GPU information from relay: %w", err))
	}
	return res, mw.Ok()
}

// getContract godoc
// @Summary Show single contract info
// @Description Get data about a single contract with its id
// @Tags Contract
// @Param contract_id path int yes "Contract ID"
// @Accept  json
// @Produce  json
// @Success 200 {object} types.Contract
// @Failure 400 {object} string
// @Failure 404 {object} string
// @Failure 500 {object} string
// @Router /contracts/{contract_id} [get]
func (a *App) getContract(r *http.Request) (interface{}, mw.Response) {
	contractID := mux.Vars(r)["contract_id"]

	contractData, err := a.getContractData(r.Context(), contractID)
	if err != nil {
		return nil, errorReply(err)
	}

	return contractData, nil
}

// getContractBills godoc
// @Summary Show single contract bills
// @Description Get all bills reports for a single contract with its id
// @Tags ContractDills
// @Param contract_id path int yes "Contract ID"
// @Param page query int false "Page number"
// @Param size query int false "Max result per page"
// @Param ret_count query bool false "Set bill reports' count on headers"
// @Accept  json
// @Produce  json
// @Success 200 {object} []types.ContractBilling
// @Failure 400 {object} string
// @Failure 404 {object} string
// @Failure 500 {object} string
// @Router /contracts/{contract_id}/bills [get]
func (a *App) getContractBills(r *http.Request) (interface{}, mw.Response) {
	contractID := mux.Vars(r)["contract_id"]

	limit := types.DefaultLimit()
	if err := parseQueryParams(r, &limit); err != nil {
		return nil, mw.BadRequest(err)
	}

	contractBillsData, totalCount, err := a.getContractBillsData(r.Context(), contractID, limit)
	if err != nil {
		return nil, errorReply(err)
	}

	resp := createResponse(totalCount, limit)
	return contractBillsData, resp
}

// Setup is the server and do initial configurations
// @title Grid Proxy Server API
// @version 1.0
// @description grid proxy server has the main methods to list farms, nodes, node details in the grid.
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @BasePath /
func Setup(router *mux.Router, gitCommit string, cl DBClient, relayClient rmb.Client) error {

	a := App{
		cl:             cl,
		releaseVersion: gitCommit,
		relayClient:    relayClient,
	}

	router.HandleFunc("/farms", mw.AsHandlerFunc(a.listFarms))
	router.HandleFunc("/stats", mw.AsHandlerFunc(a.getStats))
	router.HandleFunc("/twins", mw.AsHandlerFunc(a.listTwins))

	router.HandleFunc("/nodes", mw.AsHandlerFunc(a.getNodes))
	router.HandleFunc("/nodes/{node_id:[0-9]+}", mw.AsHandlerFunc(a.getNode))
	router.HandleFunc("/nodes/{node_id:[0-9]+}/status", mw.AsHandlerFunc(a.getNodeStatus))
	router.HandleFunc("/nodes/{node_id:[0-9]+}/statistics", mw.AsHandlerFunc(a.getNodeStatistics))
	router.HandleFunc("/nodes/{node_id:[0-9]+}/gpu", mw.AsHandlerFunc(a.getNodeGpus))

	router.HandleFunc("/gateways", mw.AsHandlerFunc(a.getGateways))
	router.HandleFunc("/gateways/{node_id:[0-9]+}", mw.AsHandlerFunc(a.getGateway))
	router.HandleFunc("/gateways/{node_id:[0-9]+}/status", mw.AsHandlerFunc(a.getNodeStatus))

	router.HandleFunc("/contracts", mw.AsHandlerFunc(a.listContracts))
	router.HandleFunc("/contracts/{contract_id:[0-9]+}", mw.AsHandlerFunc(a.getContract))
	router.HandleFunc("/contracts/{contract_id:[0-9]+}/bills", mw.AsHandlerFunc(a.getContractBills))

	router.HandleFunc("/", mw.AsHandlerFunc(a.indexPage(router)))
	router.HandleFunc("/ping", mw.AsHandlerFunc(a.ping))
	router.HandleFunc("/version", mw.AsHandlerFunc(a.version))
	router.PathPrefix("/swagger/").Handler(httpSwagger.WrapHandler)

	return nil
}
