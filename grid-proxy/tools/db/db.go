package main

import (
	"database/sql"
	"flag"
	"fmt"

	// used by the orm

	_ "github.com/lib/pq"
	"github.com/pkg/errors"
)

type flags struct {
	postgresHost     string
	postgresPort     int
	postgresDB       string
	postgresUser     string
	postgresPassword string
	reset            bool
	seed             int
}

func parseCmdline() flags {
	f := flags{}
	flag.StringVar(&f.postgresHost, "postgres-host", "", "postgres host")
	flag.IntVar(&f.postgresPort, "postgres-port", 5432, "postgres port")
	flag.StringVar(&f.postgresDB, "postgres-db", "", "postgres database")
	flag.StringVar(&f.postgresUser, "postgres-user", "", "postgres username")
	flag.StringVar(&f.postgresPassword, "postgres-password", "", "postgres password")
	flag.BoolVar(&f.reset, "reset", false, "reset the db before starting")
	flag.IntVar(&f.seed, "seed", 0, "seed used for the random generation of tests")
	flag.Parse()
	return f
}

func main() {
	f := parseCmdline()

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		f.postgresHost, f.postgresPort, f.postgresUser, f.postgresPassword, f.postgresDB)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(errors.Wrap(err, "failed to open db"))
	}
	defer db.Close()

	if f.reset {
		if err := reset(db); err != nil {
			panic(err)
		}
	}

	if err := initSchema(db); err != nil {
		panic(err)
	}

	// it looks like a useless block but everything breaks when it's removed
	_, err = db.Query("SELECT current_database();")
	if err != nil {
		panic(err)
	}
	// ----

	if err := generateData(db, f.seed); err != nil {
		panic(err)
	}
}
