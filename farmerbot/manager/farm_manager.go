// Package manager provides how to manage nodes, farms and power
package manager

import (
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/tfgrid-sdk-go/farmerbot/models"
)

// FarmManager manages farms
type FarmManager struct {
	db models.RedisManager
}

// NewFarmManager creates a new FarmManager
func NewFarmManager(db models.RedisManager) FarmManager {
	return FarmManager{db}
}

// Define defines a farm
func (f *FarmManager) Define(farm models.Farm) error {
	log.Debug().Msgf("[FARM MANAGER] Define farm with ID %d.", farm.ID)
	return f.db.SetFarm(farm)
}
