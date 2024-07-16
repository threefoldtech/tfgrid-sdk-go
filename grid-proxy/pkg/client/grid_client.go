package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/gorilla/schema"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
)

var encoder *schema.Encoder

func init() {
	encoder = schema.NewEncoder()

	encoder.RegisterEncoder([]string{}, func(value reflect.Value) string {
		slice, ok := value.Interface().([]string)
		if ok {
			return strings.Join(slice, ",")
		}
		return ""
	})
}

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
	endpoints []string
	r         int
}

// NewClient grid proxy client constructor
func NewClient(endpoints ...string) Client {
	for i, endpoint := range endpoints {
		if endpoint[len(endpoint)-1] != '/' {
			endpoints[i] += "/"
		}
	}

	proxy := Clientimpl{
		endpoints: endpoints,
		r:         rand.Intn(len(endpoints)),
	}

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

// RequestPages returns the pages value from header response
func RequestPages(r *http.Response) (uint64, error) {
	pageStr := r.Header.Get("Pages")
	if pageStr == "" {
		return 0, errors.New("Pages not found on header")
	}

	page, err := strconv.ParseUint(pageStr, 10, 64)
	if err != nil {
		return 0, errors.Wrap(err, "could not parse page header")
	}

	return page, nil
}

// Ping makes sure the server is up
func (g *Clientimpl) Ping() error {
	res, err := g.httpGet("ping")
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
	res, err := g.httpGet("nodes", filter, limit)
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
	res, err := g.httpGet("farms", filter, limit)
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
	res, err := g.httpGet("twins", filter, limit)
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
	res, err := g.httpGet("contracts", filter, limit)
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
	res, err := g.httpGet(fmt.Sprintf("nodes/%d", nodeID))
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
	err = json.Unmarshal(data, &node)
	return
}

// NodeStatus returns the node status up/down
func (g *Clientimpl) NodeStatus(ctx context.Context, nodeID uint32) (status types.NodeStatus, err error) {
	res, err := g.httpGet(fmt.Sprintf("nodes/%d/status", nodeID))
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
	if err := json.NewDecoder(res.Body).Decode(&status); err != nil {
		return status, err
	}
	return
}

// Stats return statistics about the grid
func (g *Clientimpl) Stats(ctx context.Context, filter types.StatsFilter) (stats types.Stats, err error) {
	res, err := g.httpGet("stats", filter)
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
	if err := json.NewDecoder(res.Body).Decode(&stats); err != nil {
		return stats, err
	}
	return
}

// Contract returns a single contract based on the contractID
func (g *Clientimpl) Contract(ctx context.Context, contractID uint32) (types.Contract, error) {
	res, err := g.httpGet(fmt.Sprintf("contracts/%d", contractID))
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
	res, err := g.httpGet(fmt.Sprintf("contracts/%d/bills", contractID), limit)
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
		Timeout: time.Second * 30,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout: time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   time.Second,
			ResponseHeaderTimeout: 5 * time.Second,
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

	baseURL := g.endpoints[g.r]

	u, err := url.ParseRequestURI(baseURL)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse request URI: %s", baseURL)
	}

	u.Path = fmt.Sprintf("/%s", path)
	u.RawQuery = values.Encode()

	return u.String(), nil
}

func (g *Clientimpl) httpGet(path string, params ...interface{}) (resp *http.Response, reqErr error) {
	client := g.newHTTPClient()

	backoffCfg := backoff.WithMaxRetries(
		backoff.NewConstantBackOff(1*time.Millisecond),
		2,
	)

	err := backoff.RetryNotify(func() error {
		url, err := g.prepareURL(path, params...)
		if err != nil {
			reqErr = errors.Wrap(err, "failed to prepare url")
			return nil
		}

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			reqErr = err
			return nil
		}

		resp, reqErr = client.Do(req)
		if reqErr != nil &&
			(errors.Is(reqErr, http.ErrAbortHandler) ||
				errors.Is(reqErr, http.ErrHandlerTimeout) ||
				errors.Is(reqErr, http.ErrServerClosed)) {
			g.r = (g.r + 1) % len(g.endpoints)
			return reqErr
		}

		return nil
	}, backoffCfg, func(err error, _ time.Duration) {
		log.Error().Err(err).Msg("failed to connect to endpoint, retrying")
	})

	if err != nil {
		log.Error().Err(err).Msg("failed to connect to endpoint")
	}

	return
}
