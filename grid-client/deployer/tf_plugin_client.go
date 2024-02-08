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
	// RelayURLS relay urls
	RelayURLS = map[string]string{
		"dev":  "wss://relay.dev.grid.tf",
		"test": "wss://relay.test.grid.tf",
		"qa":   "wss://relay.qa.grid.tf",
		"main": "wss://relay.grid.tf",
	}
)

// TFPluginClient is a Threefold plugin client
type TFPluginClient struct {
	TwinID         uint32
	mnemonicOrSeed string
	Identity       substrate.Identity
	substrateURL   []string
	relayURL       string
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

// NewTFPluginClient generates a new tf plugin client
func NewTFPluginClient(
	mnemonicOrSeed string,
	keyType string,
	network string,
	substrateURL string,
	relayURL string,
	proxyURL string,
	rmbTimeout int,
	showLogs bool,
	rmbInMemCache bool,
) (TFPluginClient, error) {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	if showLogs {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
		baseLog.SetOutput(io.Discard)
	}

	var err error
	tfPluginClient := TFPluginClient{}

	if valid := validateMnemonics(mnemonicOrSeed); !valid {
		_, ok := subkey.DecodeHex(mnemonicOrSeed)
		if !ok {
			return TFPluginClient{}, fmt.Errorf("mnemonic/seed '%s' is invalid", mnemonicOrSeed)
		}
	}
	tfPluginClient.mnemonicOrSeed = mnemonicOrSeed

	var identity substrate.Identity
	switch keyType {
	case "ed25519":
		identity, err = substrate.NewIdentityFromEd25519Phrase(tfPluginClient.mnemonicOrSeed)
	case "sr25519":
		identity, err = substrate.NewIdentityFromSr25519Phrase(tfPluginClient.mnemonicOrSeed)
	default:
		err = errors.Errorf("key type must be one of ed25519 and sr25519 not %s", keyType)
	}

	if err != nil {
		return TFPluginClient{}, errors.Wrapf(err, "error getting identity using '%s'", mnemonicOrSeed)
	}
	tfPluginClient.Identity = identity

	keyPair, err := identity.KeyPair()
	if err != nil {
		return TFPluginClient{}, errors.Wrap(err, "error getting user's identity key pair")
	}

	if network != "dev" && network != "qa" && network != "test" && network != "main" {
		return TFPluginClient{}, errors.Errorf("network must be one of dev, qa, test, and main not %s", network)
	}
	tfPluginClient.Network = network

	tfPluginClient.substrateURL = SubstrateURLs[network]
	if len(strings.TrimSpace(substrateURL)) != 0 {
		if err := validateWssURL(substrateURL); err != nil {
			return TFPluginClient{}, errors.Wrapf(err, "could not validate substrate url %s", substrateURL)
		}
		tfPluginClient.substrateURL = []string{substrateURL}
	}

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

	tfPluginClient.proxyURL = ProxyURLs[network]
	if len(strings.TrimSpace(proxyURL)) != 0 {
		if err := validateProxyURL(proxyURL); err != nil {
			return TFPluginClient{}, errors.Wrapf(err, "could not validate proxy url %s", proxyURL)
		}
		tfPluginClient.proxyURL = proxyURL
	}

	tfPluginClient.useRmbProxy = true
	// if tfPluginClient.useRmbProxy
	sessionID := generateSessionID()

	tfPluginClient.relayURL = RelayURLS[network]
	if len(strings.TrimSpace(relayURL)) != 0 {
		if err := validateWssURL(relayURL); err != nil {
			return TFPluginClient{}, errors.Wrapf(err, "could not validate relay url %s", relayURL)
		}
		tfPluginClient.relayURL = relayURL
	}

	// default rmbTimeout is 60
	if rmbTimeout == 0 {
		rmbTimeout = 60
	}
	tfPluginClient.RMBTimeout = time.Second * time.Duration(rmbTimeout)

	ctx, cancel := context.WithCancel(context.Background())
	tfPluginClient.cancelRelayContext = cancel

	 peerOpts := []peer.PeerOpt{
	          peer.WithRelay(tfPluginClient.relayURL),
	          peer.WithSession(sessionID)),
	          peer.WithKeyType(keyType),
	 }
	 
	if !rmbInMemCache {
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

	graphqlURL := GraphQlURLs[network]
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
