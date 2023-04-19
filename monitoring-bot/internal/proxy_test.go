// Package internal contains all logic for monitoring service
package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProxy(t *testing.T) {
	t.Run("test_success", func(t *testing.T) {
		gridProxy, err := NewGridProxyClient(ProxyUrls[devNetwork])
		assert.NoError(t, err)

		err = gridProxy.Ping()
		assert.NoError(t, err)
	})

	t.Run("test_wrong_endpoint", func(t *testing.T) {
		gridProxy, err := NewGridProxyClient("wrong")
		assert.NoError(t, err)

		err = gridProxy.Ping()
		assert.Error(t, err)
	})
}
