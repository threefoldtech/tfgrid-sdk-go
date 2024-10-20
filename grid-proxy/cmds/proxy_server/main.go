package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	substrate "github.com/threefoldtech/tfchain/clients/tfchain-client-go"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/certmanager"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/indexer"
	logging "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	rmb "github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
	"gorm.io/gorm/logger"
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
	debug                  string
	postgresHost           string
	postgresPort           int
	postgresDB             string
	postgresUser           string
	postgresPassword       string
	sqlLogLevel            int
	address                string
	version                bool
	nocert                 bool
	domain                 string
	TLSEmail               string
	CA                     string
	certCacheDir           string
	tfChainURL             string
	relayURL               string
	mnemonics              string
	maxPoolOpenConnections int

	noIndexer                    bool // true to stop the indexer, useful on running for testing
	indexerUpserterBatchSize     uint
	gpuIndexerIntervalMins       uint
	gpuIndexerNumWorkers         uint
	healthIndexerNumWorkers      uint
	healthIndexerIntervalMins    uint
	dmiIndexerNumWorkers         uint
	dmiIndexerIntervalMins       uint
	speedIndexerNumWorkers       uint
	speedIndexerIntervalMins     uint
	ipv6IndexerNumWorkers        uint
	ipv6IndexerIntervalMins      uint
	workloadsIndexerNumWorkers   uint
	workloadsIndexerIntervalMins uint
	featuresIndexerNumWorkers    uint
	featuresIndexerIntervalMins  uint
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
	flag.IntVar(&f.sqlLogLevel, "sql-log-level", 2, "sql logger level")
	flag.BoolVar(&f.version, "v", false, "shows the package version")
	flag.BoolVar(&f.nocert, "no-cert", false, "start the server without certificate")
	flag.StringVar(&f.domain, "domain", "", "domain on which the server will be served")
	flag.StringVar(&f.TLSEmail, "email", "", "tmail address to generate certificate with")
	flag.StringVar(&f.CA, "ca", "https://acme-v02.api.letsencrypt.org/directory", "certificate authority used to generate certificate")
	flag.StringVar(&f.certCacheDir, "cert-cache-dir", CertDefaultCacheDir, "path to store generated certs in")
	flag.StringVar(&f.tfChainURL, "tfchain-url", DefaultTFChainURL, "TF chain url")
	flag.StringVar(&f.relayURL, "relay-url", DefaultRelayURL, "RMB relay url")
	flag.StringVar(&f.mnemonics, "mnemonics", "", "Dummy user mnemonics for relay calls")
	flag.IntVar(&f.maxPoolOpenConnections, "max-open-conns", 80, "max number of db connection pool open connections")

	flag.BoolVar(&f.noIndexer, "no-indexer", false, "do not start the indexer")
	flag.UintVar(&f.indexerUpserterBatchSize, "indexer-upserter-batch-size", 20, "results batch size which collected before upserting")
	flag.UintVar(&f.gpuIndexerIntervalMins, "gpu-indexer-interval", 60, "the interval that the GPU indexer will run")
	flag.UintVar(&f.gpuIndexerNumWorkers, "gpu-indexer-workers", 100, "number of workers to process indexer GPU info")
	flag.UintVar(&f.healthIndexerIntervalMins, "health-indexer-interval", 5, "node health check interval in min")
	flag.UintVar(&f.healthIndexerNumWorkers, "health-indexer-workers", 100, "number of workers checking on node health")
	flag.UintVar(&f.dmiIndexerIntervalMins, "dmi-indexer-interval", 60*24, "node dmi check interval in min")
	flag.UintVar(&f.dmiIndexerNumWorkers, "dmi-indexer-workers", 1, "number of workers checking on node dmi")
	flag.UintVar(&f.speedIndexerIntervalMins, "speed-indexer-interval", 5, "node speed check interval in min")
	flag.UintVar(&f.speedIndexerNumWorkers, "speed-indexer-workers", 100, "number of workers checking on node speed")
	flag.UintVar(&f.ipv6IndexerIntervalMins, "ipv6-indexer-interval", 60*24, "node ipv6 check interval in min")
	flag.UintVar(&f.ipv6IndexerNumWorkers, "ipv6-indexer-workers", 10, "number of workers checking on node having ipv6")
	flag.UintVar(&f.workloadsIndexerIntervalMins, "workloads-indexer-interval", 60, "node workloads check interval in min")
	flag.UintVar(&f.workloadsIndexerNumWorkers, "workloads-indexer-workers", 10, "number of workers checking on node workloads number")
	flag.UintVar(&f.featuresIndexerIntervalMins, "features-indexer-interval", 60*24, "node features check interval in min")
	flag.UintVar(&f.featuresIndexerNumWorkers, "features-indexer-workers", 10, "number of workers checking on node supported features")
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

	db, err := db.NewPostgresDatabase(f.postgresHost, f.postgresPort, f.postgresUser, f.postgresPassword, f.postgresDB, f.maxPoolOpenConnections, logger.LogLevel(f.sqlLogLevel))
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't get postgres client")
	}

	if err := db.Initialize(); err != nil {
		log.Fatal().Err(err).Msg("failed to initialize database")
	}

	dbClient := explorer.DBClient{DB: &db}
	rpcRmbClient, err := createRPCRMBClient(ctx, f.relayURL, f.mnemonics, subManager)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create relay client")
	}

	indexerIntervals := make(map[string]uint)
	if !f.noIndexer {
		startIndexers(ctx, f, &db, rpcRmbClient)
		indexerIntervals["gpu"] = f.gpuIndexerIntervalMins
		indexerIntervals["health"] = f.healthIndexerIntervalMins
		indexerIntervals["dmi"] = f.dmiIndexerIntervalMins
		indexerIntervals["workloads"] = f.workloadsIndexerIntervalMins
		indexerIntervals["ipv6"] = f.ipv6IndexerIntervalMins
		indexerIntervals["speed"] = f.speedIndexerIntervalMins
		indexerIntervals["features"] = f.featuresIndexerIntervalMins
	} else {
		log.Info().Msg("Indexers did not start")
	}

	s, err := createServer(f, dbClient, GitCommit, rpcRmbClient, indexerIntervals)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create mux server")
	}

	if err := app(s, f); err != nil {
		log.Fatal().Msg(err.Error())
	}

}

func startIndexers(ctx context.Context, f flags, db db.Database, rpcRmbClient *peer.RpcClient) {
	gpuIdx := indexer.NewIndexer[types.NodeGPU](
		indexer.NewGPUWork(f.gpuIndexerIntervalMins),
		"GPU",
		db,
		rpcRmbClient,
		f.gpuIndexerNumWorkers,
	)
	gpuIdx.Start(ctx)

	healthIdx := indexer.NewIndexer[types.HealthReport](
		indexer.NewHealthWork(f.healthIndexerIntervalMins),
		"Health",
		db,
		rpcRmbClient,
		f.healthIndexerNumWorkers,
	)
	healthIdx.Start(ctx)

	dmiIdx := indexer.NewIndexer[types.Dmi](
		indexer.NewDMIWork(f.dmiIndexerIntervalMins),
		"DMI",
		db,
		rpcRmbClient,
		f.dmiIndexerNumWorkers,
	)
	dmiIdx.Start(ctx)

	speedIdx := indexer.NewIndexer[types.Speed](
		indexer.NewSpeedWork(f.speedIndexerIntervalMins),
		"Speed",
		db,
		rpcRmbClient,
		f.speedIndexerNumWorkers,
	)
	speedIdx.Start(ctx)

	ipv6Idx := indexer.NewIndexer[types.HasIpv6](
		indexer.NewIpv6Work(f.ipv6IndexerIntervalMins),
		"IPV6",
		db,
		rpcRmbClient,
		f.ipv6IndexerNumWorkers,
	)
	ipv6Idx.Start(ctx)

	wlNumIdx := indexer.NewIndexer[types.NodesWorkloads](
		indexer.NewWorkloadWork(f.workloadsIndexerIntervalMins),
		"workloads",
		db,
		rpcRmbClient,
		f.workloadsIndexerNumWorkers,
	)
	wlNumIdx.Start(ctx)

	featIdx := indexer.NewIndexer[types.NodeFeatures](
		indexer.NewFeatureWork(f.featuresIndexerIntervalMins),
		"features",
		db,
		rpcRmbClient,
		f.featuresIndexerNumWorkers,
	)
	featIdx.Start(ctx)
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

func createRPCRMBClient(ctx context.Context, relayURL, mnemonics string, subManager substrate.Manager) (*peer.RpcClient, error) {
	sessionId := fmt.Sprintf("tfgrid-proxy-%s", strings.Split(uuid.NewString(), "-")[0])
	client, err := peer.NewRpcClient(ctx, mnemonics, subManager, peer.WithRelay(relayURL), peer.WithSession(sessionId))
	if err != nil {
		return nil, fmt.Errorf("failed to create direct RPC RMB client: %w", err)
	}
	return client, nil
}

func createServer(f flags, dbClient explorer.DBClient, gitCommit string, relayClient rmb.Client, idxIntervals map[string]uint) (*http.Server, error) {
	log.Info().Msg("Creating server")

	router := mux.NewRouter().StrictSlash(true)

	// setup explorer
	if err := explorer.Setup(router, gitCommit, dbClient, relayClient, idxIntervals); err != nil {
		return nil, err
	}

	return &http.Server{
		Handler:           http.TimeoutHandler(router, 30*time.Second, http.ErrHandlerTimeout.Error()), // 30 seconds for slow sql operations
		Addr:              f.address,
		ReadHeaderTimeout: 5 * time.Second,
	}, nil
}
