package zos

import (
	"encoding/hex"
)

type EncryptionAlgorithm string
type EncryptionKey []byte

func (k EncryptionKey) MarshalText() ([]byte, error) {
	return []byte(hex.EncodeToString(k)), nil
}

func (k *EncryptionKey) UnmarshalText(data []byte) error {
	b, err := hex.DecodeString(string(data))
	if err != nil {
		return err
	}
	*k = b
	return nil
}

type Encryption struct {
	Algorithm EncryptionAlgorithm `json:"algorithm" toml:"algorithm"`
	Key       EncryptionKey       `json:"key" toml:"key"`
}

type ZdbBackend struct {
	Address   string `json:"address" toml:"address"`
	Namespace string `json:"namespace" toml:"namespace"`
	Password  string `json:"password" toml:"password"`
}

type QuantumSafeConfig struct {
	Prefix     string       `json:"prefix" toml:"prefix"`
	Encryption Encryption   `json:"encryption" toml:"encryption"`
	Backends   []ZdbBackend `json:"backends" toml:"backends"`
}

type QuantumSafeMeta struct {
	Type   string            `json:"type" toml:"type"`
	Config QuantumSafeConfig `json:"config" toml:"config"`
}

type ZdbGroup struct {
	Backends []ZdbBackend `json:"backends" toml:"backends"`
}

type QuantumCompression struct {
	Algorithm string `json:"algorithm" toml:"algorithm"`
}

type QuantumSafeFSConfig struct {
	MinimalShards     uint32             `json:"minimal_shards" toml:"minimal_shards"`
	ExpectedShards    uint32             `json:"expected_shards" toml:"expected_shards"`
	RedundantGroups   uint32             `json:"redundant_groups" toml:"redundant_groups"`
	RedundantNodes    uint32             `json:"redundant_nodes" toml:"redundant_nodes"`
	MaxZDBDataDirSize uint32             `json:"max_zdb_data_dir_size" toml:"max_zdb_data_dir_size"`
	Encryption        Encryption         `json:"encryption" toml:"encryption"`
	Meta              QuantumSafeMeta    `json:"meta" toml:"meta"`
	Groups            []ZdbGroup         `json:"groups" toml:"groups"`
	Compression       QuantumCompression `json:"compression" toml:"compression"`
}

type QuantumSafeFS struct {
	Cache  uint64              `json:"cache"`
	Config QuantumSafeFSConfig `json:"config"`
}

type QuatumSafeFSResult struct {
	Path            string `json:"path"`
	MetricsEndpoint string `json:"metrics_endpoint"`
}
