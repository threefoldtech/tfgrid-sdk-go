// Package integration for integration tests
package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContractsGetter(t *testing.T) {
	tfPluginClient, err := setup()
	if !assert.NoError(t, err) {
		return
	}

	_, err = tfPluginClient.ContractsGetter.ListContractsByTwinID([]string{"Created, GracePeriod"})
	if !assert.NoError(t, err) {
		return
	}

	contracts, err := tfPluginClient.ContractsGetter.ListContractsOfProjectName("badName")
	assert.Empty(t, contracts.NameContracts)
	assert.Empty(t, contracts.NodeContracts)
	assert.Empty(t, contracts.RentContracts)
	assert.NoError(t, err)
}
