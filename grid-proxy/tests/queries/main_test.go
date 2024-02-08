package test

import (
	"database/sql"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"testing"

	// used by the orm

	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	proxyDB "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	proxyclient "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client"
	mock "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/tests/queries/mock_client"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/tools/db/crafter"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	POSTGRES_HOST      string
	POSTGRES_PORT      int
	POSTGRES_USER      string
	POSTGRES_PASSSWORD string
	POSTGRES_DB        string
	ENDPOINT           string
	SEED               int
	STATUS_DOWN        = "down"
	STATUS_UP          = "up"
	NO_MODIFY          = false

	mockClient      proxyclient.Client
	data            mock.DBData
	gridProxyClient proxyclient.Client
	DBClient        db.Database
)

func parseCmdline() {
	flag.StringVar(&POSTGRES_HOST, "postgres-host", "", "postgres host")
	flag.IntVar(&POSTGRES_PORT, "postgres-port", 5432, "postgres port")
	flag.StringVar(&POSTGRES_DB, "postgres-db", "", "postgres database")
	flag.StringVar(&POSTGRES_USER, "postgres-user", "", "postgres username")
	flag.StringVar(&POSTGRES_PASSSWORD, "postgres-password", "", "postgres password")
	flag.StringVar(&ENDPOINT, "endpoint", "", "the grid proxy endpoint to test against")
	flag.IntVar(&SEED, "seed", 0, "seed used for the random generation of tests")
	flag.BoolVar(&NO_MODIFY, "no-modify", false, "stop modify the dump data")
	flag.Parse()
}

func TestMain(m *testing.M) {
	var exitCode int

	parseCmdline()
	if SEED != 0 {
		rand.New(rand.NewSource(int64(SEED)))
	}

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSSWORD, POSTGRES_DB)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(errors.Wrap(err, "failed to open db"))
	}
	gormDB, err := gorm.Open(postgres.Open(psqlInfo), &gorm.Config{
		Logger: logger.Default.LogMode(4),
	})
	if err != nil {
		panic(fmt.Errorf("failed to generate gorm db: %w", err))
	}
	defer func() {
		db.Close()
		db_gorm, err := gormDB.DB()
		if err != nil {
			panic(fmt.Errorf("failed to get gorm db: %w", err))
		}
		db_gorm.Close()
	}()

	// proxy client
	gridProxyClient = proxyclient.NewClient(ENDPOINT)

	// mock client
	dbClient, err := proxyDB.NewPostgresDatabase(POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSSWORD, POSTGRES_DB, 80, logger.Error)
	if err != nil {
		panic(err)
	}
	DBClient = &dbClient

	// load mock client
	data, err = mock.Load(db)
	if err != nil {
		panic(err)
	}

	if !NO_MODIFY {
		err = modifyDataToFireTriggers(db, gormDB, data)
		if err != nil {
			panic(err)
		}
		data, err = mock.Load(db)
		if err != nil {
			panic(err)
		}
	}

	mockClient = mock.NewGridProxyMockClient(data)
	exitCode = m.Run()
	os.Exit(exitCode)
}

func modifyDataToFireTriggers(db *sql.DB, gormDB *gorm.DB, data mock.DBData) error {
	twinStart := len(data.Twins) + 1
	farmStart := len(data.Farms) + 1
	nodeStart := len(data.Nodes) + 1
	contractStart := len(data.NodeContracts) + len(data.RentContracts) + len(data.NameContracts) + 1
	billStart := data.BillReports + 1
	publicIpStart := len(data.PublicIPs) + 1

	const (
		NodeCount         = 600
		FarmCount         = 100
		TwinCount         = 600 + 100 + 600 // nodes + farms + normal users
		PublicIPCount     = 10
		NodeContractCount = 50
		NameContractCount = 10
		RentContractCount = 1
	)

	generator := crafter.NewCrafter(db, gormDB,
		SEED,
		NodeCount,
		FarmCount,
		TwinCount,
		PublicIPCount,
		NodeContractCount,
		NameContractCount,
		RentContractCount,
		uint(nodeStart),
		uint(farmStart),
		uint(twinStart),
		uint(contractStart),
		uint(billStart),
		uint(publicIpStart))

	// insertion
	if err := generator.GenerateTwins(); err != nil {
		return fmt.Errorf("failed to generate twins: %w", err)
	}

	if err := generator.GenerateFarms(); err != nil {
		return fmt.Errorf("failed to generate farms: %w", err)
	}

	if err := generator.GenerateNodes(); err != nil {
		return fmt.Errorf("failed to generate nodes: %w", err)
	}

	// rentCount is 1 because the generate method have .1 percent of 10 farms to be dedicated
	if err := generator.GenerateContracts(); err != nil {
		return fmt.Errorf("failed to generate contracts: %w", err)
	}

	if err := generator.GeneratePublicIPs(); err != nil {
		return fmt.Errorf("failed to generate public ips: %w", err)
	}

	// updates
	if err := generator.UpdateNodeCountry(); err != nil {
		return fmt.Errorf("failed to update node country: %w", err)
	}

	if err := generator.UpdateNodeTotalResources(); err != nil {
		return fmt.Errorf("failed to update node total resources: %w", err)
	}

	if err := generator.UpdateContractResources(); err != nil {
		return fmt.Errorf("failed to update contract resources: %w", err)
	}

	if err := generator.UpdateNodeContractState(); err != nil {
		return fmt.Errorf("failed to update node node contract: %w", err)
	}

	if err := generator.UpdateRentContract(); err != nil {
		return fmt.Errorf("failed to update rent contract: %w", err)
	}

	if err := generator.UpdatePublicIps(); err != nil {
		return fmt.Errorf("failed to update public ips: %w", err)
	}

	// deletions
	if err := generator.DeleteNodes(); err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	if err := generator.DeletePublicIps(); err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	return nil
}
