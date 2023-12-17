package db

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/nodestatus"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"gorm.io/gorm"
)

type farmQuery struct {
	d         *gorm.DB
	q         *gorm.DB
	nodeQuery *gorm.DB

	hasPublicIPCount  bool // to indicate if public ip count subquery was joined
	hasNodes          bool // to indicate if nodes subquery was joined
	hasNodesResources bool // to indicate if nodes_resources_view was joined
	hasNodeGPU        bool
	hasRentContract   bool
	hasCountry        bool
}

func newFarmsQuery(d *gorm.DB) farmQuery {
	return farmQuery{
		d: d,
		q: d.Table("farm").
			Select(`
				farm.farm_id,
				farm.name,
				farm.twin_id,
				farm.pricing_policy_id,
				farm.certification,
				farm.stellar_address,
				farm.dedicated_farm as dedicated,
				COALESCE(public_ips.public_ips, '[]') as public_ips
			`).
			Joins(`
				LEFT JOIN (
					SELECT
						farm_id,
						jsonb_agg(jsonb_build_object('id', id, 'ip', ip, 'contract_id', contract_id, 'gateway', gateway)) AS public_ips
					FROM public_ip
					GROUP BY farm_id
				) public_ips ON public_ips.farm_id = farm.id
			`).
			Group(`
				farm.id,
				farm.farm_id,
				farm.name,
				farm.twin_id,
				farm.pricing_policy_id,
				farm.certification,
				farm.stellar_address,
				farm.dedicated_farm,
				COALESCE(public_ips.public_ips, '[]')`),
	}
}

func (f *farmQuery) joinPublicIPCount() {
	if !f.hasPublicIPCount {
		f.hasPublicIPCount = true

		f.q = f.q.Joins(`
			LEFT JOIN(
				SELECT
					p1.farm_id,
					COUNT(p1.id) total_ips,
					COUNT(CASE WHEN p2.contract_id = 0 THEN 1 END) free_ips
				FROM public_ip p1
				LEFT JOIN public_ip p2 ON p1.id = p2.id
				GROUP BY p1.farm_id
			) public_ip_count ON public_ip_count.farm_id = farm.id
		`)
	}
}

func (f *farmQuery) createNodesSubquery() {
	if f.nodeQuery == nil {
		f.nodeQuery = f.d.Table("node").Select("node.farm_id")
	}
}

func (f *farmQuery) joinNodesResourcesView() {
	f.createNodesSubquery()

	if !f.hasNodesResources {
		f.hasNodesResources = true
		f.nodeQuery = f.nodeQuery.Joins("LEFT JOIN nodes_resources_view ON nodes_resources_view.node_id = node.node_id")
	}
}

func (f *farmQuery) joinGPUTable() {
	f.createNodesSubquery()

	if !f.hasNodeGPU {
		f.hasNodeGPU = true

		f.nodeQuery = f.nodeQuery.Joins("LEFT JOIN node_gpu ON node.twin_id = node_gpu.node_twin_id").
			Group("node.farm_id")
		if f.hasRentContract {
			f.nodeQuery = f.nodeQuery.Group(`node.farm_id, rent_contract.twin_id`)
		}
	}
}

func (f *farmQuery) joinRentContractTable() {
	f.createNodesSubquery()

	if !f.hasRentContract {
		f.hasRentContract = true
		f.nodeQuery = f.nodeQuery.Joins("LEFT JOIN rent_contract ON rent_contract.state IN ('Created', 'GracePeriod') AND rent_contract.node_id = node.node_id")
	}
}

func (f *farmQuery) joinCountryTable() {
	f.createNodesSubquery()

	if !f.hasCountry {
		f.hasCountry = true
		f.nodeQuery = f.nodeQuery.Joins("LEFT JOIN country ON country.name = node.country")
	}
}

func (f *farmQuery) addNodeResourcesFilter(str string, args ...interface{}) {
	// ensure node subquery is joined
	f.joinNodesResourcesView()

	// add where clause
	f.nodeQuery = f.nodeQuery.Where(str, args...)
}

func (f *farmQuery) addGPUFilter(str string) {
	f.joinGPUTable()

	f.nodeQuery = f.nodeQuery.Where(str)
}

func (f *farmQuery) addNodeFilter(str string, args ...interface{}) {
	f.createNodesSubquery()

	f.nodeQuery = f.nodeQuery.Where(str, args...)
}

func (f *farmQuery) addNodeAvailabilityFilter(str string, args ...interface{}) {
	f.joinRentContractTable()

	f.nodeQuery = f.nodeQuery.Select(
		"node.farm_id",
		"COALESCE(rent_contract.twin_id, 0) as renter",
	)

	if f.hasNodeGPU {
		f.nodeQuery = f.nodeQuery.Group(`node.farm_id, rent_contract.twin_id`)
	}

	f.q = f.q.Where(str, args...)
}

func (f *farmQuery) addCountryFilter(str string, args ...interface{}) {
	f.joinCountryTable()

	f.nodeQuery = f.nodeQuery.Where(str, args...)
}

func (f *farmQuery) addPublicIPCountFilter(str string, args ...interface{}) {
	// ensure public ip table is joined
	f.joinPublicIPCount()

	// add where clause
	f.q = f.q.Where(str, args...)
}

func (f *farmQuery) addFarmFilter(str string, args ...interface{}) {
	f.q = f.q.Where(str, args...)
}

func (f *farmQuery) getQuery() *gorm.DB {
	if f.nodeQuery != nil {
		f.q = f.q.Joins("RIGHT JOIN (?) AS node ON farm.farm_id = node.farm_id", f.nodeQuery)
	}

	if f.hasRentContract {
		f.q = f.q.Select(`
		farm.farm_id,
		farm.name,
		farm.twin_id,
		farm.pricing_policy_id,
		farm.certification,
		farm.stellar_address,
		farm.dedicated_farm as dedicated,
		COALESCE(public_ips.public_ips, '[]') as public_ips,
		bool_or(node.renter != 0)
	`)
	}

	return f.q
}

// GetFarms return farms filtered and paginated
func (d *PostgresDatabase) GetFarms(ctx context.Context, filter types.FarmFilter, limit types.Limit) ([]Farm, uint, error) {
	fq := newFarmsQuery(d.gormDB)

	if filter.NodeFreeMRU != nil {
		fq.addNodeResourcesFilter("nodes_resources_view.free_mru >= ?", *filter.NodeFreeMRU)
	}

	if filter.NodeFreeHRU != nil {
		fq.addNodeResourcesFilter("nodes_resources_view.free_hru >= ?", *filter.NodeFreeHRU)
	}

	if filter.NodeFreeSRU != nil {
		fq.addNodeResourcesFilter("nodes_resources_view.free_sru >= ?", *filter.NodeFreeSRU)
	}

	if filter.NodeHasGPU != nil {
		if *filter.NodeHasGPU == true {
			fq.addGPUFilter("node_gpu.node_twin_id IS NOT NULL")
		} else {
			fq.addGPUFilter("node_gpu.node_twin_id IS NULL")
		}
	}

	if filter.NodeRentedBy != nil {
		fq.addNodeAvailabilityFilter("renter = ?", *filter.NodeRentedBy)
	}

	if filter.Country != nil {
		fq.addCountryFilter("node.country ILIKE ?", *filter.Country)
	}

	if filter.Region != nil {
		fq.addCountryFilter("country.subregion ILIKE ?", *filter.Region)
	}

	if filter.NodeStatus != nil {
		condition := nodestatus.DecideNodeStatusCondition(*filter.NodeStatus)
		fq.addNodeFilter(condition)
	}

	if filter.NodeCertified != nil {
		fq.addNodeFilter("(node.certification = 'Certified') = ?", *filter.NodeCertified)
	}

	if filter.NodeAvailableFor != nil {
		fq.addNodeAvailabilityFilter("renter = ? OR (renter = 0 AND farm.dedicated_farm = false)", *filter.NodeAvailableFor)
	}

	if filter.FreeIPs != nil {
		fq.addPublicIPCountFilter("COALESCE(public_ip_count.free_ips, 0) >= ?", *filter.FreeIPs)
	}
	if filter.TotalIPs != nil {
		fq.addPublicIPCountFilter("COALESCE(public_ip_count.total_ips, 0) >= ?", *filter.TotalIPs)
	}
	if filter.StellarAddress != nil {
		fq.addFarmFilter("farm.stellar_address = ?", *filter.StellarAddress)
	}
	if filter.PricingPolicyID != nil {
		fq.addFarmFilter("farm.pricing_policy_id = ?", *filter.PricingPolicyID)
	}
	if filter.FarmID != nil {
		fq.addFarmFilter("farm.farm_id = ?", *filter.FarmID)
	}
	if filter.TwinID != nil {
		fq.addFarmFilter("farm.twin_id = ?", *filter.TwinID)
	}
	if filter.Name != nil {
		fq.addFarmFilter("farm.name ILIKE ?", *filter.Name)
	}

	if filter.NameContains != nil {
		escaped := strings.Replace(*filter.NameContains, "%", "\\%", -1)
		escaped = strings.Replace(escaped, "_", "\\_", -1)
		fq.addFarmFilter("farm.name ILIKE ?", fmt.Sprintf("%%%s%%", escaped))
	}

	if filter.CertificationType != nil {
		fq.addFarmFilter("farm.certification = ?", *filter.CertificationType)
	}

	if filter.Dedicated != nil {
		fq.addFarmFilter("farm.dedicated_farm = ?", *filter.Dedicated)
	}

	q := fq.getQuery()

	var count int64
	if limit.Randomize || limit.RetCount {
		if res := q.Count(&count); res.Error != nil {
			return nil, 0, errors.Wrap(res.Error, "couldn't get farm count")
		}
	}

	if limit.Randomize {
		q = q.Order("random()")
	} else {
		if filter.NodeAvailableFor != nil {
			q = q.Order("bool_or(node.renter != 0) DESC")
		}

		if limit.SortBy != "" {
			order := types.SortOrderAsc
			if strings.EqualFold(string(limit.SortOrder), string(types.SortOrderDesc)) {
				order = types.SortOrderDesc
			}
			q = q.Order(fmt.Sprintf("%s %s", limit.SortBy, order))
		} else {
			q = q.Order("farm.farm_id")
		}
	}
	// Pagination
	q = q.Limit(int(limit.Size)).Offset(int(limit.Page-1) * int(limit.Size))

	var farms []Farm
	err := q.Scan(&farms).Error
	if d.shouldRetry(err) {
		err = q.Scan(&farms).Error
	}
	if err != nil {
		return farms, 0, errors.Wrap(err, "failed to scan returned farm from database")
	}
	return farms, uint(count), nil
}
