package indexer

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/internal/explorer/db"
	"github.com/threefoldtech/tfgrid-sdk-go/grid-proxy/pkg/types"
	"github.com/threefoldtech/tfgrid-sdk-go/rmb-sdk-go/peer"
)

const (
	DmiCallCmd       = "zos.system.dmi"
	flushingInterval = 60 * time.Second
)

type DmiWatcher struct {
	database        db.Database
	rmbClient       *peer.RpcClient
	nodeTwinIdsChan chan uint32
	resultChan      chan types.DmiInfo
	interval        time.Duration
	workers         uint
	batchSize       uint
}

func NewDmiWatcher(
	ctx context.Context,
	database db.Database,
	rmbClient *peer.RpcClient,
	interval uint,
	workers uint,
	batchSize uint,
) *DmiWatcher {
	return &DmiWatcher{
		database:        database,
		rmbClient:       rmbClient,
		nodeTwinIdsChan: make(chan uint32),
		resultChan:      make(chan types.DmiInfo),
		interval:        time.Duration(interval) * time.Minute,
		workers:         workers,
		batchSize:       batchSize,
	}
}

func (w *DmiWatcher) Start(ctx context.Context) {
	go w.startNodeQuerier(ctx)

	for i := uint(0); i < w.workers; i++ {
		go w.startNodeCaller(ctx)
	}

	go w.startUpserter(ctx, w.database)
}

// TODO: not only on interval but also on any node goes from down>up or newly added nodes
func (w *DmiWatcher) startNodeQuerier(ctx context.Context) {
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

func (w *DmiWatcher) startNodeCaller(ctx context.Context) {
	for {
		select {
		case twinId := <-w.nodeTwinIdsChan:
			response, err := w.callNode(ctx, twinId)
			if err != nil {
				continue
			}
			parsedDmi := parseDmiResponse(response)
			parsedDmi.NodeTwinId = twinId
			w.resultChan <- parsedDmi
		case <-ctx.Done():
			return
		}
	}
}

// TODO: make it generic and then assert the result in each watcher
func (w *DmiWatcher) callNode(ctx context.Context, twinId uint32) (DMI, error) {
	var result DMI
	subCtx, cancel := context.WithTimeout(ctx, indexerCallTimeout)
	defer cancel()

	err := w.rmbClient.Call(subCtx, twinId, DmiCallCmd, nil, &result)
	if err != nil {
		log.Error().Err(err).Uint32("twinId", twinId).Msg("failed to call node")
	}

	return result, err
}

func (w *DmiWatcher) startUpserter(ctx context.Context, database db.Database) {
	buffer := make([]types.DmiInfo, 0, w.batchSize)

	ticker := time.NewTicker(flushingInterval)
	for {
		select {
		case dmiData := <-w.resultChan:
			buffer = append(buffer, dmiData)
			if len(buffer) >= int(w.batchSize) {
				err := w.database.UpsertNodeDmi(ctx, buffer)
				if err != nil {
					log.Error().Err(err).Msgf("failed")
				}
				buffer = nil
			}
		case <-ticker.C:
			if len(buffer) != 0 {
				err := w.database.UpsertNodeDmi(ctx, buffer)
				if err != nil {
					log.Error().Err(err).Msgf("failed")
				}
				buffer = nil
			}
		case <-ctx.Done():
			return
		}
	}
}

func parseDmiResponse(dmiResponse DMI) types.DmiInfo {
	var info types.DmiInfo
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

	return info
}
