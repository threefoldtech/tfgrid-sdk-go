// Package workloads includes workloads types (vm, zdb, QSFS, public IP, gateway name, gateway fqdn, disk)
package workloads

import (
	"encoding/json"

	"github.com/pkg/errors"
)

// ToMap converts workload data to a map (dict)
func ToMap(workload interface{}) (map[string]interface{}, error) {
	var wlMap map[string]interface{}
	bytes, err := json.Marshal(workload)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal workload data")
	}

	err = json.Unmarshal(bytes, &wlMap)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal workload bytes to map")
	}

	return wlMap, nil
}

// NewWorkloadFromMap converts a map (dict) to a workload
func NewWorkloadFromMap(wlMap map[string]interface{}, result interface{}) (interface{}, error) {
	mapBytes, err := json.Marshal(wlMap)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal map")
	}

	err = json.Unmarshal(mapBytes, result)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal data")
	}

	return result, nil
}
