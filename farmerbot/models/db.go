package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redis/redis"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/constants"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/slice"
)

// RedisManager represents interface for redis DB actions
type RedisManager interface {
	GetNode(nodeID uint32) (Node, error)
	GetNodeByTwinID(twinID uint32) (Node, error)
	SetFarm(farm Farm) error
	SetPower(power Power) error
	SetNodes(nodes []Node) error
	SetCPUOverProvision(cpuOP uint8) error
	GetNodes() ([]Node, error)
	GetFarm() (Farm, error)
	GetPower() (Power, error)
	GetCPUOverProvision() (uint8, error)
	UpdatesNode(node Node) error
	Save(config Config) error
	FilterNodesPower(states []PowerState) ([]Node, error)
	FilterAllowedNodesToShutDown() ([]Node, error)
}

// Config is the configuration for farmerbot
type Config struct {
	Farm                    Farm   `json:"farm"`
	Nodes                   []Node `json:"nodes"`
	Power                   Power  `json:"power"`
	DefaultCPUOverProvision uint8  `json:"default_cpu_overprovision"`
}

// RedisDB for saving config for farmerbot
type RedisDB struct {
	redis *redis.Client
}

// NewRedisDB generates new redis db
func NewRedisDB(address string) RedisDB {
	return RedisDB{
		redis: redis.NewClient(&redis.Options{
			Addr: address,
		}),
	}
}

// Save saves the configuration in the database
func (db *RedisDB) Save(config Config) error {
	if err := db.SetFarm(config.Farm); err != nil {
		return err
	}

	if err := db.SetPower(config.Power); err != nil {
		return err
	}

	if config.DefaultCPUOverProvision == 0 {
		config.DefaultCPUOverProvision = constants.DefaultCPUProvision
	}

	if err := db.SetCPUOverProvision(config.DefaultCPUOverProvision); err != nil {
		return err
	}

	return db.SetNodes(config.Nodes)
}

// GetNode gets a node from the database
func (db *RedisDB) GetNode(nodeID uint32) (Node, error) {
	var dest []Node
	nodes, err := db.redis.Get("nodes").Bytes()
	if err != nil {
		return Node{}, err
	}

	if err := json.Unmarshal(nodes, &dest); err != nil {
		return Node{}, err
	}

	for _, n := range dest {
		if n.ID == nodeID {
			return n, nil
		}
	}

	return Node{}, fmt.Errorf("the farmerbot is not managing the node with id %d", nodeID)
}

// GetNodeByTwinID gets a node using twin ID
func (db *RedisDB) GetNodeByTwinID(twinID uint32) (Node, error) {
	var dest []Node
	nodes, err := db.redis.Get("nodes").Bytes()
	if err != nil {
		return Node{}, err
	}

	if err := json.Unmarshal(nodes, &dest); err != nil {
		return Node{}, err
	}

	var resNodes []Node
	for _, n := range dest {
		if n.TwinID == twinID {
			resNodes = append(resNodes, n)
		}
	}

	if len(resNodes) == 0 {
		return Node{}, fmt.Errorf("the farmerbot is not managing the node with twin id %d", twinID)
	} else if len(resNodes) > 1 {
		return Node{}, fmt.Errorf("multiple nodes with twin id %d, that should not be possible", twinID)
	}

	return resNodes[0], nil
}

// SetCPUOverProvision sets default cpu over provision
func (db *RedisDB) SetCPUOverProvision(cpuOP uint8) error {
	f, err := json.Marshal(cpuOP)
	if err != nil {
		return err
	}
	return db.redis.Set("default_cpu_overprovision", f, 0).Err()
}

// SetFarm sets the farm in the database
func (db *RedisDB) SetFarm(farm Farm) error {
	f, err := json.Marshal(farm)
	if err != nil {
		return err
	}
	return db.redis.Set("farm", f, 0).Err()
}

// SetPower sets the power in the database
func (db *RedisDB) SetPower(power Power) error {
	p, err := json.Marshal(power)
	if err != nil {
		return err
	}
	return db.redis.Set("power", p, 0).Err()
}

// SetNodes sets the nodes in the database
func (db *RedisDB) SetNodes(nodes []Node) error {
	n, err := json.Marshal(nodes)
	if err != nil {
		return err
	}

	return db.redis.Set("nodes", n, 0).Err()
}

// GetNodes gets nodes from the database
func (db *RedisDB) GetNodes() ([]Node, error) {
	var dest []Node
	nodes, err := db.redis.Get("nodes").Bytes()
	if err != nil {
		return []Node{}, err
	}

	if err := json.Unmarshal(nodes, &dest); err != nil {
		return []Node{}, err
	}

	return dest, nil
}

// GetPower gets power from the database
func (db *RedisDB) GetPower() (Power, error) {
	var dest Power
	nodes, err := db.redis.Get("power").Bytes()
	if err != nil {
		return Power{}, err
	}

	if err := json.Unmarshal(nodes, &dest); err != nil {
		return Power{}, err
	}

	return dest, nil
}

// GetFarm gets farm from the database
func (db *RedisDB) GetFarm() (Farm, error) {
	var dest Farm
	nodes, err := db.redis.Get("farm").Bytes()
	if err != nil {
		return Farm{}, err
	}

	if err := json.Unmarshal(nodes, &dest); err != nil {
		return Farm{}, err
	}

	return dest, nil
}

// SetCPUOverProvision gets default cpu over provision
func (db *RedisDB) GetCPUOverProvision() (uint8, error) {
	var dest uint8
	nodes, err := db.redis.Get("default_cpu_overprovision").Bytes()
	if err != nil {
		return 0, err
	}

	if err := json.Unmarshal(nodes, &dest); err != nil {
		return 0, err
	}

	return dest, nil
}

// UpdatesNodes adds or updates a node in the database
func (db *RedisDB) UpdatesNode(node Node) error {
	nodes, err := db.GetNodes()
	if err != nil {
		return err
	}

	found := false
	for i, n := range nodes {
		if n.ID == node.ID {
			nodes[i] = node
			found = true
		}
	}

	if !found {
		nodes = append(nodes, node)
	}

	n, err := json.Marshal(nodes)
	if err != nil {
		return err
	}

	return db.redis.Set("nodes", n, 0).Err()
}

// FilterNodesPower filters db ON or OFF nodes
func (db *RedisDB) FilterNodesPower(states []PowerState) ([]Node, error) {
	nodes, err := db.GetNodes()
	if err != nil {
		return []Node{}, errors.New("failed to get nodes from db")
	}

	filtered := make([]Node, 0)
	for _, node := range nodes {
		if slice.Contains(states, node.PowerState) {
			filtered = append(filtered, node)
		}
	}
	return filtered, nil
}

// FilterAllowedNodesToShutDown filters nodes that are allowed to shut down
//
// nodes with public config can't be shutdown
// Do not shutdown a node that just came up (give it some time)
func (db *RedisDB) FilterAllowedNodesToShutDown() ([]Node, error) {
	nodes, err := db.GetNodes()
	if err != nil {
		return []Node{}, errors.New("failed to get nodes from db")
	}

	filtered := make([]Node, 0)
	for _, node := range nodes {
		if node.IsUnused() && !node.PublicConfig && !node.NeverShutDown &&
			time.Since(node.LastTimePowerStateChanged) >= constants.PeriodicWakeUpDuration {
			filtered = append(filtered, node)
		}
	}
	return filtered, nil
}
