package mock

import (
	proxyclient "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client"
)

// GridProxyMockClient client that returns data directly from the db
type GridProxyMockClient struct {
	data DBData
}

// NewGridProxyMockClient local grid proxy client constructor
func NewGridProxyMockClient(data DBData) proxyclient.Client {
	proxy := GridProxyMockClient{data}
	return &proxy
}

// Ping makes sure the server is up
func (g *GridProxyMockClient) Ping() error {
	return nil
}
