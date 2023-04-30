package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/substrate-client"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/certmanager"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	logging "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/direct"
)

// GitCommit holds the commit version
var GitCommit string

type flags struct {
	LogLevel         string `arg:"--log-level,env" help:"log level [debug|info|warn|error|fatal|panic]" default:"info"`
	PostgresHost     string `arg:"--postgres-host,env:POSTGRES_HOST" help:"postgres host"`
	PostgresPort     int    `arg:"--postgres-port,env:POSTGRES_PORT" help:"postgres port" default:"5432"`
	PostgresDB       string `arg:"--postgres-db,env:POSTGRES_DB" help:"postgres database"`
	PostgresUser     string `arg:"--postgres-user,env:POSTGRES_USER" help:"postgres username"`
	PostgresPassword string `arg:"--postgres-password,env:POSTGRES_PASSWORD" help:"postgres password"`
	Address          string `arg:"env:SERVER_PORT" help:"explorer running ip address" default:":443"`
	Version          bool   `arg:"-v,env" help:"shows the package version" default:"false"`
	Nocert           bool   `arg:"--no-cert,env" help:"start the server without certificate" default:"false"`
	Domain           string `arg:"env" help:"domain on which the server will be served"`
	TLSEmail         string `arg:"--email,env" help:"email address to generate certificate with"`
	CA               string `arg:"env" help:"ertificate authority used to generate certificate" default:"https://acme-v02.api.letsencrypt.org/directory"`
	CertCacheDir     string `arg:"--cert-cache-dir,env" help:"path to store generated certs in" default:"/tmp/certs"`
	TfChainURL       string `arg:"--tfchain-url,env" help:"TF chain url" default:"wss://tfchain.dev.grid.tf/ws"`
	RelayURL         string `arg:"--relay-url,env" help:"RMB relay url" default:"wss://relay.dev.grid.tf"`
	Mnemonics        string `arg:"env,required" help:"Dummy user mnemonics for relay calls"`
}

func main() {
	f := flags{}
	arg.MustParse(&f)

	fmt.Printf("%+v", f)

	// shows version and exit
	if f.Version {
		fmt.Printf("git rev: %s\n", GitCommit)
		os.Exit(0)
	}
	logging.SetupLogging(f.LogLevel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subManager := substrate.NewManager(f.TfChainURL)
	sub, err := subManager.Substrate()
	if err != nil {
		log.Fatal().Err(err).Msg(fmt.Sprintf("failed to connect to TF chain URL: %s", err))
	}
	defer sub.Close()

	relayClient, err := createRMBClient(ctx, f.RelayURL, f.Mnemonics, sub)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create realy client")
	}

	s, err := createServer(f, GitCommit, relayClient)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create mux server")
	}

	if err := app(s, f); err != nil {
		log.Fatal().Msg(err.Error())
	}

}

func app(s *http.Server, f flags) error {

	if f.Nocert {
		log.Info().Str("listening on", f.Address).Msg("Server started ...")
		if err := s.ListenAndServe(); err != nil {
			if err == http.ErrServerClosed {
				log.Info().Msg("server stopped gracefully")
			} else {
				log.Error().Err(err).Msg("server stopped unexpectedly")
			}
		}
		return nil
	}

	config := certmanager.CertificateConfig{
		Domain:   f.Domain,
		Email:    f.TLSEmail,
		CA:       f.CA,
		CacheDir: f.CertCacheDir,
	}
	cm := certmanager.NewCertificateManager(config)
	go func() {
		if err := cm.ListenForChallenges(); err != nil {
			log.Error().Err(err).Msg("error occurred when listening for challenges")
		}
	}()
	kpr, err := certmanager.NewKeypairReloader(cm)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initiate key reloader")
	}
	s.TLSConfig = &tls.Config{
		GetCertificate: kpr.GetCertificateFunc(),
	}

	log.Info().Str("listening on", f.Address).Msg("Server started ...")
	if err := s.ListenAndServeTLS("", ""); err != nil {
		if err == http.ErrServerClosed {
			log.Info().Msg("server stopped gracefully")
		} else {
			log.Error().Err(err).Msg("server stopped unexpectedly")
		}
	}
	return nil
}

func createRMBClient(ctx context.Context, relayURL, mnemonics string, sub *substrate.Substrate) (rmb.Client, error) {
	client, err := direct.NewClient(ctx, direct.KeyTypeSr25519, mnemonics, relayURL, "tfgrid_proxy", sub, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create direct RMB client: %w", err)
	}
	return client, nil
}

func createServer(f flags, gitCommit string, relayClient rmb.Client) (*http.Server, error) {
	log.Info().Msg("Creating server")

	router := mux.NewRouter().StrictSlash(true)
	db, err := db.NewPostgresDatabase(f.PostgresHost, f.PostgresPort, f.PostgresUser, f.PostgresPassword, f.PostgresDB)
	if err != nil {
		return nil, errors.Wrap(err, "couldn't get postgres client")
	}

	// setup explorer
	if err := explorer.Setup(router, gitCommit, db, relayClient); err != nil {
		return nil, err
	}

	return &http.Server{
		Handler: router,
		Addr:    f.Address,
	}, nil
}
