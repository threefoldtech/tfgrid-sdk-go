package explorer

import (
	"encoding/json"
	"math"

	"github.com/pkg/errors"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/nodestatus"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"

	"github.com/threefoldtech/zosbase/pkg/gridtypes"
)

func calculateAvailableSlices(freeMru, freeSru, freeHru, freeCru int64, sliceMru, sliceSru, sliceHru, sliceCru int64) uint64 {
	if sliceMru == 0 {
		return 0
	}

	// Calculate how many complete slices each resource can provide
	slicesByMru := uint64(freeMru / sliceMru)
	var slicesBySru, slicesByHru, slicesByCru uint64

	if sliceSru > 0 && freeSru > 0 {
		slicesBySru = uint64(freeSru / sliceSru)
	}
	if sliceHru > 0 && freeHru > 0 {
		slicesByHru = uint64(freeHru / sliceHru)
	}
	if sliceCru > 0 && freeCru > 0 {
		slicesByCru = uint64(freeCru / sliceCru)
	}

	// Return the minimum (bottleneck resource)
	minSlices := slicesByMru
	if slicesBySru < minSlices {
		minSlices = slicesBySru
	}
	if slicesByHru < minSlices {
		minSlices = slicesByHru
	}
	if slicesByCru < minSlices {
		minSlices = slicesByCru
	}

	return minSlices
}

func nodeFromDBNode(info db.Node) types.Node {
	// Calculate free resources
	freeMru := info.TotalMru - info.UsedMru
	freeSru := info.TotalSru - info.UsedSru
	freeHru := info.TotalHru - info.UsedHru
	freeCru := info.TotalCru - info.UsedCru

	node := types.Node{
		ID:              info.ID,
		NodeID:          int(info.NodeID),
		FarmID:          int(info.FarmID),
		FarmName:        info.FarmName,
		TwinID:          int(info.TwinID),
		Country:         info.Country,
		GridVersion:     int(info.GridVersion),
		City:            info.City,
		Uptime:          info.Uptime,
		Created:         info.Created,
		FarmingPolicyID: int(info.FarmingPolicyID),
		UpdatedAt:       info.UpdatedAt,
		TotalResources: types.Capacity{
			CRU: uint64(info.TotalCru),
			SRU: gridtypes.Unit(info.TotalSru),
			HRU: gridtypes.Unit(info.TotalHru),
			MRU: gridtypes.Unit(info.TotalMru),
		},
		UsedResources: types.Capacity{
			CRU: uint64(info.UsedCru),
			SRU: gridtypes.Unit(info.UsedSru),
			HRU: gridtypes.Unit(info.UsedHru),
			MRU: gridtypes.Unit(info.UsedMru),
		},
		Location: types.Location{
			Country:   info.Country,
			City:      info.City,
			Longitude: info.Longitude,
			Latitude:  info.Latitude,
			Region:    info.Continent,
		},
		PublicConfig: types.PublicConfig{
			Domain: info.Domain,
			Gw4:    info.Gw4,
			Gw6:    info.Gw6,
			Ipv4:   info.Ipv4,
			Ipv6:   info.Ipv6,
		},
		InDedicatedFarm:   info.FarmDedicated,
		CertificationType: info.Certification,
		RentContractID:    uint(info.RentContractID),
		RentedByTwinID:    uint(info.Renter),
		Rented:            info.Rented,
		Rentable:          info.Rentable,
		SerialNumber:      info.SerialNumber,
		Power:             types.NodePower(info.Power),
		NumGPU:            info.NumGPU,
		ExtraFee:          info.ExtraFee,
		Healthy:           info.Healthy,
		Dmi: types.Dmi{
			Processor: info.Processor,
			Memory:    info.Memory,
			BIOS:      info.Bios,
			Baseboard: info.Baseboard,
		},
		Speed: types.Speed{
			Upload:          info.UploadSpeed,
			Download:        info.DownloadSpeed,
			UDPDownloadIPv4: info.UDPDownloadIPv4,
			UDPUploadIPv4:   info.UDPUploadIPv4,
			TCPDownloadIPv6: info.TCPDownloadIPv6,
			TCPUploadIPv6:   info.TCPUploadIPv6,
			UDPDownloadIPv6: info.UDPDownloadIPv6,
			UDPUploadIPv6:   info.UDPUploadIPv6,
		},
		CpuBenchmark: types.CpuBenchmark{
			SingleThreaded: info.SingleThreaded,
			MultiThreaded:  info.MultiThreaded,
			Threads:        info.Threads,
			Workloads:      info.Workloads,
		},
		GPUs:        info.Gpus,
		PriceUsd:    math.Round(info.PriceUsd*1000) / 1000,
		FarmFreeIps: info.FarmFreeIps,
		Features:    info.Features,
		Slice: types.Capacity{
			CRU: uint64(info.SliceCru),
			SRU: gridtypes.Unit(info.SliceSru),
			HRU: gridtypes.Unit(info.SliceHru),
			MRU: gridtypes.Unit(info.SliceMru),
		},
		AvailableSlices: calculateAvailableSlices(freeMru, freeSru, freeHru, freeCru, info.SliceMru, info.SliceSru, info.SliceHru, info.SliceCru),
	}
	node.Status = nodestatus.DecideNodeStatus(node.Power, node.UpdatedAt)
	node.Dedicated = info.FarmDedicated || info.NodeContractsCount == 0 || info.Renter != 0 || info.ExtraFee > 0
	// have an empty list instead of null in the json response
	if node.Features == nil {
		node.Features = []string{}
	}
	return node
}

func farmFromDBFarm(info db.Farm) (types.Farm, error) {
	farm := types.Farm{
		Name:              info.Name,
		FarmID:            info.FarmID,
		TwinID:            info.TwinID,
		PricingPolicyID:   info.PricingPolicyID,
		CertificationType: info.Certification,
		StellarAddress:    info.StellarAddress,
		Dedicated:         info.Dedicated,
	}
	if err := json.Unmarshal([]byte(info.PublicIps), &farm.PublicIps); err != nil {
		return farm, errors.Wrap(err, "couldn't unmarshal public ips returned from db")
	}
	return farm, nil
}

func nodeWithNestedCapacityFromDBNode(info db.Node) types.NodeWithNestedCapacity {
	// Calculate free resources
	freeMru := info.TotalMru - info.UsedMru
	freeSru := info.TotalSru - info.UsedSru
	freeHru := info.TotalHru - info.UsedHru
	freeCru := info.TotalCru - info.UsedCru

	node := types.NodeWithNestedCapacity{
		ID:              info.ID,
		NodeID:          int(info.NodeID),
		FarmID:          int(info.FarmID),
		FarmName:        info.FarmName,
		TwinID:          int(info.TwinID),
		Country:         info.Country,
		GridVersion:     int(info.GridVersion),
		City:            info.City,
		Uptime:          info.Uptime,
		Created:         info.Created,
		FarmingPolicyID: int(info.FarmingPolicyID),
		UpdatedAt:       info.UpdatedAt,
		Capacity: types.CapacityResult{

			Total: types.Capacity{
				CRU: uint64(info.TotalCru),
				SRU: gridtypes.Unit(info.TotalSru),
				HRU: gridtypes.Unit(info.TotalHru),
				MRU: gridtypes.Unit(info.TotalMru),
			},
			Used: types.Capacity{
				CRU: uint64(info.UsedCru),
				SRU: gridtypes.Unit(info.UsedSru),
				HRU: gridtypes.Unit(info.UsedHru),
				MRU: gridtypes.Unit(info.UsedMru),
			},
		},
		Location: types.Location{
			Country:   info.Country,
			City:      info.City,
			Longitude: info.Longitude,
			Latitude:  info.Latitude,
			Region:    info.Continent,
		},
		PublicConfig: types.PublicConfig{
			Domain: info.Domain,
			Gw4:    info.Gw4,
			Gw6:    info.Gw6,
			Ipv4:   info.Ipv4,
			Ipv6:   info.Ipv6,
		},
		InDedicatedFarm:   info.FarmDedicated,
		CertificationType: info.Certification,
		RentContractID:    uint(info.RentContractID),
		RentedByTwinID:    uint(info.Renter),
		Rented:            info.Rented,
		Rentable:          info.Rentable,
		SerialNumber:      info.SerialNumber,
		Power:             types.NodePower(info.Power),
		NumGPU:            info.NumGPU,
		ExtraFee:          info.ExtraFee,
		Healthy:           info.Healthy,
		Dmi: types.Dmi{
			Processor: info.Processor,
			Memory:    info.Memory,
			BIOS:      info.Bios,
			Baseboard: info.Baseboard,
		},
		Speed: types.Speed{
			Upload:          info.UploadSpeed,
			Download:        info.DownloadSpeed,
			UDPDownloadIPv4: info.UDPDownloadIPv4,
			UDPUploadIPv4:   info.UDPUploadIPv4,
			TCPDownloadIPv6: info.TCPDownloadIPv6,
			TCPUploadIPv6:   info.TCPUploadIPv6,
			UDPDownloadIPv6: info.UDPDownloadIPv6,
			UDPUploadIPv6:   info.UDPUploadIPv6,
		},
		CpuBenchmark: types.CpuBenchmark{
			SingleThreaded: info.SingleThreaded,
			MultiThreaded:  info.MultiThreaded,
			Threads:        info.Threads,
			Workloads:      info.Workloads,
		},
		GPUs:        info.Gpus,
		PriceUsd:    math.Round(info.PriceUsd*1000) / 1000,
		FarmFreeIps: info.FarmFreeIps,
		Features:    info.Features,
		Slice: types.Capacity{
			CRU: uint64(info.SliceCru),
			SRU: gridtypes.Unit(info.SliceSru),
			HRU: gridtypes.Unit(info.SliceHru),
			MRU: gridtypes.Unit(info.SliceMru),
		},
		AvailableSlices: calculateAvailableSlices(freeMru, freeSru, freeHru, freeCru, info.SliceMru, info.SliceSru, info.SliceHru, info.SliceCru),
	}
	node.Status = nodestatus.DecideNodeStatus(node.Power, node.UpdatedAt)
	node.Dedicated = info.FarmDedicated || info.NodeContractsCount == 0 || info.Renter != 0 || info.ExtraFee > 0
	if node.Features == nil {
		node.Features = []string{}
	}
	return node
}

func contractFromDBContract(info db.DBContract) (types.Contract, error) {
	var details interface{}
	switch info.Type {
	case "node":
		details = types.NodeContractDetails{
			NodeID:            info.NodeID,
			DeploymentData:    info.DeploymentData,
			DeploymentHash:    info.DeploymentHash,
			NumberOfPublicIps: info.NumberOfPublicIps,
			FarmName:          info.FarmName,
			FarmId:            info.FarmId,
		}
	case "name":
		details = types.NameContractDetails{
			Name: info.Name,
		}
	case "rent":
		details = types.RentContractDetails{
			NodeID:   info.NodeID,
			FarmName: info.FarmName,
			FarmId:   info.FarmId,
		}
	}
	contract := types.Contract{
		ContractID: info.ContractID,
		TwinID:     info.TwinID,
		State:      info.State,
		CreatedAt:  info.CreatedAt,
		Type:       info.Type,
		Details:    details,
	}
	return contract, nil
}
