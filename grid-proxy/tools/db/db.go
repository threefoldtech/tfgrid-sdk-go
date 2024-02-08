package main

import (
	"database/sql"
	"flag"
	"fmt"

	// used by the orm

	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
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

	if err := generateData(db, gormDB, f.seed); err != nil {
		panic(err)
	}
}
