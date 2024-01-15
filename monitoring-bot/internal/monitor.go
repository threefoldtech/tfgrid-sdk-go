// Package internal contains all logic for monitoring service
package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/cosmos/go-bip39"
	"github.com/rs/zerolog/log"
	client "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

type address string
type network string

type config struct {
	testMnemonic         string `env:"TESTNET_MNEMONIC"`
	mainMnemonic         string `env:"MAINNET_MNEMONIC"`
	devMnemonic          string `env:"DEVNET_MNEMONIC"`
	qaMnemonic           string `env:"QANET_MNEMONIC"`
	devFarmName          string `env:"DEV_FARM_NAME"`
	qaFarmName           string `env:"QA_FARM_NAME"`
	testFarmName         string `env:"TEST_FARM_NAME"`
	mainFarmName         string `env:"MAIN_FARM_NAME"`
	botToken             string `env:"BOT_TOKEN"`
	chatID               string `env:"CHAT_ID"`
	intervalMins         int    `env:"MINS"`
	publicStellarSecret  string `env:"PUBLIC_STELLAR_SECRET"`
	publicStellarAddress string `env:"PUBLIC_STELLAR_ADDRESS"`
	testStellarSecret    string `env:"TEST_STELLAR_SECRET"`
	testStellarAddress   string `env:"TEST_STELLAR_ADDRESS"`
}

type wallet struct {
	Address   address `json:"address"`
	Threshold int     `json:"threshold"`
	Name      string  `json:"name"`
}
type wallets struct {
	Mainnet []wallet `json:"mainnet"`
	Testnet []wallet `json:"testnet"`
	Devnet  []wallet `json:"devnet"`
	Qanet   []wallet `json:"qanet"`
}

// Monitor for bot monitoring
type Monitor struct {
	env        config
	mnemonics  map[network]string
	farms      map[network]string
	wallets    wallets
	managers   map[network]client.Manager
	rmbClients map[network]*peer.RpcClient
}

// NewMonitor creates a new instance of monitor
func NewMonitor(ctx context.Context, env config, wallets wallets) (Monitor, error) {
	mon := Monitor{}

	mon.wallets = wallets
	mon.env = env

	mon.mnemonics = make(map[network]string, 4)
	if !bip39.IsMnemonicValid(mon.env.devMnemonic) {
		return mon, errors.New("invalid mnemonic for devNetwork")
	}

	if !bip39.IsMnemonicValid(mon.env.testMnemonic) {
		return mon, errors.New("invalid mnemonic for testNetwork")
	}

	if !bip39.IsMnemonicValid(mon.env.qaMnemonic) {
		return mon, errors.New("invalid mnemonic for qaNetwork")
	}

	if !bip39.IsMnemonicValid(mon.env.mainMnemonic) {
		return mon, errors.New("invalid mnemonic for mainNetwork")
	}

	mon.mnemonics[devNetwork] = mon.env.devMnemonic
	mon.mnemonics[testNetwork] = mon.env.testMnemonic
	mon.mnemonics[qaNetwork] = mon.env.qaMnemonic
	mon.mnemonics[mainNetwork] = mon.env.mainMnemonic

	mon.managers = make(map[network]client.Manager, 4)
	mon.rmbClients = make(map[network]*peer.RpcClient, 4)
	for _, network := range networks {
		mon.managers[network] = client.NewManager(SubstrateURLs[network]...)

		sessionID := fmt.Sprintf("monbot-%d", os.Getpid())
		rmbClient, err := peer.NewRpcClient(ctx, "sr25519", mon.mnemonics[network], RelayURLS[network], sessionID, mon.managers[network], true)
		if err != nil {
			return mon, fmt.Errorf("couldn't create rpc client in network %s with error: %w", network, err)
		}
		mon.rmbClients[network] = rmbClient
	}

	mon.farms = make(map[network]string, 4)
	mon.farms[devNetwork] = mon.env.devFarmName
	mon.farms[testNetwork] = mon.env.testFarmName
	mon.farms[qaNetwork] = mon.env.qaFarmName
	mon.farms[mainNetwork] = mon.env.mainFarmName

	return mon, nil
}

// Start starting the monitoring service
func (m *Monitor) Start(ctx context.Context) error {
	for {
		startTime := time.Now()

		for network, manager := range m.managers {

			wallets := []wallet{}
			switch network {
			case mainNetwork:
				wallets = m.wallets.Mainnet
			case testNetwork:
				wallets = m.wallets.Testnet
			case devNetwork:
				wallets = m.wallets.Devnet
			case qaNetwork:
				wallets = m.wallets.Qanet
			}

			for _, wallet := range wallets {
				log.Debug().Msgf("monitoring for network %v, address %v", network, wallet.Address)
				err := m.monitorBalance(manager, wallet)
				if err != nil {
					log.Error().Err(err).Msg("monitoring balances failed")
				}
			}
		}

		log.Debug().Msg("monitoring proxy for all networks")
		err := m.pingGridProxies()
		if err != nil {
			log.Error().Err(err).Msg("monitoring proxies failed")
		}

		log.Debug().Msg("monitoring relay for all networks")
		err = m.monitorRelay(ctx)
		if err != nil {
			log.Error().Err(err).Msg("monitoring relay failed")
		}

		if len(strings.TrimSpace(m.env.publicStellarAddress)) == 0 ||
			len(strings.TrimSpace(m.env.publicStellarSecret)) == 0 ||
			len(strings.TrimSpace(m.env.testStellarAddress)) == 0 ||
			len(strings.TrimSpace(m.env.testStellarSecret)) == 0 {
			log.Info().Msg("No monitoring for stellar bridges. If you want to monitor it please set the stellar configs")
			continue
		}

		log.Debug().Msg("monitoring stellar bridges")
		if err := m.monitorBridges(); err != nil {
			log.Error().Err(err).Msg("monitoring bridges failed")
		}

		// Time to sleep
		monitorTime := time.Since(startTime)
		timeToSleep := 0 * time.Minute
		if time.Duration(m.env.intervalMins)*time.Minute > monitorTime {
			timeToSleep = time.Duration(m.env.intervalMins)*time.Minute - monitorTime
		}

		log.Debug().Msgf("monitoring time: %v, time to sleep: %v", monitorTime, timeToSleep)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(timeToSleep):
		}
	}
}

// getTelegramUrl returns the telegram bot api url
func (m *Monitor) getTelegramURL() string {
	return fmt.Sprintf("%s%s", telegramBotURL, m.env.botToken)
}

func (m *Monitor) sendBotMessage(msg string) error {
	url := fmt.Sprintf("%s/sendMessage", m.getTelegramURL())
	body, _ := json.Marshal(map[string]string{
		"chat_id": m.env.chatID,
		"text":    msg,
	})

	res, err := http.Post(
		url,
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return fmt.Errorf("failed to send telegram message request: %w", err)
	}

	if res.StatusCode >= 400 {
		return fmt.Errorf("failed to send telegram message request with status code: %d", res.StatusCode)
	}

	defer res.Body.Close()
	return nil
}

// getBalance gets the balance in TFT for the address given
func (m *Monitor) getBalance(manager client.Manager, address address) (float64, error) {
	con, err := manager.Substrate()
	if err != nil {
		return 0, err
	}
	defer con.Close()

	account, err := client.FromAddress(string(address))
	if err != nil {
		return 0, err
	}

	balance, err := con.GetBalance(account)
	if err != nil {
		return 0, err
	}

	return float64(balance.Free.Int64()) / math.Pow(10, 7), nil
}

// monitorBalance sends a message with the balance to a telegram bot
// if it is less than the tft threshold
func (m *Monitor) monitorBalance(manager client.Manager, wallet wallet) error {
	log.Debug().Msgf("get balance for %v", wallet.Address)
	balance, err := m.getBalance(manager, wallet.Address)
	if err != nil {
		return err
	}

	if balance >= float64(wallet.Threshold) {
		return nil
	}
	return m.sendBotMessage(fmt.Sprintf("wallet %s with address:\n%s\nhas balance = %v ⚠️", wallet.Name, wallet.Address, balance))
}

// pingGridProxies pings the different grid proxy networks
func (m *Monitor) pingGridProxies() error {
	var message string
	var failure bool

	for _, network := range networks {
		log.Debug().Msgf("pinging grid proxy for network %s", network)

		gridProxy, err := NewGridProxyClient(ProxyUrls[network])
		if err != nil {
			log.Error().Err(err).Msgf("grid proxy for %v network failed", network)
			message += fmt.Sprintf("Proxy for %v is not working ❌\n", network)
			failure = true
			continue
		}

		if err := gridProxy.Ping(); err != nil {
			log.Error().Err(err).Msgf("failed to ping grid proxy on network %v", network)
			message += fmt.Sprintf("Proxy for %v is not working ❌\n", network)
			failure = true
			continue
		}

		message += fmt.Sprintf("Proxy for %v is working ✅\n", network)
	}

	if !failure {
		return nil
	}

	return m.sendBotMessage(message)
}

// monitorRelay checks if relay is working against all networks
func (m *Monitor) monitorRelay(ctx context.Context) error {
	versions, workingNodes, failedNodes := m.systemVersion(ctx)

	var message string
	var failure bool

	for _, network := range networks {
		if _, ok := versions[network]; !ok {
			message += fmt.Sprintf("relay is not working for network: %s ❌\n\n", network)
			failure = true
			continue
		}

		message += fmt.Sprintf("relay is working for network: %s ✅\n", network)

		if len(failedNodes[network]) > 0 {
			notWorkingTestedNodes := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(failedNodes[network])), ", "), "[]")
			message += fmt.Sprintf("Nodes tested but failed: %v ❌\n", notWorkingTestedNodes)
			failure = true
		}

		if len(workingNodes[network]) > 0 {
			workingTestedNodes := strings.Trim(strings.Join(strings.Fields(fmt.Sprint(workingNodes[network])), ", "), "[]")
			message += fmt.Sprintf("Nodes successfully tested: %v ✅\n", workingTestedNodes)
		}

		message += "\n"
	}

	if !failure {
		return nil
	}

	return m.sendBotMessage(message)
}

type version struct {
	ZOS   string `json:"zos"`
	ZInit string `json:"zinit"`
}

// systemVersion executes system version cmd
func (m *Monitor) systemVersion(ctx context.Context) (map[network]version, map[network][]uint32, map[network][]uint32) {
	versions := map[network]version{}
	workingNodes := make(map[network][]uint32)
	failedNodes := make(map[network][]uint32)

	for _, network := range networks {
		log.Debug().Msgf("get system version for network %v", network)

		con, err := m.managers[network].Substrate()
		if err != nil {
			log.Error().Err(err).Msgf("substrate connection for %v network failed", network)
			continue
		}
		defer con.Close()

		farmID, err := con.GetFarmByName(m.farms[network])
		if err != nil {
			log.Error().Err(err).Msgf("cannot get farm ID for farm '%s'", m.farms[network])
			continue
		}

		farmNodes, err := con.GetNodes(farmID)
		if err != nil {
			log.Error().Err(err).Msgf("cannot get farm nodes for farm %d", farmID)
			continue
		}

		rand.Shuffle(len(farmNodes), func(i, j int) { farmNodes[i], farmNodes[j] = farmNodes[j], farmNodes[i] })
		var randomNodes []uint32
		if len(farmNodes) < 4 {
			randomNodes = farmNodes[:]
		} else {
			randomNodes = farmNodes[:4]
		}

		for _, NodeID := range randomNodes {
			log.Debug().Msgf("check node %d", NodeID)
			ver, working, failed, err := m.checkNodeSystemVersion(ctx, NodeID, network)
			failedNodes[network] = append(failedNodes[network], failed...)
			if err != nil {
				log.Error().Err(err).Msgf("check node %d failed", NodeID)
				continue
			}

			versions[network] = ver
			workingNodes[network] = append(workingNodes[network], working...)
		}
	}

	return versions, workingNodes, failedNodes
}

func (m *Monitor) checkNodeSystemVersion(ctx context.Context, NodeID uint32, net network) (version, []uint32, []uint32, error) {
	const cmd = "zos.system.version"
	var ver version
	var workingNodes []uint32
	var failedNodes []uint32

	con, err := m.managers[net].Substrate()
	if err != nil {
		return version{}, []uint32{}, []uint32{}, fmt.Errorf("substrate connection for %v network failed with error: %w", NodeID, err)
	}
	defer con.Close()

	node, err := con.GetNode(NodeID)
	if err != nil {
		failedNodes = append(failedNodes, NodeID)
		return version{}, []uint32{}, failedNodes, fmt.Errorf("cannot get node %d. failed with error: %w", NodeID, err)
	}

	rmbCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	err = m.rmbClients[net].Call(rmbCtx, uint32(node.TwinID), cmd, nil, &ver)
	if err != nil {
		failedNodes = append(failedNodes, NodeID)
		return version{}, []uint32{}, failedNodes, fmt.Errorf("rmb version call in %s failed using node twin %d with node ID %d: %w", net, node.TwinID, NodeID, err)
	}

	workingNodes = append(workingNodes, NodeID)
	return ver, workingNodes, failedNodes, nil
}
