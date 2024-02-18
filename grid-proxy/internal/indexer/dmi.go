package indexer

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
	zosDmiTypes "github.com/threefoldtech/zos/pkg/capacity/dmi"
)

const (
	DmiCallCmd = "zos.system.dmi"
)

type DmiIndexer struct {
	database        db.Database
	rmbClient       *peer.RpcClient
	interval        time.Duration
	workers         uint
	batchSize       uint
	nodeTwinIdsChan chan uint32
	resultChan      chan types.Dmi
	batchChan       chan []types.Dmi
}

func NewDmiIndexer(
	rmbClient *peer.RpcClient,
	database db.Database,
	batchSize uint,
	interval uint,
	workers uint,
) *DmiIndexer {
	return &DmiIndexer{
		database:        database,
		rmbClient:       rmbClient,
		interval:        time.Duration(interval) * time.Minute,
		workers:         workers,
		batchSize:       batchSize,
		nodeTwinIdsChan: make(chan uint32),
		resultChan:      make(chan types.Dmi),
		batchChan:       make(chan []types.Dmi),
	}
}

func (w *DmiIndexer) Start(ctx context.Context) {
	go w.startNodeTableWatcher(ctx)
	go w.StartNodeFinder(ctx)

	for i := uint(0); i < w.workers; i++ {
		go w.StartNodeCaller(ctx)
	}

	for i := uint(0); i < w.workers; i++ {
		go w.StartResultBatcher(ctx)
	}

	go w.StartBatchUpserter(ctx)
}

func (w *DmiIndexer) StartNodeFinder(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	queryUpNodes(ctx, w.database, w.nodeTwinIdsChan)
	for {
		select {
		case <-ticker.C:
			queryUpNodes(ctx, w.database, w.nodeTwinIdsChan)
		case <-ctx.Done():
			return
		}
	}
}

func (n *DmiIndexer) startNodeTableWatcher(ctx context.Context) {
	ticker := time.NewTicker(newNodesCheckInterval)
	latestCheckedID, err := n.database.GetLastNodeTwinID(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get last node twin id")
	}

	for {
		select {
		case <-ticker.C:
			newIDs, err := n.database.GetNodeTwinIDsAfter(ctx, latestCheckedID)
			if err != nil {
				log.Error().Err(err).Msgf("failed to get node twin ids after %d", latestCheckedID)
				continue
			}
			if len(newIDs) == 0 {
				continue
			}

			latestCheckedID = newIDs[0]
			for _, id := range newIDs {
				n.nodeTwinIdsChan <- id
			}
		case <-ctx.Done():
			return
		}
	}
}

func (w *DmiIndexer) StartNodeCaller(ctx context.Context) {
	for {
		select {
		case twinId := <-w.nodeTwinIdsChan:
			var dmi zosDmiTypes.DMI
			err := callNode(ctx, w.rmbClient, DmiCallCmd, nil, twinId, &dmi)
			if err != nil {
				continue
			}

			w.resultChan <- parseDmiResponse(dmi, twinId)
		case <-ctx.Done():
			return
		}
	}
}

func (w *DmiIndexer) StartResultBatcher(ctx context.Context) {
	buffer := make([]types.Dmi, 0, w.batchSize)

	ticker := time.NewTicker(flushingBufferInterval)
	for {
		select {
		case dmiData := <-w.resultChan:
			buffer = append(buffer, dmiData)
			if len(buffer) >= int(w.batchSize) {
				w.batchChan <- buffer
				buffer = nil
			}
		case <-ticker.C:
			if len(buffer) != 0 {
				w.batchChan <- buffer
				buffer = nil
			}
		case <-ctx.Done():
			return
		}
	}
}

func (w *DmiIndexer) StartBatchUpserter(ctx context.Context) {
	for {

		select {
		case batch := <-w.batchChan:
			err := w.database.UpsertNodeDmi(ctx, batch)
			if err != nil {
				log.Error().Err(err).Msg("failed to upsert node dmi")
			}
		case <-ctx.Done():
			return
		}
	}
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
