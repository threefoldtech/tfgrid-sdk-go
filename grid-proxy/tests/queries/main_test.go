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
	proxyclient "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/client"
	mock "github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/tests/queries/mock_client"
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
)

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
		rand.Seed(int64(SEED))
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

	mockClient = mock.NewGridProxyMockClient(data)
	gridProxyClient = proxyclient.NewClient(ENDPOINT)

	exitcode := m.Run()
	os.Exit(exitcode)
}
