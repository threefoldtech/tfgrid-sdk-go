// Package app for activation backend app
package app

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func setUp(t testing.TB) *App {
	dir := t.TempDir()

	configPath := filepath.Join(dir, ".env")

	// Alice mnemonic
	configs := `
	URL=wss://tfchain.dev.grid.tf
	MNEMONIC=bottom drive obey lake curtain smoke basket hold race lonely fit walk
	KYC_PUBLIC_KEY=kyc service 25119 public key
	ACTIVATION_AMOUNT=0
	`
	err := os.WriteFile(configPath, []byte(configs), 0644)
	assert.NoError(t, err)

	app, err := NewApp(context.Background(), configPath)
	assert.NoError(t, err)

	app.registerHandlers()
	return app
}

type handlerConfig struct {
	body        io.Reader
	handlerFunc Handler
	api         string
}

func handler(req handlerConfig) (response *httptest.ResponseRecorder) {
	request := httptest.NewRequest("GET", req.api, req.body)
	response = httptest.NewRecorder()

	WrapFunc(req.handlerFunc).ServeHTTP(response, request)
	return
}

func TestActivateHandler(t *testing.T) {
	app := setUp(t)

	body := []byte(`{
		"substrateAccountID": "5Fno3ccM612oKKvj8X7c1vmuYjDP3nSP7NWpvmmeedphmSfM"
	}`)

	t.Run("Activate: success", func(t *testing.T) {
		req := handlerConfig{
			body:        bytes.NewBuffer(body),
			handlerFunc: app.activateHandler,
			api:         "/activation/activate",
		}

		response := handler(req)
		assert.Equal(t, response.Code, http.StatusOK)
	})

	t.Run("Activate: invalid data", func(t *testing.T) {
		body := []byte(`{
			"substrateAccountID": ""
		}`)

		req := handlerConfig{
			body:        bytes.NewBuffer(body),
			handlerFunc: app.activateHandler,
			api:         "/activation/activate",
		}

		response := handler(req)
		assert.Equal(t, response.Code, http.StatusBadRequest)
	})

	t.Run("Activate: failed to read data", func(t *testing.T) {
		req := handlerConfig{
			body:        nil,
			handlerFunc: app.activateHandler,
			api:         "/activation/activate",
		}

		response := handler(req)
		assert.Equal(t, response.Code, http.StatusBadRequest)
	})

	t.Run("Activate: account not found", func(t *testing.T) {
		body := []byte(`{
			"substrateAccountID": "invalid"
		}`)

		req := handlerConfig{
			body:        bytes.NewBuffer(body),
			handlerFunc: app.activateHandler,
			api:         "/activation/activate",
		}

		response := handler(req)
		assert.Equal(t, response.Code, http.StatusNotFound)
	})
}
