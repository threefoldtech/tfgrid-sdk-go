package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gorilla/schema"
	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

var encoder = schema.NewEncoder()

type DBClient interface {
	Nodes(ctx context.Context, filter types.NodeFilter, pagination types.Limit) (res []types.Node, totalCount int, err error)
	Farms(ctx context.Context, filter types.FarmFilter, pagination types.Limit) (res []types.Farm, totalCount int, err error)
	Contracts(ctx context.Context, filter types.ContractFilter, pagination types.Limit) (res []types.Contract, totalCount int, err error)
	Contract(ctx context.Context, contractID uint32) (types.Contract, error)
	ContractBills(ctx context.Context, contractID uint32, limit types.Limit) ([]types.ContractBilling, uint, error)
	Twins(ctx context.Context, filter types.TwinFilter, pagination types.Limit) (res []types.Twin, totalCount int, err error)
	Node(ctx context.Context, nodeID uint32) (res types.NodeWithNestedCapacity, err error)
	NodeStatus(ctx context.Context, nodeID uint32) (res types.NodeStatus, err error)
	Stats(ctx context.Context, filter types.StatsFilter) (res types.Stats, err error)
}

// Client a client to communicate with the grid proxy
type Client interface {
	Ping() error
	DBClient
}

// Clientimpl concrete implementation of the client to communicate with the grid proxy
type Clientimpl struct {
	endpoint string
}

// NewClient grid proxy client constructor
func NewClient(endpoint string) Client {
	if endpoint[len(endpoint)-1] != '/' {
		endpoint += "/"
	}
	proxy := Clientimpl{endpoint}
	return &proxy
}

func parseError(body io.ReadCloser) error {
	text, err := io.ReadAll(body)
	if err != nil {
		return errors.Wrap(err, "couldn't read body response")
	}
	var res ErrorReply
	if err := json.Unmarshal(text, &res); err != nil {
		return errors.New(string(text))
	}
	return fmt.Errorf("%s", res.Error)
}

func requestCounters(r *http.Response) (int, error) {
	counth := r.Header.Get("Count")
	if counth != "" {
		count, err := strconv.ParseInt(counth, 10, 32)
		if err != nil {
			return 0, errors.Wrap(err, "couldn't parse count header")
		}
		return int(count), nil
	}
	return 0, nil
}

// Ping makes sure the server is up
func (g *Clientimpl) Ping() error {
	client := g.newHTTPClient()
	url, err := g.prepareURL("ping")
	if err != nil {
		return errors.Wrap(err, "failed to prepare url")
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	res, err := client.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("non ok return status code from the the grid proxy home page: %s", http.StatusText(res.StatusCode))
	}

	return nil
}

// Nodes returns nodes with the given filters and pagination parameters
func (g *Clientimpl) Nodes(ctx context.Context, filter types.NodeFilter, limit types.Limit) (nodes []types.Node, totalCount int, err error) {
	client := g.newHTTPClient()
	url, err := g.prepareURL("nodes", filter, limit)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to prepare url")
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return
	}

	res, err := client.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return
	}

	if res.StatusCode != http.StatusOK {
		err = parseError(res.Body)
		return
	}

	if err := json.NewDecoder(res.Body).Decode(&nodes); err != nil {
		return nodes, 0, err
	}
	totalCount, err = requestCounters(res)
	return
}

// Farms returns farms with the given filters and pagination parameters
func (g *Clientimpl) Farms(ctx context.Context, filter types.FarmFilter, limit types.Limit) (farms []types.Farm, totalCount int, err error) {
	url, err := g.prepareURL("farms", filter, limit)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to prepare url")
	}

	client := g.newHTTPClient()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return
	}

	res, err := client.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return
	}

	if res.StatusCode != http.StatusOK {
		err = parseError(res.Body)
		return
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &farms)
	if err != nil {
		return
	}
	totalCount, err = requestCounters(res)
	return
}

// Twins returns twins with the given filters and pagination parameters
func (g *Clientimpl) Twins(ctx context.Context, filter types.TwinFilter, limit types.Limit) (twins []types.Twin, totalCount int, err error) {
	url, err := g.prepareURL("twins", filter, limit)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to prepare url")
	}

	client := g.newHTTPClient()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return
	}

	res, err := client.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return
	}

	if res.StatusCode != http.StatusOK {
		err = parseError(res.Body)
		return
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &twins)
	if err != nil {
		return
	}
	totalCount, err = requestCounters(res)
	return
}

// Contracts returns contracts with the given filters and pagination parameters
func (g *Clientimpl) Contracts(ctx context.Context, filter types.ContractFilter, limit types.Limit) (contracts []types.Contract, totalCount int, err error) {
	url, err := g.prepareURL("contracts", filter, limit)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to prepare url")
	}

	client := g.newHTTPClient()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return
	}

	res, err := client.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return
	}

	contracts, err = decodeMultipleContracts(data)
	if err != nil {
		return
	}
	totalCount, err = requestCounters(res)
	return
}

// Node returns the node with the give id
func (g *Clientimpl) Node(ctx context.Context, nodeID uint32) (node types.NodeWithNestedCapacity, err error) {
	client := g.newHTTPClient()
	url, err := g.prepareURL(fmt.Sprintf("nodes/%d", nodeID))
	if err != nil {
		return types.NodeWithNestedCapacity{}, errors.Wrap(err, "failed to prepare url")
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return types.NodeWithNestedCapacity{}, fmt.Errorf("failed to create node request: %w", err)
	}

	res, err := client.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return types.NodeWithNestedCapacity{}, err
	}

	if res.StatusCode != http.StatusOK {
		err = parseError(res.Body)
		return
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(data, &node)
	return
}

// NodeStatus returns the node status up/down
func (g *Clientimpl) NodeStatus(ctx context.Context, nodeID uint32) (status types.NodeStatus, err error) {
	client := g.newHTTPClient()
	url, err := g.prepareURL(fmt.Sprintf("nodes/%d/status", nodeID))
	if err != nil {
		return types.NodeStatus{}, errors.Wrap(err, "failed to prepare url")
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return types.NodeStatus{}, fmt.Errorf("failed to create nodes request: %w", err)
	}

	res, err := client.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return types.NodeStatus{}, err
	}

	if res.StatusCode != http.StatusOK {
		err = parseError(res.Body)
		return
	}
	if err := json.NewDecoder(res.Body).Decode(&status); err != nil {
		return status, err
	}
	return
}

// Stats return statistics about the grid
func (g *Clientimpl) Stats(ctx context.Context, filter types.StatsFilter) (stats types.Stats, err error) {
	url, err := g.prepareURL("stats", filter)
	if err != nil {
		return types.Stats{}, errors.Wrap(err, "failed to prepare url")
	}

	client := g.newHTTPClient()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return types.Stats{}, fmt.Errorf("failed to create stats request: %w", err)
	}

	res, err := client.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return types.Stats{}, err
	}

	if res.StatusCode != http.StatusOK {
		err = parseError(res.Body)
		return
	}
	if err := json.NewDecoder(res.Body).Decode(&stats); err != nil {
		return stats, err
	}
	return
}

// Contract returns a single contract based on the contractID
func (g *Clientimpl) Contract(ctx context.Context, contractID uint32) (types.Contract, error) {
	client := g.newHTTPClient()
	url, err := g.prepareURL(fmt.Sprintf("contracts/%d", contractID))
	if err != nil {
		return types.Contract{}, errors.Wrap(err, "failed to prepare url")
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return types.Contract{}, fmt.Errorf("failed to create contract request: %w", err)
	}

	res, err := client.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return types.Contract{}, err
	}

	if res.StatusCode != http.StatusOK {
		err = parseError(res.Body)
		return types.Contract{}, err
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return types.Contract{}, err
	}

	contract, err := decodeSingleContract(data)
	if err != nil {
		return types.Contract{}, err
	}

	return contract, nil
}

// ContractBills returns all bills for a single contract based on contractID and pagination params
func (g *Clientimpl) ContractBills(ctx context.Context, contractID uint32, limit types.Limit) ([]types.ContractBilling, uint, error) {
	client := g.newHTTPClient()
	url, err := g.prepareURL(fmt.Sprintf("contracts/%d/bills", contractID), limit)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed to prepare url")
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, 0, err
	}

	res, err := client.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return nil, 0, err
	}

	if res.StatusCode != http.StatusOK {
		err = parseError(res.Body)
		return nil, 0, err
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, 0, err
	}

	count, err := requestCounters(res)
	if err != nil {
		return nil, 0, err
	}

	contractBills := []types.ContractBilling{}
	if err := json.Unmarshal(data, &contractBills); err != nil {
		return nil, 0, err
	}

	totalCount := uint(count)

	return contractBills, totalCount, nil
}

func (g *Clientimpl) newHTTPClient() *http.Client {
	return &http.Client{
		Timeout: time.Second * 100,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   time.Second,
			ResponseHeaderTimeout: 100 * time.Second,
		},
	}
}

func (g *Clientimpl) prepareURL(path string, params ...interface{}) (string, error) {
	values := url.Values{}

	for _, param := range params {
		if err := encoder.Encode(param, values); err != nil {
			return "", errors.Wrap(err, "failed to encode query params")
		}
	}

	baseURL := g.endpoint

	u, err := url.ParseRequestURI(baseURL)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse request URI: %s", baseURL)
	}

	u.Path = fmt.Sprintf("/%s", path)
	u.RawQuery = values.Encode()

	return u.String(), nil
}
