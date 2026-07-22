package db

import (
	"log"
	"os"
	"testing"

	"database/sql"

	"gorm.io/gorm/logger"
)

func TestMain(m *testing.M) {

	Setup()
	code := m.Run()
	os.Exit(code)

}
func Setup() {
	// Connect to the default `postgres` database to create `testdb`
	initialDb, err := NewPostgresDatabase("localhost", 5432, "postgres", "mypassword", "postgres", 80, logger.Error)
	if err != nil {
		log.Fatalf("could not connect to default database: %v", err)
	}
	defer initialDb.Close()
	// Force close all connections to testdb before dropping
	err = initialDb.gormDB.Exec(`
        SELECT pg_terminate_backend(pg_stat_activity.pid)
        FROM pg_stat_activity
        WHERE pg_stat_activity.datname = 'testdb'
        AND pid <> pg_backend_pid();
    `).Error
	if err != nil {
		log.Printf("warning while terminating connections: %v", err)
	}

	// drop the `testdb` database
	err = initialDb.gormDB.Exec(`DROP DATABASE IF EXISTS testdb;`).Error
	if err != nil {
		log.Fatalf("could not create testdb database: %v", err)
	}

	// Create the `testdb` database
	err = initialDb.gormDB.Exec(`CREATE DATABASE testdb;`).Error
	if err != nil {
		log.Fatalf("could not create testdb database: %v", err)
	}

	// Connect to `testdb`
	testDb, err := sql.Open("postgres", "host=localhost user=postgres dbname=testdb port=5432 password=mypassword sslmode=disable")
	if err != nil {
		log.Fatalf("could not connect to testdb: %v", err)
	}
	defer testDb.Close()

	// Load and execute schema
	schema, err := os.ReadFile("../../../tools/db/schema.sql")
	if err != nil {
		log.Fatalf("could not load schema sql file: %v", err)
	}
	_, err = testDb.Exec(string(schema))
	if err != nil {
		log.Fatalf("could not apply schema: %v", err)
	}

	// it looks like a useless block but everything breaks when it's removed
	_, err = testDb.Query("SELECT current_database();")
	if err != nil {
		log.Fatalf("%e", err)
	}

	// Load and execute schema
	setup, err := os.ReadFile("./setup.sql")
	if err != nil {
		log.Fatalf("could not load setup sql file: %v", err)
	}
	_, err = testDb.Exec(string(setup))
	if err != nil {
		log.Fatalf("could not apply setup: %v", err)
	}

	// Load and execute fixture data
	queries, err := os.ReadFile("./fixtures/testdata.sql")
	if err != nil {
		log.Fatalf("could not load query sql file: %v", err)
	}
	_, err = testDb.Exec(string(queries))
	if err != nil {
		log.Fatalf("could not populate db: %v", err)
	}
}
