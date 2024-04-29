package db

import (
	"context"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"gorm.io/gorm/clause"
)

func (p *PostgresDatabase) DeleteOldGpus(ctx context.Context, nodeTwinIds []uint32, expiration int64) error {
	return p.gormDB.WithContext(ctx).Table("node_gpu").Where("node_twin_id IN (?) AND updated_at < ?", nodeTwinIds, expiration).Delete(types.NodeGPU{}).Error
}

func (p *PostgresDatabase) GetLastNodeTwinID(ctx context.Context) (uint32, error) {
	var node Node
	err := p.gormDB.WithContext(ctx).Table("node").Order("twin_id DESC").Limit(1).Scan(&node).Error
	return uint32(node.TwinID), err
}

func (p *PostgresDatabase) GetNodeTwinIDsAfter(ctx context.Context, twinID uint32) ([]uint32, error) {
	nodeTwinIDs := make([]uint32, 0)
	err := p.gormDB.WithContext(ctx).Table("node").Select("twin_id").Where("twin_id > ?", twinID).Order("twin_id DESC").Scan(&nodeTwinIDs).Error
	return nodeTwinIDs, err
}

func (p *PostgresDatabase) GetHealthyNodeTwinIds(ctx context.Context) ([]uint32, error) {
	nodeTwinIDs := make([]uint32, 0)
	err := p.gormDB.WithContext(ctx).Table("health_report").Select("node_twin_id").Where("healthy = true").Scan(&nodeTwinIDs).Error
	return nodeTwinIDs, err
}

func (p *PostgresDatabase) UpsertNodesGPU(ctx context.Context, gpus []types.NodeGPU) error {
	conflictClause := clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}, {Name: "node_twin_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"vendor", "device", "contract", "updated_at"}),
	}
	return p.gormDB.WithContext(ctx).Table("node_gpu").Clauses(conflictClause).Create(&gpus).Error
}

func (p *PostgresDatabase) UpsertNodeHealth(ctx context.Context, healthReports []types.HealthReport) error {
	conflictClause := clause.OnConflict{
		Columns:   []clause.Column{{Name: "node_twin_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"healthy"}),
	}
	return p.gormDB.WithContext(ctx).Table("health_report").Clauses(conflictClause).Create(&healthReports).Error
}

func (p *PostgresDatabase) UpsertNodeDmi(ctx context.Context, dmis []types.Dmi) error {
	conflictClause := clause.OnConflict{
		Columns:   []clause.Column{{Name: "node_twin_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"bios", "baseboard", "processor", "memory"}),
	}
	return p.gormDB.WithContext(ctx).Table("dmi").Clauses(conflictClause).Create(&dmis).Error
}

func (p *PostgresDatabase) UpsertNetworkSpeed(ctx context.Context, speeds []types.Speed) error {
	conflictClause := clause.OnConflict{
		Columns:   []clause.Column{{Name: "node_twin_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"download", "upload"}),
	}
	return p.gormDB.WithContext(ctx).Table("speed").Clauses(conflictClause).Create(&speeds).Error
}

func (p *PostgresDatabase) UpsertNodeIpv6Report(ctx context.Context, ips []types.HasIpv6) error {
	onConflictClause := clause.OnConflict{
		Columns:   []clause.Column{{Name: "node_twin_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"has_ipv6"}),
	}
	return p.gormDB.WithContext(ctx).Table("node_ipv6").Clauses(onConflictClause).Create(&ips).Error
}
