package deployer

import (
	"fmt"
)

func ExampleNewTFPluginClient() {
	mnemonic := "<mnemonics goes here>"
	network := "<dev, test, qa, main>"

    tfPluginClient, err := NewTFPluginClient(mnemonic, WithNetwork(network))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("tfPluginClient is created successfully ", tfPluginClient)
}

func ExampleTFPluginClient_BatchCancelContract() {
	const mnemonic = "<mnemonics goes here>"
	const network = "<dev, test, qa, main>"

    tfPluginClient, err := NewTFPluginClient(mnemonic, WithNetwork(network))
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
	fmt.Println("all contracts were deleted successfully")
}
