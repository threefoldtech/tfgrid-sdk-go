// Package integration for integration tests
package integration

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContractsGetter(t *testing.T) {
	tfPluginClient, err := setup()
	require.NoError(t, err)

	_, err = tfPluginClient.ContractsGetter.ListContractsByTwinID([]string{"Created, GracePeriod"})
	require.NoError(t, err)

	contracts, err := tfPluginClient.ContractsGetter.ListContractsOfProjectName("badName")
	assert.Empty(t, contracts.NameContracts)
	assert.Empty(t, contracts.NodeContracts)
	assert.Empty(t, contracts.RentContracts)
	assert.NoError(t, err)
}
