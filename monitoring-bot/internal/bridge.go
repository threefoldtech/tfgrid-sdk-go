package internal

import (
	"fmt"
	"math/big"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	strNet "github.com/stellar/go/network"
	"github.com/stellar/go/txnbuild"
	client "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
)

func (m *Monitor) monitorBridges() error {
	message := "TfChain Bridges to Stellar Monitor Results:\n"
	for _, net := range networks {

		res, err := m.monitorBridge(net)
		if err != nil {
			message += fmt.Sprintf("\n - %s\n%s\n\n", res, err.Error())
			fmt.Println(err.Error())
			continue
		}
		message += fmt.Sprintf("\n - %s\n\n", res)
	}

	if err := m.sendBotMessage(message); err != nil {
		return err
	}

	return nil
}

func (m *Monitor) monitorBridge(net network) (string, error) {
	failureMessage := fmt.Sprintf("Bridge for %v is not working ❌", net)
	err := m.sendToTfChain(net)
	if err != nil {
		return failureMessage, err
	}

	err = m.sendToStellar(net)
	if err != nil {
		return failureMessage, err
	}

	return fmt.Sprintf("Bridge for %v is working ✅", net), nil
}

func (m *Monitor) sendToTfChain(tfChain network) error {
	conn, err := m.substrate[tfChain].Substrate()
	if err != nil {
		return fmt.Errorf("failed to get substrate connection for %s: %w", tfChain, err)
	}
	defer conn.Close()

	identity, err := client.NewIdentityFromSr25519Phrase(m.mnemonics[tfChain])
	if err != nil {
		return fmt.Errorf("failed to get identity from mnemonic: %w", err)
	}

	twinID, err := conn.GetTwinByPubKey(identity.PublicKey())
	if err != nil {
		return fmt.Errorf("failed to get twinId: %w", err)
	}

	tftTrustLine := txnbuild.CreditAsset{Code: "TFT", Issuer: tftIssuerAddress}
	strClient := horizonclient.DefaultTestNetClient
	destAccountRequest := horizonclient.AccountRequest{AccountID: BridgeAddresses[tfChain]}

	_, err = strClient.AccountDetail(destAccountRequest)
	if err != nil {
		return fmt.Errorf("failed to verify destination account: %w", err)
	}

	strSecret := m.env.testStellarSecret
	if tfChain == mainNetwork || tfChain == testNetwork {
		strSecret = m.env.publicStellarSecret
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

func (m *Monitor) sendToStellar(tfChain network) error {
	conn, err := m.substrate[tfChain].Substrate()
	if err != nil {
		return fmt.Errorf("failed to get substrate connection for %s: %w", tfChain, err)
	}
	defer conn.Close()

	identity, err := client.NewIdentityFromSr25519Phrase(m.mnemonics[tfChain])
	if err != nil {
		return fmt.Errorf("failed to get identity from mnemonic: %w", err)
	}

	strAddress := m.env.testStellarAddress
	if tfChain == mainNetwork || tfChain == testNetwork {
		strAddress = m.env.publicStellarAddress
	}

	err = conn.SwapToStellar(identity, strAddress, *big.NewInt(int64(bridgeTestTFTAmount * 10000000)))
	if err != nil {
		return fmt.Errorf("failed to send %d TFT to stellar: %w", bridgeTestTFTAmount, err)
	}

	return nil
}
