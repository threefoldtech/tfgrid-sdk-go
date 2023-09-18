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

type bridgeTX func(*client.Substrate, client.Identity, network) error

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
	conn, err := m.substrate[net].Substrate()
	if err != nil {
		return balanceReport{}, fmt.Errorf("failed to create substrate connection for %s: %w", net, err)
	}
	defer conn.Close()

	identity, err := client.NewIdentityFromSr25519Phrase(m.mnemonics[net])
	if err != nil {
		return balanceReport{}, err
	}

	// get current balance
	originalBalance, err := m.getBalance(conn, address(identity.Address()))
	if err != nil {
		return balanceReport{}, fmt.Errorf("failed to get balance for account ")
	}

	balanceAfterChain, err := m.bridgeTXWrapper(m.sendToTfChain)(conn, identity, net)
	if err != nil {
		return balanceReport{}, err
	}

	balanceAfterStellar, err := m.bridgeTXWrapper(m.sendToStellar)(conn, identity, net)
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
func (m *Monitor) bridgeTXWrapper(tx bridgeTX) func(conn *client.Substrate, identity client.Identity, net network) (float64, error) {
	return func(conn *client.Substrate, identity client.Identity, net network) (float64, error) {
		if err := tx(conn, identity, net); err != nil {
			return 0, err
		}

		<-time.After(balanceWaitIntervalSeconds * time.Second)

		balanceAfterChain, err := m.getBalance(conn, address(identity.Address()))
		if err != nil {
			return 0, fmt.Errorf("failed to get balance for account ")
		}

		return balanceAfterChain, nil
	}
}

func (m *Monitor) sendToTfChain(conn *client.Substrate, identity client.Identity, tfChain network) error {
	twinID, err := conn.GetTwinByPubKey(identity.PublicKey())
	if err != nil {
		return fmt.Errorf("failed to get twinId: %w", err)
	}

	strSecret := m.env.testStellarSecret
	stellarTFTIssuerAddress := tftIssuerStellarTest
	if tfChain == mainNetwork || tfChain == testNetwork {
		strSecret = m.env.publicStellarSecret
		stellarTFTIssuerAddress = tftIssuerStellarPublic
	}

	tftTrustLine := txnbuild.CreditAsset{Code: "TFT", Issuer: stellarTFTIssuerAddress}
	strClient := horizonclient.DefaultTestNetClient
	destAccountRequest := horizonclient.AccountRequest{AccountID: BridgeAddresses[tfChain]}

	_, err = strClient.AccountDetail(destAccountRequest)
	if err != nil {
		return fmt.Errorf("failed to verify destination account: %w", err)
	}

	sourceKP := keypair.MustParseFull(strSecret)
	sourceAccountRequest := horizonclient.AccountRequest{AccountID: sourceKP.Address()}
	sourceAccount, err := strClient.AccountDetail(sourceAccountRequest)
	if err != nil {
		return fmt.Errorf("failed to load source account: %w", err)
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
					Destination: BridgeAddresses[tfChain],
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

	netPassphrase := strNet.TestNetworkPassphrase
	if tfChain == mainNetwork || tfChain == testNetwork {
		netPassphrase = strNet.PublicNetworkPassphrase
	}

	tx, err = tx.Sign(netPassphrase, sourceKP)
	if err != nil {
		return fmt.Errorf("failed to sign the txn: %w", err)
	}

	_, err = horizonclient.DefaultTestNetClient.SubmitTransaction(tx)
	if err != nil {
		return fmt.Errorf("failed to submit the txn: %w", err)
	}

	return nil
}

func (m *Monitor) sendToStellar(conn *client.Substrate, identity client.Identity, tfChain network) error {
	strAddress := m.env.testStellarAddress
	if tfChain == mainNetwork || tfChain == testNetwork {
		strAddress = m.env.publicStellarAddress
	}

	if err := conn.SwapToStellar(identity, strAddress, *big.NewInt(int64(bridgeTestTFTAmount * 10000000))); err != nil {
		return fmt.Errorf("failed to send %d TFT to stellar: %w", bridgeTestTFTAmount, err)
	}

	return nil
}
