package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/tools/db/modifiers"
)

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

func generateData(db *sql.DB, seed int) error {
	generator := modifiers.NewGenerator(db, seed)

	if err := generator.GenerateTwins(1, modifiers.TwinCount); err != nil {
		return fmt.Errorf("failed to generate twins: %w", err)
	}

	if err := generator.GenerateFarms(1, modifiers.FarmCount, 1); err != nil {
		return fmt.Errorf("failed to generate farms: %w", err)
	}

	if err := generator.GenerateNodes(1, modifiers.NodeCount, 1, modifiers.FarmCount, 1); err != nil {
		return fmt.Errorf("failed to generate nodes: %w", err)
	}

	if err := generator.GenerateContracts(1, 1, modifiers.NodeContractCount, modifiers.NameContractCount, modifiers.RentContractCount, 1, modifiers.NodeCount); err != nil {
		return fmt.Errorf("failed to generate contracts: %w", err)
	}

	if err := generator.GeneratePublicIPs(1, modifiers.PublicIPCount); err != nil {
		return fmt.Errorf("failed to generate public ips: %w", err)
	}

	if err := generator.GenerateNodeGPUs(); err != nil {
		return fmt.Errorf("failed to generate node gpus: %w", err)
	}

	if err := generator.GenerateCountries(); err != nil {
		return fmt.Errorf("failed to generate countries: %w", err)
	}
	return nil
}
