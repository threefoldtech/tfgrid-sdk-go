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
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/tools/db/modifiers"
	"gorm.io/gorm/logger"
	// _ "embed"
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

	mockClient      proxyclient.Client
	data            mock.DBData
	gridProxyClient proxyclient.Client
	DBClient        db.Database
)

// //go:embed modifiers.sql
// var modifiersFile string

func parseCmdline() {
	flag.StringVar(&POSTGRES_HOST, "postgres-host", "", "postgres host")
	flag.IntVar(&POSTGRES_PORT, "postgres-port", 5432, "postgres port")
	flag.StringVar(&POSTGRES_DB, "postgres-db", "", "postgres database")
	flag.StringVar(&POSTGRES_USER, "postgres-user", "", "postgres username")
	flag.StringVar(&POSTGRES_PASSSWORD, "postgres-password", "", "postgres password")
	flag.StringVar(&ENDPOINT, "endpoint", "", "the grid proxy endpoint to test against")
	flag.IntVar(&SEED, "seed", 0, "seed used for the random generation of tests")
	flag.Parse()
}

func TestMain(m *testing.M) {
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
	defer db.Close()

	data, err = mock.Load(db)
	if err != nil {
		panic(err)
	}
	dbClient, err := proxyDB.NewPostgresDatabase(POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSSWORD, POSTGRES_DB, 80, logger.Error)
	if err != nil {
		panic(err)
	}
	DBClient = &dbClient

	mockClient = mock.NewGridProxyMockClient(data)
	gridProxyClient = proxyclient.NewClient(ENDPOINT)

	exitcode := m.Run()
	if exitcode != 0 {
		os.Exit(exitcode)
	}

	// err = modifyDataToFireTriggers(db)
	// if err != nil {
	// 	panic(err)
	// }

	// data, err = mock.Load(db)
	// if err != nil {
	// 	panic(err)
	// }
	// mockClient = mock.NewGridProxyMockClient(data)

	// exitcode = m.Run()

	// cleanup modified data
	// os.Exit(exitcode)
}

func modifyDataToFireTriggers(db *sql.DB) error {
	/*
		- insert nodes - y
			- should be on new/old farms
			- should be on new/old locations
		- insert node total resources - y
		- insert node contracts - y
		- insert contract resources - y
		- insert rent contracts - y
		- insert public ips - y

		- update node country - y
		- update node total resources - y
		- update contract_resources - y
		- update node contract state - y
		- update rent contract state - y
		- update public ip contract id - y

		- delete node - y
		- delete public ip - y
	*/
	// modifiers.GenerateTwins(db)

	generator := modifiers.NewGenerator(db, SEED)

	// insertion
	// if err := generator.GenerateTwins(); err != nil {
	// 	return fmt.Errorf("failed to genrate twins: %w", err)
	// }

	// if err := generator.GenerateFarms(); err != nil {
	// 	return fmt.Errorf("failed to generate farms: %w", err)
	// }

	// if err := generator.GenerateNodes(); err != nil {
	// 	return fmt.Errorf("failed to generate nodes: %w", err)
	// }

	// if err := generator.GenerateContracts(); err != nil {
	// 	return fmt.Errorf("failed to generate contracts: %w", err)
	// }

	// if err := generator.GeneratePublicIPs(); err != nil {
	// 	return fmt.Errorf("failed to generate public ips: %w", err)
	// }

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
	if err := generator.DeleteNode(); err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	if err := generator.DeletePublicIps(); err != nil {
		return fmt.Errorf("failed to delete node: %w", err)
	}

	return nil
}
