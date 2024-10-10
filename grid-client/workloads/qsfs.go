// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"encoding/hex"
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	zosTypes "github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// QSFS struct
type QSFS struct {
	Name                 string   `json:"name"`
	Description          string   `json:"description"`
	Cache                int      `json:"cache"`
	MinimalShards        uint32   `json:"minimal_shards"`
	ExpectedShards       uint32   `json:"expected_shards"`
	RedundantGroups      uint32   `json:"redundant_groups"`
	RedundantNodes       uint32   `json:"redundant_nodes"`
	MaxZDBDataDirSize    uint32   `json:"max_zdb_data_dir_size"`
	EncryptionAlgorithm  string   `json:"encryption_algorithm"`
	EncryptionKey        string   `json:"encryption_key"`
	CompressionAlgorithm string   `json:"compression_algorithm"`
	Metadata             Metadata `json:"metadata"`
	Groups               Groups   `json:"groups"`

	// OUTPUT
	MetricsEndpoint string `json:"metrics_endpoint"`
}

func (q *QSFS) Validate() error {
	if err := validateName(q.Name); err != nil {
		return errors.Wrap(err, "qsfs name is invalid")
	}

	if q.MinimalShards > q.ExpectedShards {
		return errors.New("minimal shards can't be greater than expected shards")
	}

	return nil
}

// Metadata for QSFS
type Metadata struct {
	Type                string   `json:"type"`
	Prefix              string   `json:"prefix"`
	EncryptionAlgorithm string   `json:"encryption_algorithm"`
	EncryptionKey       string   `json:"encryption_key"`
	Backends            Backends `json:"backends"`
}

// Group is a zos group
type Group struct {
	Backends Backends `json:"backends"`
}

// Backend is a zos backend
type Backend zosTypes.ZdbBackend

// Groups is a list of groups
type Groups []Group

// Backends is a list of backends
type Backends []Backend

func (g *Group) zosGroup() (zdbGroup zosTypes.ZdbGroup) {
	for _, b := range g.Backends {
		zdbGroup.Backends = append(zdbGroup.Backends, b.zosBackend())
	}
	return zdbGroup
}

func (gs Groups) zosGroups() (zdbGroups []zosTypes.ZdbGroup) {
	for _, e := range gs {
		zdbGroups = append(zdbGroups, e.zosGroup())
	}
	return zdbGroups
}

func (b *Backend) zosBackend() zosTypes.ZdbBackend {
	return zosTypes.ZdbBackend(*b)
}

func (bs Backends) zosBackends() (zdbBackends []zosTypes.ZdbBackend) {
	for _, e := range bs {
		zdbBackends = append(zdbBackends, e.zosBackend())
	}
	return zdbBackends
}

// BackendsFromZos gets backends from zos
func BackendsFromZos(bs []zos.ZdbBackend) (backends Backends) {
	for _, e := range bs {
		backends = append(backends, Backend(e))
	}
	return backends
}

// GroupsFromZos gets groups from zos
func GroupsFromZos(gs []zos.ZdbGroup) (groups Groups) {
	for _, e := range gs {
		groups = append(groups, Group{
			Backends: BackendsFromZos(e.Backends),
		})
	}
	return groups
}

// NewQSFSFromWorkload generates a new QSFS from a workload
func NewQSFSFromWorkload(wl *zosTypes.Workload) (QSFS, error) {
	var dataI interface{}

	dataI, err := wl.Workload3().WorkloadData()
	if err != nil {
		dataI, err = wl.Workload4().WorkloadData()
		if err != nil {
			return QSFS{}, errors.Wrap(err, "failed to get workload data")
		}
	}

	data, ok := dataI.(*zos.QuantumSafeFS)
	if !ok {
		return QSFS{}, fmt.Errorf("could not create qsfs workload from data %v", dataI)
	}

	var result zos.QuatumSafeFSResult
	if !reflect.DeepEqual(wl.Result, zosTypes.Result{}) {
		if err := wl.Result.Unmarshal(&result); err != nil {
			return QSFS{}, err
		}
	}

	return QSFS{
		Name:                 wl.Name,
		Description:          wl.Description,
		Cache:                int(data.Cache) / int(zosTypes.Megabyte),
		MinimalShards:        data.Config.MinimalShards,
		ExpectedShards:       data.Config.ExpectedShards,
		RedundantGroups:      data.Config.RedundantGroups,
		RedundantNodes:       data.Config.RedundantNodes,
		MaxZDBDataDirSize:    data.Config.MaxZDBDataDirSize,
		EncryptionAlgorithm:  string(data.Config.Encryption.Algorithm),
		EncryptionKey:        hex.EncodeToString(data.Config.Encryption.Key),
		CompressionAlgorithm: data.Config.Compression.Algorithm,
		Metadata: Metadata{
			Type:                data.Config.Meta.Type,
			Prefix:              data.Config.Meta.Config.Prefix,
			EncryptionAlgorithm: string(data.Config.Meta.Config.Encryption.Algorithm),
			EncryptionKey:       hex.EncodeToString(data.Config.Meta.Config.Encryption.Key),
			Backends:            BackendsFromZos(data.Config.Meta.Config.Backends),
		},
		Groups:          GroupsFromZos(data.Config.Groups),
		MetricsEndpoint: result.MetricsEndpoint,
	}, nil
}

// ZosWorkload generates a zos workload
func (q *QSFS) ZosWorkload() (zosTypes.Workload, error) {
	k, err := hex.DecodeString(q.EncryptionKey)
	if err != nil {
		return zosTypes.Workload{}, err
	}
	mk, err := hex.DecodeString(q.EncryptionKey)
	if err != nil {
		return zosTypes.Workload{}, err
	}

	workload := zosTypes.Workload{
		Version:     0,
		Name:        q.Name,
		Type:        zosTypes.QuantumSafeFSType,
		Description: q.Description,
		Data: zosTypes.MustMarshal(zosTypes.QuantumSafeFS{
			Cache: uint64(q.Cache) * zosTypes.Megabyte,
			Config: zosTypes.QuantumSafeFSConfig{
				MinimalShards:     q.MinimalShards,
				ExpectedShards:    q.ExpectedShards,
				RedundantGroups:   q.RedundantGroups,
				RedundantNodes:    q.RedundantNodes,
				MaxZDBDataDirSize: q.MaxZDBDataDirSize,
				Encryption: zosTypes.Encryption{
					Algorithm: zosTypes.EncryptionAlgorithm(q.EncryptionAlgorithm),
					Key:       zosTypes.EncryptionKey(k),
				},
				Meta: zosTypes.QuantumSafeMeta{
					Type: q.Metadata.Type,
					Config: zosTypes.QuantumSafeConfig{
						Prefix: q.Metadata.Prefix,
						Encryption: zosTypes.Encryption{
							Algorithm: zosTypes.EncryptionAlgorithm(q.EncryptionAlgorithm),
							Key:       zosTypes.EncryptionKey(mk),
						},
						Backends: q.Metadata.Backends.zosBackends(),
					},
				},
				Groups: q.Groups.zosGroups(),
				Compression: zosTypes.QuantumCompression{
					Algorithm: q.CompressionAlgorithm,
				},
			},
		}),
	}

	return workload, nil
}

// UpdateFromWorkload updates a QSFS from a workload
// TODO: no updates, should construct itself from the workload
func (q *QSFS) UpdateFromWorkload(wl *zosTypes.Workload) error {
	if wl == nil {
		q.MetricsEndpoint = ""
		return nil
	}
	var res zos.QuatumSafeFSResult

	if !reflect.DeepEqual(wl.Result, zosTypes.Result{}) {
		if err := wl.Result.Unmarshal(&res); err != nil {
			return err
		}
	}

	q.MetricsEndpoint = res.MetricsEndpoint
	return nil
}
