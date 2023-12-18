package deployer

import (
	"fmt"
)

func ExampleNewTFPluginClient() {
	mnemonic := "<mnemonics goes here>"
	network := "<dev, test, qa, main>"

	tfPluginClient, err := NewTFPluginClient(mnemonic, "sr25519", network, "", "", "", 0, false)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(tfPluginClient)
}

func (t *TFPluginClient) ExampleBatchCancelContract() {
	mnemonic := "<mnemonics goes here>"
	network := "<dev, test, qa, main>"

	tfPluginClient, err := NewTFPluginClient(mnemonic, "sr25519", network, "", "", "", 0, false)
	if err != nil {
		fmt.Println(err)
		return
	}

	// list of contracts ids
	contracts := []uint64{}
	err = tfPluginClient.BatchCancelContract(contracts)
	if err != nil {
		fmt.Println(err)
		return
	}
}
