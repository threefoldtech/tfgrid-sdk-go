package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/certmanager"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/gpuindexer"
	logging "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg"
	rmb "github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

const (
	// CertDefaultCacheDir directory to keep the genreated certificates
	CertDefaultCacheDir = "/tmp/certs"
	DefaultTFChainURL   = "wss://tfchain.dev.grid.tf/ws"
	DefaultRelayURL     = "wss://relay.dev.grid.tf"
)

// GitCommit holds the commit version
var GitCommit string

type flags struct {
	debug                    string
	postgresHost             string
	postgresPort             int
	postgresDB               string
	postgresUser             string
	postgresPassword         string
	address                  string
	version                  bool
	nocert                   bool
	domain                   string
	TLSEmail                 string
	CA                       string
	certCacheDir             string
	tfChainURL               string
	relayURL                 string
	mnemonics                string
	indexerCheckIntervalMins int
	indexerBatchSize         int
	indexerResultWorkers     int
	indexerBatchWorkers      int
	maxPoolOpenConnections   int
}

func main() {
	f := flags{}
	flag.StringVar(&f.debug, "log-level", "info", "log level [debug|info|warn|error|fatal|panic]")
	flag.StringVar(&f.address, "address", ":443", "explorer running ip address")
	flag.StringVar(&f.postgresHost, "postgres-host", "", "postgres host")
	flag.IntVar(&f.postgresPort, "postgres-port", 5432, "postgres port")
	flag.StringVar(&f.postgresDB, "postgres-db", "", "postgres database")
	flag.StringVar(&f.postgresUser, "postgres-user", "", "postgres username")
	flag.StringVar(&f.postgresPassword, "postgres-password", "", "postgres password")
	flag.BoolVar(&f.version, "v", false, "shows the package version")
	flag.BoolVar(&f.nocert, "no-cert", false, "start the server without certificate")
	flag.StringVar(&f.domain, "domain", "", "domain on which the server will be served")
	flag.StringVar(&f.TLSEmail, "email", "", "tmail address to generate certificate with")
	flag.StringVar(&f.CA, "ca", "https://acme-v02.api.letsencrypt.org/directory", "certificate authority used to generate certificate")
	flag.StringVar(&f.certCacheDir, "cert-cache-dir", CertDefaultCacheDir, "path to store generated certs in")
	flag.StringVar(&f.tfChainURL, "tfchain-url", DefaultTFChainURL, "TF chain url")
	flag.StringVar(&f.relayURL, "relay-url", DefaultRelayURL, "RMB relay url")
	flag.StringVar(&f.mnemonics, "mnemonics", "", "Dummy user mnemonics for relay calls")
	flag.IntVar(&f.indexerCheckIntervalMins, "indexer-interval-min", 60, "the interval that the GPU indexer will run")
	flag.IntVar(&f.indexerBatchSize, "indexer-batch-size", 20, "batch size for the GPU indexer worker batch")
	flag.IntVar(&f.indexerResultWorkers, "indexer-results-workers", 2, "number of workers to process indexer GPU info")
	flag.IntVar(&f.indexerBatchWorkers, "indexer-batch-workers", 2, "number of workers to process batch GPU info")
	flag.IntVar(&f.maxPoolOpenConnections, "max-open-conns", 80, "max number of db connection pool open connections")
	flag.Parse()

	// shows version and exit
	if f.version {
		fmt.Printf("git rev: %s\n", GitCommit)
		os.Exit(0)
	}

	if f.domain == "" {
		log.Fatal().Err(errors.New("domain is required"))
	}
	if f.TLSEmail == "" {
		log.Fatal().Err(errors.New("email is required"))
	}
	if f.mnemonics == "" {
		log.Fatal().Msg("mnemonics are required")
	}
	logging.SetupLogging(f.debug)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subManager := substrate.NewManager(f.tfChainURL)

	relayRPCClient, err := createRPCRMBClient(ctx, f.relayURL, f.mnemonics, subManager)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create relay client")
	}

	db, err := db.NewPostgresDatabase(f.postgresHost, f.postgresPort, f.postgresUser, f.postgresPassword, f.postgresDB, f.maxPoolOpenConnections)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't get postgres client")
	}

	dbClient := explorer.DBClient{DB: db}

	indexer, err := gpuindexer.NewNodeGPUIndexer(
		ctx,
		f.relayURL,
		f.mnemonics,
		subManager, db,
		f.indexerCheckIntervalMins,
		f.indexerBatchSize,
		f.indexerResultWorkers,
		f.indexerBatchWorkers,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create GPU indexer")
	}

	indexer.Start(ctx)

	s, err := createServer(f, dbClient, GitCommit, relayRPCClient)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create mux server")
	}

	if err := app(s, f); err != nil {
		log.Fatal().Msg(err.Error())
	}

}

func app(s *http.Server, f flags) error {

	if f.nocert {
		log.Info().Str("listening on", f.address).Msg("Server started ...")
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
		Domain:   f.domain,
		Email:    f.TLSEmail,
		CA:       f.CA,
		CacheDir: f.certCacheDir,
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

	log.Info().Str("listening on", f.address).Msg("Server started ...")
	if err := s.ListenAndServeTLS("", ""); err != nil {
		if err == http.ErrServerClosed {
			log.Info().Msg("server stopped gracefully")
		} else {
			log.Error().Err(err).Msg("server stopped unexpectedly")
		}
	}
	return nil
}

func createRPCRMBClient(ctx context.Context, relayURL, mnemonics string, subManager substrate.Manager) (rmb.Client, error) {
	sessionId := fmt.Sprintf("tfgrid_proxy-%d", os.Getpid())
	client, err := peer.NewRpcClient(ctx, peer.KeyTypeSr25519, mnemonics, relayURL, sessionId, subManager, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create direct RPC RMB client: %w", err)
	}
	return client, nil
}

func createServer(f flags, dbClient explorer.DBClient, gitCommit string, relayClient rmb.Client) (*http.Server, error) {
	log.Info().Msg("Creating server")

	router := mux.NewRouter().StrictSlash(true)

	// setup explorer
	if err := explorer.Setup(router, gitCommit, dbClient, relayClient); err != nil {
		return nil, err
	}

	return &http.Server{
		Handler:           http.TimeoutHandler(router, 30*time.Second, "request timed-out. server took too long to respond"), // 30 seconds for slow sql operations
		Addr:              f.address,
		ReadHeaderTimeout: 5 * time.Second,
	}, nil
}
