package db

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	// to use for database/sql
	_ "github.com/lib/pq"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var (
	// ErrNodeNotFound node not found
	ErrNodeNotFound = errors.New("node not found")
	// ErrFarmNotFound farm not found
	ErrFarmNotFound = errors.New("farm not found")
	//ErrViewNotFound
	ErrNodeResourcesViewNotFound = errors.New("ERROR: relation \"nodes_resources_view\" does not exist (SQLSTATE 42P01)")
)

const (
	nodeStateFactor = 3
	reportInterval  = time.Hour
	// the number of missed reports to mark the node down
	// if node reports every 5 mins, it's marked down if the last report is more than 15 mins in the past
)

const (
	setupPostgresql = `
	CREATE OR REPLACE VIEW nodes_resources_view AS SELECT
		node.node_id,
		COALESCE(sum(contract_resources.cru), 0) as used_cru,
		COALESCE(sum(contract_resources.mru), 0) + GREATEST(CAST((node_resources_total.mru / 10) AS bigint), 2147483648) as used_mru,
		COALESCE(sum(contract_resources.hru), 0) as used_hru,
		COALESCE(sum(contract_resources.sru), 0) + 107374182400 as used_sru,
		node_resources_total.mru - COALESCE(sum(contract_resources.mru), 0) - GREATEST(CAST((node_resources_total.mru / 10) AS bigint), 2147483648) as free_mru,
		node_resources_total.hru - COALESCE(sum(contract_resources.hru), 0) as free_hru,
		node_resources_total.sru - COALESCE(sum(contract_resources.sru), 0) - 107374182400 as free_sru,
		COALESCE(node_resources_total.cru, 0) as total_cru,
		COALESCE(node_resources_total.mru, 0) as total_mru,
		COALESCE(node_resources_total.hru, 0) as total_hru,
		COALESCE(node_resources_total.sru, 0) as total_sru,
		COALESCE(COUNT(DISTINCT state), 0) as states
	FROM contract_resources
	JOIN node_contract as node_contract
	ON node_contract.resources_used_id = contract_resources.id AND node_contract.state IN ('Created', 'GracePeriod')
	RIGHT JOIN node as node
	ON node.node_id = node_contract.node_id
	JOIN node_resources_total AS node_resources_total
	ON node_resources_total.node_id = node.id
	GROUP BY node.node_id, node_resources_total.mru, node_resources_total.sru, node_resources_total.hru, node_resources_total.cru;

	DROP FUNCTION IF EXISTS node_resources(query_node_id INTEGER);
	CREATE OR REPLACE function node_resources(query_node_id INTEGER)
	returns table (node_id INTEGER, used_cru NUMERIC, used_mru NUMERIC, used_hru NUMERIC, used_sru NUMERIC, free_mru NUMERIC, free_hru NUMERIC, free_sru NUMERIC, total_cru NUMERIC, total_mru NUMERIC, total_hru NUMERIC, total_sru NUMERIC, states BIGINT)
	as
	$body$
	SELECT
		node.node_id,
		COALESCE(sum(contract_resources.cru), 0) as used_cru,
		COALESCE(sum(contract_resources.mru), 0) + GREATEST(CAST((node_resources_total.mru / 10) AS bigint), 2147483648) as used_mru,
		COALESCE(sum(contract_resources.hru), 0) as used_hru,
		COALESCE(sum(contract_resources.sru), 0) + 107374182400 as used_sru,
		node_resources_total.mru - COALESCE(sum(contract_resources.mru), 0) - GREATEST(CAST((node_resources_total.mru / 10) AS bigint), 2147483648) as free_mru,
		node_resources_total.hru - COALESCE(sum(contract_resources.hru), 0) as free_hru,
		node_resources_total.sru - COALESCE(sum(contract_resources.sru), 0) - 107374182400 as free_sru,
		COALESCE(node_resources_total.cru, 0) as total_cru,
		COALESCE(node_resources_total.mru, 0) as total_mru,
		COALESCE(node_resources_total.hru, 0) as total_hru,
		COALESCE(node_resources_total.sru, 0) as total_sru,
		COALESCE(COUNT(DISTINCT state), 0) as states
	FROM contract_resources
	JOIN node_contract as node_contract
	ON node_contract.resources_used_id = contract_resources.id AND node_contract.state IN ('Created', 'GracePeriod')
	RIGHT JOIN node as node
	ON node.node_id = node_contract.node_id
	JOIN node_resources_total AS node_resources_total
	ON node_resources_total.node_id = node.id
	WHERE node.node_id = query_node_id
	GROUP BY node.node_id, node_resources_total.mru, node_resources_total.sru, node_resources_total.hru, node_resources_total.cru;
	$body$
	language sql;

	DROP FUNCTION IF EXISTS convert_to_decimal(v_input text);
	CREATE OR REPLACE FUNCTION convert_to_decimal(v_input text)
	RETURNS DECIMAL AS $$
	DECLARE v_dec_value DECIMAL DEFAULT NULL;
	BEGIN
		BEGIN
			v_dec_value := v_input::DECIMAL;
		EXCEPTION WHEN OTHERS THEN
			RAISE NOTICE 'Invalid decimal value: "%".  Returning NULL.', v_input;
			RETURN NULL;
		END;
	RETURN v_dec_value;
	END;
	$$ LANGUAGE plpgsql;`
)

// PostgresDatabase postgres db client
type PostgresDatabase struct {
	gormDB *gorm.DB
}

// NewPostgresDatabase returns a new postgres db client
func NewPostgresDatabase(host string, port int, user, password, dbname string) (Database, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	gormDB, err := gorm.Open(postgres.Open(psqlInfo), &gorm.Config{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create orm wrapper around db")
	}
	res := PostgresDatabase{gormDB}
	if err := res.initialize(); err != nil {
		return nil, errors.Wrap(err, "failed to setup tables")
	}
	return &res, nil
}

// Close the db connection
func (d *PostgresDatabase) Close() error {
	db, err := d.gormDB.DB()
	if err != nil {
		return err
	}
	return db.Close()
}

func (d *PostgresDatabase) initialize() error {
	res := d.gormDB.Exec(setupPostgresql)
	return res.Error
}

// Scan is a custom decoder for jsonb filed. executed while scanning the node.
func (np *NodePower) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	if data, ok := value.([]byte); ok {
		return json.Unmarshal(data, np)
	}
	return fmt.Errorf("failed to unmarshal NodePower")
}

//lint:ignore U1000 used for debugging
func convertParam(p interface{}) string {
	if v, ok := p.(string); ok {
		return fmt.Sprintf("'%s'", v)
	} else if v, ok := p.(uint64); ok {
		return fmt.Sprintf("%d", v)
	} else if v, ok := p.(int64); ok {
		return fmt.Sprintf("%d", v)
	} else if v, ok := p.(uint32); ok {
		return fmt.Sprintf("%d", v)
	} else if v, ok := p.(int); ok {
		return fmt.Sprintf("%d", v)
	} else if v, ok := p.(gridtypes.Unit); ok {
		return fmt.Sprintf("%d", v)
	}
	log.Error().Msgf("can't recognize type %s", fmt.Sprintf("%v", p))
	return "0"
}

// nolint
//
//lint:ignore U1000 used for debugging
func printQuery(query string, args ...interface{}) {
	for i, e := range args {
		query = strings.ReplaceAll(query, fmt.Sprintf("$%d", i+1), convertParam(e))
	}
	fmt.Printf("node query: %s", query)
}

func (d *PostgresDatabase) shouldRetry(resError error) bool {
	if resError != nil && resError.Error() == ErrNodeResourcesViewNotFound.Error() {
		if err := d.initialize(); err != nil {
			log.Logger.Err(err).Msg("failed to reinitialize database")
		} else {
			return true
		}
	}
	return false
}
