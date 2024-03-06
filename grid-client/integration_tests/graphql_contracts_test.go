// Package integration for integration tests
package integration

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContractsGetter(t *testing.T) {
	tfPluginClient, err := setup()
	require.NoError(t, err)

	_, err = tfPluginClient.ContractsGetter.ListContractsByTwinID([]string{"Created, GracePeriod"})
	require.NoError(t, err)

	contracts, err := tfPluginClient.ContractsGetter.ListContractsOfProjectName("badName")
	require.Empty(t, contracts.NameContracts)
	require.Empty(t, contracts.NodeContracts)
	require.Empty(t, contracts.RentContracts)
	require.NoError(t, err)
}
