package indexer

import (
	"context"
	"time"

	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
	zosDmiTypes "github.com/threefoldtech/zos/pkg/capacity/dmi"
)

const (
	DmiCallCmd = "zos.system.dmi"
)

type DMIWork struct {
	findersInterval map[string]time.Duration
}

func NewDMIWork(interval uint) *DMIWork {
	return &DMIWork{
		findersInterval: map[string]time.Duration{
			"up":  time.Duration(interval) * time.Minute,
			"new": newNodesCheckInterval,
		},
	}
}

func (w *DMIWork) Finders() map[string]time.Duration {
	return w.findersInterval
}

func (w *DMIWork) Get(ctx context.Context, rmb *peer.RpcClient, twinId uint32) ([]types.Dmi, error) {
	var dmi zosDmiTypes.DMI
	err := callNode(ctx, rmb, DmiCallCmd, nil, twinId, &dmi)
	if err != nil {
		return []types.Dmi{}, err
	}

	res := parseDmiResponse(dmi, twinId)
	return []types.Dmi{res}, nil

}

func (w *DMIWork) Upsert(ctx context.Context, db db.Database, batch []types.Dmi) error {
	return db.UpsertNodeDmi(ctx, batch)
}

func parseDmiResponse(dmiResponse zosDmiTypes.DMI, twinId uint32) types.Dmi {
	var info types.Dmi
	for _, sec := range dmiResponse.Sections {
		if sec.TypeStr == "Processor" {
			for _, subSec := range sec.SubSections {
				if subSec.Title == "Processor Information" {
					info.Processor = append(info.Processor, types.Processor{
						Version:     subSec.Properties["Version"].Val,
						ThreadCount: subSec.Properties["Thread Count"].Val,
					})
				}
			}
		}
		if sec.TypeStr == "MemoryDevice" {
			for _, subSec := range sec.SubSections {
				if subSec.Title == "Memory Device" {
					if subSec.Properties["Type"].Val == "Unknown" {
						continue
					}
					info.Memory = append(info.Memory, types.Memory{
						Type:         subSec.Properties["Type"].Val,
						Manufacturer: subSec.Properties["Manufacturer"].Val,
					})
				}
			}
		}
		if sec.TypeStr == "Baseboard" {
			for _, subSec := range sec.SubSections {
				if subSec.Title == "Base Board Information" {
					info.Baseboard.Manufacturer = subSec.Properties["Manufacturer"].Val
					info.Baseboard.ProductName = subSec.Properties["Product Name"].Val
				}
			}
		}
		if sec.TypeStr == "BIOS" {
			for _, subSec := range sec.SubSections {
				if subSec.Title == "BIOS Information" {
					info.BIOS.Vendor = subSec.Properties["Vendor"].Val
					info.BIOS.Version = subSec.Properties["Version"].Val
				}
			}
		}
	}

	info.NodeTwinId = twinId
	return info
}
