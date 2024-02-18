package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/tools/db/crafter"
	"gorm.io/gorm"
)

const (
	NodeCount         = 6000
	FarmCount         = 600
	TwinCount         = 6000 + 600 + 6000 // nodes + farms + normal users
	PublicIPCount     = 1000
	NodeContractCount = 9000
	RentContractCount = 100
	NameContractCount = 300
)

func reset(db *sql.DB) error {
	_, err := db.Exec(
		`
		DROP TABLE IF EXISTS account CASCADE;
		DROP TABLE IF EXISTS burn_transaction CASCADE;
		DROP TABLE IF EXISTS city CASCADE;
		DROP TABLE IF EXISTS contract_bill_report CASCADE;
		DROP TABLE IF EXISTS contract_resources CASCADE;
		DROP TABLE IF EXISTS country CASCADE;
		DROP TABLE IF EXISTS entity CASCADE;
		DROP TABLE IF EXISTS entity_proof CASCADE;
		DROP TABLE IF EXISTS farm CASCADE;
		DROP TABLE IF EXISTS farming_policy CASCADE;
		DROP TABLE IF EXISTS historical_balance CASCADE;
		DROP TABLE IF EXISTS interfaces CASCADE;
		DROP TABLE IF EXISTS location CASCADE;
		DROP TABLE IF EXISTS migrations CASCADE;
		DROP TABLE IF EXISTS mint_transaction CASCADE;
		DROP TABLE IF EXISTS name_contract CASCADE;
		DROP TABLE IF EXISTS node CASCADE;
		DROP TABLE IF EXISTS node_contract CASCADE;
		DROP TABLE IF EXISTS node_resources_free CASCADE;
		DROP TABLE IF EXISTS node_resources_total CASCADE;
		DROP TABLE IF EXISTS node_resources_used CASCADE;
		DROP TABLE IF EXISTS nru_consumption CASCADE;
		DROP TABLE IF EXISTS pricing_policy CASCADE;
		DROP TABLE IF EXISTS public_config CASCADE;
		DROP TABLE IF EXISTS public_ip CASCADE;
		DROP TABLE IF EXISTS refund_transaction CASCADE;
		DROP TABLE IF EXISTS rent_contract CASCADE;
		DROP TABLE IF EXISTS transfer CASCADE;
		DROP TABLE IF EXISTS twin CASCADE;
		DROP TABLE IF EXISTS typeorm_metadata CASCADE;
		DROP TABLE IF EXISTS uptime_event CASCADE;
		DROP SCHEMA IF EXISTS substrate_threefold_status CASCADE;
		DROP TABLE IF EXISTS node_gpu CASCADE;
		
	`)
	return err
}

func initSchema(db *sql.DB) error {
	schema, err := os.ReadFile("./schema.sql")
	if err != nil {
		return fmt.Errorf("failed to init schema: %w", err)
	}
	_, err = db.Exec(string(schema))
	if err != nil {
		return fmt.Errorf("failed to init schema: %w", err)
	}
	return nil
}

func generateData(db *sql.DB, gormDB *gorm.DB, seed int) error {
	generator := crafter.NewCrafter(db, gormDB,
		seed,
		NodeCount,
		FarmCount,
		TwinCount,
		PublicIPCount,
		NodeContractCount,
		NameContractCount,
		RentContractCount,
		1,
		1,
		1,
		1,
		1,
		1)

	if err := generator.GenerateTwins(); err != nil {
		return fmt.Errorf("failed to generate twins: %w", err)
	}

	if err := generator.GenerateFarms(); err != nil {
		return fmt.Errorf("failed to generate farms: %w", err)
	}

	if err := generator.GenerateNodes(); err != nil {
		return fmt.Errorf("failed to generate nodes: %w", err)
	}

	if err := generator.GenerateContracts(); err != nil {
		return fmt.Errorf("failed to generate contracts: %w", err)
	}

	if err := generator.GeneratePublicIPs(); err != nil {
		return fmt.Errorf("failed to generate public ips: %w", err)
	}

	if err := generator.GenerateNodeGPUs(); err != nil {
		return fmt.Errorf("failed to generate node gpus: %w", err)
	}

	if err := generator.GenerateCountries(); err != nil {
		return fmt.Errorf("failed to generate countries: %w", err)
	}

	if err := generator.GenerateSpeedReports(); err != nil {
		return fmt.Errorf("failed to generate speed reports: %w", err)
	}

	if err := generator.GenerateDmi(); err != nil {
		return fmt.Errorf("failed to generate dmi reports: %w", err)
	}

	if err := generator.GenerateHealthReports(); err != nil {
		return fmt.Errorf("failed to generate dmi reports: %w", err)
	}

	return nil
}
