package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
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
	PublicIps(ctx context.Context, filter types.PublicIpFilter, limit types.Limit) ([]types.PublicIP, uint, error)
}

// Client a client to communicate with the grid proxy
type Client interface {
	Ping() error
	DBClient
	UpdateNodeSlice(ctx context.Context, nodeID uint32, sliceReq types.UpdateNodeSliceRequest, twinID uint32, mnemonic string) error
}

// Clientimpl concrete implementation of the client to communicate with the grid proxy
type Clientimpl struct {
	endpoints      []string
	activeStackIdx int
	traceProvider  trace.TracerProvider
	tracer         trace.Tracer
}

type ClientOption func(*Clientimpl)

func WithTraceProvider(tp trace.TracerProvider) ClientOption {
	return func(cl *Clientimpl) {
		cl.traceProvider = tp
	}
}

func WithEndpoints(endpoints []string) ClientOption {
	return func(cl *Clientimpl) {
		for i, endpoint := range endpoints {
			if endpoint[len(endpoint)-1] != '/' {
				endpoints[i] += "/"
			}
		}

		cl.endpoints = endpoints
	}
}

// NewClient grid proxy client constructor
func NewClient(options ...ClientOption) Client {

	proxy := &Clientimpl{
		activeStackIdx: 0,
		tracer:         noop.NewTracerProvider().Tracer("no-op"),
	}

	for _, option := range options {
		option(proxy)
	}

	if proxy.traceProvider != nil {
		proxy.tracer = proxy.traceProvider.Tracer("grid-proxy")
	}

	return proxy
}

func recordSpanError(span trace.Span, description string, err error) {
	if err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, description)
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

	_, span := g.tracer.Start(ctx, "Proxy.Nodes")
	defer span.End()

	span.AddEvent("sending request to fetch nodes")
	res, err := g.httpGet("nodes", filter, limit)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		recordSpanError(span, "failed to connect to /nodes", err)
		return
	}

	if res.StatusCode != http.StatusOK {
		err = parseError(res.Body)
		recordSpanError(span, "failed to get nodes", err)
		return
	}

	if err := json.NewDecoder(res.Body).Decode(&nodes); err != nil {
		recordSpanError(span, "failed to decode data to Nodes", err)
		return nodes, 0, err
	}

	totalCount, err = requestCounters(res)
	span.SetAttributes(attribute.Int("nodes_count", totalCount))
	return
}

// Farms returns farms with the given filters and pagination parameters
func (g *Clientimpl) Farms(ctx context.Context, filter types.FarmFilter, limit types.Limit) (farms []types.Farm, totalCount int, err error) {

	_, span := g.tracer.Start(ctx, "Proxy.Farms")
	defer span.End()

	span.AddEvent("sending request to fetch farms")
	res, err := g.httpGet("farms", filter, limit)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		recordSpanError(span, "failed to connect to /farms", err)
		return
	}

	if res.StatusCode != http.StatusOK {
		err = parseError(res.Body)
		recordSpanError(span, "failed to get farms", err)
		return
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		recordSpanError(span, "failed to read response body", err)
		return
	}
	err = json.Unmarshal(data, &farms)
	if err != nil {
		recordSpanError(span, "failed to unmarshal data to Farms", err)
		return
	}

	totalCount, err = requestCounters(res)
	span.SetAttributes(attribute.Int("farms_count", totalCount))
	return
}

// Twins returns twins with the given filters and pagination parameters
func (g *Clientimpl) Twins(ctx context.Context, filter types.TwinFilter, limit types.Limit) (twins []types.Twin, totalCount int, err error) {

	_, span := g.tracer.Start(ctx, "Proxy.Twins")
	defer span.End()

	span.AddEvent("sending request to fetch twins")
	res, err := g.httpGet("twins", filter, limit)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		recordSpanError(span, "failed to connect to /twins", err)
		return
	}

	if res.StatusCode != http.StatusOK {
		err = parseError(res.Body)
		recordSpanError(span, "failed to get twins", err)
		return
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		recordSpanError(span, "failed to read response body", err)
		return
	}
	err = json.Unmarshal(data, &twins)
	if err != nil {
		recordSpanError(span, "failed to unmarshal data to Twins", err)
		return
	}

	totalCount, err = requestCounters(res)
	span.SetAttributes(attribute.Int("twins_count", totalCount))
	return
}

// Contracts returns contracts with the given filters and pagination parameters
func (g *Clientimpl) Contracts(ctx context.Context, filter types.ContractFilter, limit types.Limit) (contracts []types.Contract, totalCount int, err error) {

	_, span := g.tracer.Start(ctx, "Proxy.Contracts")
	defer span.End()

	span.AddEvent("sending request to fetch contracts")
	res, err := g.httpGet("contracts", filter, limit)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		recordSpanError(span, "failed to connect to /contracts", err)
		return
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		recordSpanError(span, "failed to read response body", err)
		return
	}

	contracts, err = decodeMultipleContracts(data)
	if err != nil {
		recordSpanError(span, "failed to unmarshal data to Contracts", err)
		return
	}

	totalCount, err = requestCounters(res)
	span.SetAttributes(attribute.Int("contracts_count", totalCount))
	return
}

// Node returns the node with the give id
func (g *Clientimpl) Node(ctx context.Context, nodeID uint32) (node types.NodeWithNestedCapacity, err error) {

	_, span := g.tracer.Start(ctx, "Proxy.Node",
		trace.WithAttributes(attribute.Int("node_id", int(nodeID))))

	defer span.End()

	span.AddEvent("sending request to fetch node by ID")
	res, err := g.httpGet(fmt.Sprintf("nodes/%d", nodeID))
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		recordSpanError(span, "failed to connect to /nodes/node_id", err)
		return
	}

	if res.StatusCode != http.StatusOK {
		err = parseError(res.Body)
		recordSpanError(span, "failed to get node by ID", err)
		return
	}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		recordSpanError(span, "failed to read response body", err)
		return
	}
	err = json.Unmarshal(data, &node)
	if err != nil {
		recordSpanError(span, "failed to unmarshal data to NodeWithNestedCapacity", err)
		return
	}
	return
}

// NodeStatus returns the node status up/down
func (g *Clientimpl) NodeStatus(ctx context.Context, nodeID uint32) (status types.NodeStatus, err error) {

	_, span := g.tracer.Start(ctx, "Proxy.NodeStatus",
		trace.WithAttributes(attribute.Int("node_id", int(nodeID))))

	defer span.End()

	span.AddEvent("sending request to fetch node status")
	res, err := g.httpGet(fmt.Sprintf("nodes/%d/status", nodeID))
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		recordSpanError(span, "failed to connect to /nodes/node_id/status", err)
		return
	}

	if res.StatusCode != http.StatusOK {
		err = parseError(res.Body)
		recordSpanError(span, "failed to get node status", err)
		return
	}
	if err := json.NewDecoder(res.Body).Decode(&status); err != nil {
		recordSpanError(span, "failed to decode data to NodeStatus", err)
		return status, err
	}
	return
}

// Stats return statistics about the grid
func (g *Clientimpl) Stats(ctx context.Context, filter types.StatsFilter) (stats types.Stats, err error) {

	_, span := g.tracer.Start(ctx, "Proxy.Stats")
	defer span.End()

	span.AddEvent("sending request to fetch stats")
	res, err := g.httpGet("stats", filter)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		recordSpanError(span, "failed to connect to /stats", err)
		return
	}

	if res.StatusCode != http.StatusOK {
		err = parseError(res.Body)
		recordSpanError(span, "failed to get stats", err)
		return
	}
	if err := json.NewDecoder(res.Body).Decode(&stats); err != nil {
		recordSpanError(span, "failed to decode data to Stats", err)
		return stats, err
	}
	return
}

// Contract returns a single contract based on the contractID
func (g *Clientimpl) Contract(ctx context.Context, contractID uint32) (types.Contract, error) {

	_, span := g.tracer.Start(ctx, "Proxy.Contract",
		trace.WithAttributes(attribute.Int("contract_id", int(contractID))))

	defer span.End()

	span.AddEvent("sending request to fetch contract by ID")
	res, err := g.httpGet(fmt.Sprintf("contracts/%d", contractID))
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		recordSpanError(span, "failed to connect to /contracts/contract_id", err)
		return types.Contract{}, err
	}

	if res.StatusCode != http.StatusOK {
		err = parseError(res.Body)
		recordSpanError(span, "failed to get contract by ID", err)
		return types.Contract{}, err
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		recordSpanError(span, "failed to read response body", err)
		return types.Contract{}, err
	}

	contract, err := decodeSingleContract(data)
	if err != nil {
		recordSpanError(span, "failed to decode data to Contract", err)
		return types.Contract{}, err
	}

	return contract, nil
}

// ContractBills returns all bills for a single contract based on contractID and pagination params
func (g *Clientimpl) ContractBills(ctx context.Context, contractID uint32, limit types.Limit) ([]types.ContractBilling, uint, error) {

	_, span := g.tracer.Start(ctx, "Proxy.ContractBills",
		trace.WithAttributes(attribute.Int("contract_id", int(contractID))))

	defer span.End()

	span.AddEvent("sending request to fetch Contract bills by contract ID")
	res, err := g.httpGet(fmt.Sprintf("contracts/%d/bills", contractID), limit)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		recordSpanError(span, "failed to connect to /contracts/contract_id/bills", err)
		return nil, 0, err
	}

	if res.StatusCode != http.StatusOK {
		err = parseError(res.Body)
		recordSpanError(span, "failed to get contract bills by contract ID", err)
		return nil, 0, err
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		recordSpanError(span, "failed to read response body", err)
		return nil, 0, err
	}

	count, err := requestCounters(res)
	if err != nil {
		recordSpanError(span, "failed to get contract bills count", err)
		return nil, 0, err
	}

	contractBills := []types.ContractBilling{}
	if err := json.Unmarshal(data, &contractBills); err != nil {
		recordSpanError(span, "failed to unmarshal data to ContractBilling", err)
		return nil, 0, err
	}

	totalCount := uint(count)
	span.SetAttributes(attribute.Int("bills_count", int(totalCount)))

	return contractBills, totalCount, nil
}

// PublicIps returns all public ips on the chain based on filters and pagination params
func (g *Clientimpl) PublicIps(ctx context.Context, filter types.PublicIpFilter, limit types.Limit) ([]types.PublicIP, uint, error) {

	_, span := g.tracer.Start(ctx, "Proxy.PublicIps")
	defer span.End()

	span.AddEvent("sending request to fetch public IPs")
	res, err := g.httpGet("public_ips", filter, limit)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		recordSpanError(span, "failed to connect to /public_ips", err)
		return nil, 0, err
	}

	if res.StatusCode != http.StatusOK {
		err = parseError(res.Body)
		recordSpanError(span, "failed to get public IPs", err)
		return nil, 0, err
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		recordSpanError(span, "failed to read response body", err)
		return nil, 0, err
	}

	count, err := requestCounters(res)
	if err != nil {
		recordSpanError(span, "failed to get public IPs count", err)
		return nil, 0, err
	}

	ips := []types.PublicIP{}
	if err := json.Unmarshal(data, &ips); err != nil {
		recordSpanError(span, "failed to unmarshal data to PublicIP", err)
		return nil, 0, err
	}

	totalCount := uint(count)
	span.SetAttributes(attribute.Int("public_ips_count", int(totalCount)))

	return ips, totalCount, nil
}

func isTimeoutError(err error) bool {
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}
	if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
		return true
	}
	return errors.Is(err, context.DeadlineExceeded) ||
		strings.Contains(err.Error(), "timeout awaiting response headers")
}

func (g *Clientimpl) newHTTPClient() *http.Client {
	return &http.Client{
		Timeout: time.Second * 30,
		Transport: &http.Transport{
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

	baseURL := g.endpoints[g.activeStackIdx]

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
				errors.Is(reqErr, http.ErrServerClosed) ||
				isTimeoutError(reqErr)) {
			g.activeStackIdx = (g.activeStackIdx + 1) % len(g.endpoints)
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

func (g *Clientimpl) httpRequest(ctx context.Context, method, path string, body []byte, headers http.Header, params ...interface{}) (*http.Response, error) {
	client := g.newHTTPClient()

	url, err := g.prepareURL(path, params...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to prepare url")
	}

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return nil, err
	}

	for k, vals := range headers {
		for _, v := range vals {
			req.Header.Add(k, v)
		}
	}

	return client.Do(req)
}

// UpdateNodeSlice updates the slice configuration for a specific node
func (g *Clientimpl) UpdateNodeSlice(ctx context.Context, nodeID uint32, sliceReq types.UpdateNodeSliceRequest, twinID uint32, mnemonic string) error {
	_, span := g.tracer.Start(ctx, "Proxy.UpdateNodeSlice")
	defer span.End()

	path := fmt.Sprintf("nodes/%d/slice", nodeID)

	body, err := json.Marshal(sliceReq)
	if err != nil {
		err = errors.Wrap(err, "failed to marshal UpdateNodeSlice request body")
		recordSpanError(span, "failed to marshal UpdateNodeSlice request body", err)
		return err
	}

	headers := http.Header{}
	headers.Set("Content-Type", "application/json")

	if mnemonic != "" {
		authHeader, err := createAuthHeaderFromMnemonic(mnemonic, twinID)
		if err != nil {
			recordSpanError(span, "failed to create X-Auth header", err)
			return err
		}
		headers.Set("X-Auth", authHeader)
	}

	resp, err := g.httpRequest(ctx, http.MethodPatch, path, body, headers)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		recordSpanError(span, "failed to call UpdateNodeSlice endpoint", err)
		return err
	}

	if resp.StatusCode != http.StatusOK {
		err = parseError(resp.Body)
		recordSpanError(span, "UpdateNodeSlice returned non-OK status", err)
		return err
	}

	return nil
}
