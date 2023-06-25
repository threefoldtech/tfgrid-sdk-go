package mock

import (
	"time"

	proxyclient "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client"
)

const (
	nodeUpInterval        = -80 * time.Minute
	nodeStateFactor int64 = 3
	reportInterval        = time.Hour
)

// GridProxyMockClient client that returns data directly from the db
type GridProxyMockClient struct {
	data DBData
}

type Matcher interface{
	Satisfy(filter interface{}) bool
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
