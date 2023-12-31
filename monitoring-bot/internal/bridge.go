package internal

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	strNet "github.com/stellar/go/network"
	"github.com/stellar/go/txnbuild"
	client "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
)

type balanceReport struct {
	original              float64
	afterSendingToChain   float64
	afterSendingToStellar float64
}

type bridgeTX func(client.Identity, network) error

func (m *Monitor) monitorBridges() error {
	message := strings.Builder{}

	_, _ = message.WriteString("TfChain Bridges to Stellar Monitor Results:\n\n")

	for _, net := range networks {
		failureMessage := fmt.Sprintf("Bridge for %v is not working ❌", net)
		successMessage := fmt.Sprintf("Bridge for %v is working ✅", net)

		report, err := m.monitorBridge(net)
		if err != nil {
			message.WriteString(fmt.Sprintf("- %s\n%s\n\n", failureMessage, err.Error()))
			continue
		}

		message.WriteString(fmt.Sprintf("- %s\n", successMessage))
		message.WriteString("\tAccount balance reports:\n")
		message.WriteString(fmt.Sprintf("\t\t- Original balance: %f TFT\n", report.original))
		message.WriteString(fmt.Sprintf("\t\t- Balance after deposit with stellar bridge: %f TFT\n", report.afterSendingToChain))
		message.WriteString(fmt.Sprintf("\t\t- Balance after withdraw with stellar bridge: %f TFT\n", report.afterSendingToStellar))
		message.WriteString("\n")
	}

	if err := m.sendBotMessage(message.String()); err != nil {
		return err
	}

	return nil
}

func (m *Monitor) monitorBridge(net network) (balanceReport, error) {
	identity, err := client.NewIdentityFromSr25519Phrase(m.mnemonics[net])
	if err != nil {
		return balanceReport{}, err
	}

	// get current balance
	originalBalance, err := m.getBalance(m.managers[net], address(identity.Address()))
	if err != nil {
		return balanceReport{}, fmt.Errorf("failed to get balance for account: %w", err)
	}

	balanceAfterChain, err := m.bridgeTXWrapper(m.sendToTfChain)(identity, net)
	if err != nil {
		return balanceReport{}, err
	}

	balanceAfterStellar, err := m.bridgeTXWrapper(m.sendToStellar)(identity, net)
	if err != nil {
		return balanceReport{}, err
	}

	return balanceReport{
		original:              originalBalance,
		afterSendingToChain:   balanceAfterChain,
		afterSendingToStellar: balanceAfterStellar,
	}, nil
}

// bridgeTXWrapper does the bridge transaction, and get the balance after waiting a period of time
func (m *Monitor) bridgeTXWrapper(tx bridgeTX) func(identity client.Identity, net network) (float64, error) {
	return func(identity client.Identity, net network) (float64, error) {
		if err := tx(identity, net); err != nil {
			return 0, err
		}

		<-time.After(balanceWaitIntervalSeconds * time.Second)

		balanceAfterChain, err := m.getBalance(m.managers[net], address(identity.Address()))
		if err != nil {
			return 0, fmt.Errorf("failed to get balance for account: %w", err)
		}

		return balanceAfterChain, nil
	}
}

func (m *Monitor) sendToTfChain(identity client.Identity, net network) error {
	conn, err := m.managers[net].Substrate()
	if err != nil {
		return fmt.Errorf("failed to create substrate connection for %s: %w", net, err)
	}
	defer conn.Close()

	// decide configs based on networks
	strSecret := m.env.testStellarSecret
	stellarTFTIssuerAddress := tftIssuerStellarTest
	strClient := horizonclient.DefaultTestNetClient
	netPassphrase := strNet.TestNetworkPassphrase
	if net == mainNetwork || net == testNetwork {
		strSecret = m.env.publicStellarSecret
		stellarTFTIssuerAddress = tftIssuerStellarPublic
		strClient = horizonclient.DefaultPublicNetClient
		netPassphrase = strNet.PublicNetworkPassphrase
	}

	// Validate destination and Load source Accounts
	destAccountRequest := horizonclient.AccountRequest{AccountID: BridgeAddresses[net]}
	_, err = strClient.AccountDetail(destAccountRequest)
	if err != nil {
		errMsg := getHorizonError(err)
		return fmt.Errorf("failed to verify destination account: %s", errMsg)
	}

	sourceKP, err := keypair.ParseFull(strSecret)
	if err != nil {
		return fmt.Errorf("failed to parse secret address: %w", err)
	}

	sourceAccountRequest := horizonclient.AccountRequest{AccountID: sourceKP.Address()}
	sourceAccount, err := strClient.AccountDetail(sourceAccountRequest)
	if err != nil {
		errMsg := getHorizonError(err)
		return fmt.Errorf("failed to load source account: %s", errMsg)
	}

	// Build, Sign and Submit the txn
	tftTrustLine := txnbuild.CreditAsset{Code: "TFT", Issuer: stellarTFTIssuerAddress}
	twinID, err := conn.GetTwinByPubKey(identity.PublicKey())
	if err != nil {
		return fmt.Errorf("failed to get twin id: %w", err)
	}
	tx, err := txnbuild.NewTransaction(
		txnbuild.TransactionParams{
			SourceAccount:        &sourceAccount,
			IncrementSequenceNum: true,
			BaseFee:              txnbuild.MinBaseFee,
			Preconditions: txnbuild.Preconditions{
				TimeBounds: txnbuild.NewTimeout(txnTimeoutSeconds),
			},
			Operations: []txnbuild.Operation{
				&txnbuild.Payment{
					Destination: BridgeAddresses[net],
					Amount:      fmt.Sprintf("%d", bridgeTestTFTAmount),
					Asset:       tftTrustLine,
				},
			},
			Memo: txnbuild.MemoText(fmt.Sprintf("twin_%d", twinID)),
		},
	)
	if err != nil {
		return fmt.Errorf("failed to build the txn: %w", err)
	}

	tx, err = tx.Sign(netPassphrase, sourceKP)
	if err != nil {
		return fmt.Errorf("failed to sign the txn: %w", err)
	}

	_, err = strClient.SubmitTransaction(tx)
	if err != nil {
		errMsg := getHorizonError(err)
		return fmt.Errorf("failed to submit the txn: %s", errMsg)
	}

	return nil
}

func (m *Monitor) sendToStellar(identity client.Identity, net network) error {
	conn, err := m.managers[net].Substrate()
	if err != nil {
		return fmt.Errorf("failed to create substrate connection for %s: %w", net, err)
	}
	defer conn.Close()

	strAddress := m.env.testStellarAddress
	if net == mainNetwork || net == testNetwork {
		strAddress = m.env.publicStellarAddress
	}

	if err := conn.SwapToStellar(identity, strAddress, *big.NewInt(int64(bridgeTestTFTAmount * 10000000))); err != nil {
		return fmt.Errorf("failed to send %d TFT to stellar: %w", bridgeTestTFTAmount, err)
	}

	return nil
}

func getHorizonError(err error) string {
	errMsg := ""
	if p := horizonclient.GetError(err); p != nil {
		errMsg += fmt.Sprintf("  Info: %s\n", p.Problem)
		if results, ok := p.Problem.Extras["result_codes"]; ok {
			errMsg += fmt.Sprintf("  Extras: %s\n", results)
		}
	}
	return errMsg
}
