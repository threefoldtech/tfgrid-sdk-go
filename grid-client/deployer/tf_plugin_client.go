package deployer

import (
	"context"
	"fmt"
	"io"
	baseLog "log"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/calculator"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/graphql"
	client "github.com/threefoldtech/tfgrid-sdk-go/grid-client/node"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/state"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-client/subi"
	proxy "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
	"github.com/vedhavyas/go-subkey"
)

var (
	// SubstrateURLs are substrate urls
	SubstrateURLs = map[string][]string{
		"dev":  {"wss://tfchain.dev.grid.tf/ws", "wss://tfchain.dev.grid.tf:443"},
		"test": {"wss://tfchain.test.grid.tf/ws", "wss://tfchain.test.grid.tf:443"},
		"qa":   {"wss://tfchain.qa.grid.tf/ws", "wss://tfchain.qa.grid.tf:443"},
		"main": {"wss://tfchain.grid.tf/ws", "wss://tfchain.grid.tf:443"},
	}
	// ProxyURLs are rmb proxy urls
	ProxyURLs = map[string]string{
		"dev":  "https://gridproxy.dev.grid.tf/",
		"test": "https://gridproxy.test.grid.tf/",
		"qa":   "https://gridproxy.qa.grid.tf/",
		"main": "https://gridproxy.grid.tf/",
	}
	// GraphQlURLs urls
	GraphQlURLs = map[string]string{
		"dev":  "https://graphql.dev.grid.tf/graphql",
		"test": "https://graphql.test.grid.tf/graphql",
		"qa":   "https://graphql.qa.grid.tf/graphql",
		"main": "https://graphql.grid.tf/graphql",
	}
	// RelayURLs relay urls
	RelayURLs = map[string][]string{
		"dev": {
			"wss://relay.dev.grid.tf",
			"wss://relay.02.dev.grid.tf",
		},
		"test": {
			"wss://relay.test.grid.tf",
			"wss://relay.02.test.grid.tf",
		},
		"qa": {
			"wss://relay.qa.grid.tf",
		},
		"main": {
			"wss://relay.grid.tf",
		},
	}
)

// TFPluginClient is a Threefold plugin client
type TFPluginClient struct {
	TwinID         uint32
	mnemonicOrSeed string
	Identity       substrate.Identity
	substrateURL   []string
	relayURLs      []string
	RMBTimeout     time.Duration
	proxyURL       string
	useRmbProxy    bool

	// network
	Network string

	// clients
	GridProxyClient proxy.Client
	RMB             rmb.Client
	SubstrateConn   subi.SubstrateExt
	NcPool          client.NodeClientGetter

	// deployers
	DeploymentDeployer  DeploymentDeployer
	NetworkDeployer     NetworkDeployer
	GatewayFQDNDeployer GatewayFQDNDeployer
	GatewayNameDeployer GatewayNameDeployer
	K8sDeployer         K8sDeployer

	// state
	State *state.State

	// contracts
	graphQl         graphql.GraphQl
	ContractsGetter graphql.ContractsGetter

	// calculator
	Calculator calculator.Calculator

	cancelRelayContext context.CancelFunc
}

type pluginCfg struct {
	keyType       string
	network       string
	substrateURL  []string
	relayURLs     []string
	proxyURL      string
	rmbTimeout    int
	showLogs      bool
	rmbInMemCache bool
}

type PluginOpt func(*pluginCfg)

func WithNetwork(network string) PluginOpt {
	return func(p *pluginCfg) {
		p.network = network
	}
}

func WithKeyType(keyType string) PluginOpt {
	return func(p *pluginCfg) {
		p.keyType = keyType
	}
}

func WithSubstrateURL(substrateURL ...string) PluginOpt {
	return func(p *pluginCfg) {
		p.substrateURL = substrateURL
	}
}

func WithRelayURL(relayURLs ...string) PluginOpt {
	return func(p *pluginCfg) {
		p.relayURLs = relayURLs
	}
}

func WithProxyURL(proxyURL string) PluginOpt {
	return func(p *pluginCfg) {
		p.proxyURL = proxyURL
	}
}

func WithRMBTimeout(rmbTimeout int) PluginOpt {
	return func(p *pluginCfg) {
		p.rmbTimeout = rmbTimeout
	}
}

func WithLogs() PluginOpt {
	return func(p *pluginCfg) {
		p.showLogs = true
	}
}

func WithTwinCache() PluginOpt {
	return func(p *pluginCfg) {
		p.rmbInMemCache = false
	}
}

func parsePluginOpts(opts ...PluginOpt) (pluginCfg, error) {
	cfg := pluginCfg{
		network:       "main",
		keyType:       peer.KeyTypeSr25519,
		substrateURL:  []string{},
		proxyURL:      "",
		relayURLs:     []string{},
		rmbTimeout:    60, // default rmbTimeout is 60
		showLogs:      false,
		rmbInMemCache: true,
	}

	for _, o := range opts {
		o(&cfg)
	}

	if strings.TrimSpace(cfg.proxyURL) == "" {
		cfg.proxyURL = ProxyURLs[cfg.network]
	}
	if err := validateProxyURL(cfg.proxyURL); err != nil {
		return cfg, errors.Wrapf(err, "could not validate proxy url %s", cfg.proxyURL)
	}

	if len(cfg.relayURLs) == 0 {
		cfg.relayURLs = RelayURLs[cfg.network]
	}
	for _, url := range cfg.relayURLs {
		if err := validateWssURL(url); err != nil {
			return cfg, errors.Wrapf(err, "could not validate relay url %s", url)
		}
	}

	if len(cfg.substrateURL) == 0 {
		cfg.substrateURL = SubstrateURLs[cfg.network]
	}
	for _, url := range cfg.substrateURL {
		if err := validateWssURL(url); err != nil {
			return cfg, errors.Wrapf(err, "could not validate substrate url %s", url)
		}
	}

	return cfg, nil
}

// NewTFPluginClient generates a new tf plugin client
func NewTFPluginClient(
	mnemonicOrSeed string,
	opts ...PluginOpt,
) (TFPluginClient, error) {
	cfg, err := parsePluginOpts(opts...)
	if err != nil {
		return TFPluginClient{}, err
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	if cfg.showLogs {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		baseLog.SetOutput(io.Discard)
	}

	tfPluginClient := TFPluginClient{}

	if valid := validateMnemonics(mnemonicOrSeed); !valid {
		_, ok := subkey.DecodeHex(mnemonicOrSeed)
		if !ok {
			return TFPluginClient{}, fmt.Errorf("mnemonic/seed '%s' is invalid", mnemonicOrSeed)
		}
	}
	tfPluginClient.mnemonicOrSeed = mnemonicOrSeed

	var identity substrate.Identity
	switch cfg.keyType {
	case "ed25519":
		identity, err = substrate.NewIdentityFromEd25519Phrase(tfPluginClient.mnemonicOrSeed)
	case "sr25519":
		identity, err = substrate.NewIdentityFromSr25519Phrase(tfPluginClient.mnemonicOrSeed)
	default:
		err = errors.Errorf("key type must be one of ed25519 and sr25519 not %s", cfg.keyType)
	}

	if err != nil {
		return TFPluginClient{}, errors.Wrapf(err, "error getting identity using '%s'", mnemonicOrSeed)
	}
	tfPluginClient.Identity = identity

	keyPair, err := identity.KeyPair()
	if err != nil {
		return TFPluginClient{}, errors.Wrap(err, "error getting user's identity key pair")
	}

	if cfg.network != "dev" && cfg.network != "qa" && cfg.network != "test" && cfg.network != "main" {
		return TFPluginClient{}, errors.Errorf("network must be one of dev, qa, test, and main not %s", cfg.network)
	}
	tfPluginClient.Network = cfg.network

	tfPluginClient.substrateURL = cfg.substrateURL
	tfPluginClient.proxyURL = cfg.proxyURL
	tfPluginClient.relayURLs = cfg.relayURLs

	manager := subi.NewManager(tfPluginClient.substrateURL...)
	sub, err := manager.SubstrateExt()
	if err != nil {
		return TFPluginClient{}, errors.Wrap(err, "could not get substrate client")
	}

	if err := validateAccount(sub, tfPluginClient.Identity, tfPluginClient.mnemonicOrSeed); err != nil {
		return TFPluginClient{}, errors.Wrap(err, "could not validate substrate account")
	}

	tfPluginClient.SubstrateConn = sub

	twinID, err := sub.GetTwinByPubKey(keyPair.Public())
	if err != nil && errors.Is(err, substrate.ErrNotFound) {
		return TFPluginClient{}, errors.Wrap(err, "no twin associated with the account with the given mnemonic/seed")
	}
	if err != nil {
		return TFPluginClient{}, errors.Wrapf(err, "failed to get twin for the given mnemonic/seed %s", mnemonicOrSeed)
	}
	tfPluginClient.TwinID = twinID

	tfPluginClient.useRmbProxy = true
	// if tfPluginClient.useRmbProxy
	sessionID := generateSessionID()

	// default rmbTimeout is 60
	if cfg.rmbTimeout == 0 {
		cfg.rmbTimeout = 60
	}
	tfPluginClient.RMBTimeout = time.Second * time.Duration(cfg.rmbTimeout)

	ctx, cancel := context.WithCancel(context.Background())
	tfPluginClient.cancelRelayContext = cancel

	peerOpts := []peer.PeerOpt{
		peer.WithRelay(tfPluginClient.relayURLs...),
		peer.WithSession(sessionID),
		peer.WithKeyType(cfg.keyType),
	}

	if !cfg.rmbInMemCache {
		peerOpts = append(peerOpts, peer.WithTwinCache(10*60*60)) // in seconds that's 10 hours
	}
	rmbClient, err := peer.NewRpcClient(ctx, tfPluginClient.mnemonicOrSeed, manager, peerOpts...)
	if err != nil {
		return TFPluginClient{}, errors.Wrap(err, "could not create rmb client")
	}

	tfPluginClient.RMB = rmbClient

	gridProxyClient := proxy.NewClient(tfPluginClient.proxyURL)
	if err := validateRMBProxyServer(gridProxyClient); err != nil {
		return TFPluginClient{}, errors.Wrap(err, "could not validate rmb proxy server")
	}
	tfPluginClient.GridProxyClient = proxy.NewRetryingClient(gridProxyClient)

	ncPool := client.NewNodeClientPool(tfPluginClient.RMB, tfPluginClient.RMBTimeout)
	tfPluginClient.NcPool = ncPool

	tfPluginClient.DeploymentDeployer = NewDeploymentDeployer(&tfPluginClient)
	tfPluginClient.NetworkDeployer = NewNetworkDeployer(&tfPluginClient)
	tfPluginClient.GatewayFQDNDeployer = NewGatewayFqdnDeployer(&tfPluginClient)
	tfPluginClient.K8sDeployer = NewK8sDeployer(&tfPluginClient)
	tfPluginClient.GatewayNameDeployer = NewGatewayNameDeployer(&tfPluginClient)

	tfPluginClient.State = state.NewState(tfPluginClient.NcPool, tfPluginClient.SubstrateConn)

	graphqlURL := GraphQlURLs[cfg.network]
	tfPluginClient.graphQl, err = graphql.NewGraphQl(graphqlURL)
	if err != nil {
		return TFPluginClient{}, errors.Wrapf(err, "could not create a new graphql with url: %s", graphqlURL)
	}

	tfPluginClient.ContractsGetter = graphql.NewContractsGetter(tfPluginClient.TwinID, tfPluginClient.graphQl, tfPluginClient.SubstrateConn, tfPluginClient.NcPool)

	tfPluginClient.Calculator = calculator.NewCalculator(tfPluginClient.SubstrateConn, tfPluginClient.Identity)

	return tfPluginClient, nil
}

// Close closes the relay connection and the substrate connection
func (t *TFPluginClient) Close() {
	// close substrate connection
	t.SubstrateConn.Close()

	// close relay connection
	t.cancelRelayContext()
}

// BatchCancelContract to cancel a batch of contracts
func (t *TFPluginClient) BatchCancelContract(contracts []uint64) error {
	return t.SubstrateConn.BatchCancelContract(t.Identity, contracts)
}

func generateSessionID() string {
	return fmt.Sprintf("tf-%d", os.Getpid())
}
