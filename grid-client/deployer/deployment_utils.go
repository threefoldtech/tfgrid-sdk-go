// Package deployer for grid deployer
package deployer

import (
	"crypto/md5"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zos/pkg/gridtypes"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// CountDeploymentPublicIPs counts the public IPs of a deployment
func CountDeploymentPublicIPs(dl gridtypes.Deployment) (uint32, error) {
	var res uint32
	for _, wl := range dl.Workloads {
		if wl.Type == zos.PublicIPType {
			data, err := wl.WorkloadData()
			if err != nil {
				return res, errors.Wrapf(err, "could not parse workload data for workload %s", wl.Name)
			}
			if data.(*zos.PublicIP).V4 {
				res++
			}
		}
	}
	return res, nil
}

// HashDeployment returns deployment hash
func HashDeployment(dl gridtypes.Deployment) (string, error) {
	md5Hash := md5.New()
	if err := dl.Challenge(md5Hash); err != nil {
		return "", err
	}
	hash := string(md5Hash.Sum(nil))
	return hash, nil
}

// GetWorkloadHashes returns a mapping between workload name to the workload hash
func GetWorkloadHashes(dl gridtypes.Deployment) (map[string]string, error) {
	hashes := make(map[string]string)

	for _, w := range dl.Workloads {
		hash, err := ChallengeWorkloadHash(w)
		if err != nil {
			return nil, err
		}

		hashes[w.Name.String()] = hash
	}

	return hashes, nil
}

// ChallengeWorkloadHash returns the string representation of the md5hash of the workload
func ChallengeWorkloadHash(wl gridtypes.Workload) (string, error) {
	md5Hash := md5.New()
	if err := wl.Challenge(md5Hash); err != nil {
		return "", errors.Wrapf(err, "could not get a hash for a workload %s", wl.Name)
	}

	return string(md5Hash.Sum(nil)), nil
}

// SameWorkloadsNames compares names of 2 workload lists
func SameWorkloadsNames(w1 []gridtypes.Workload, w2 []gridtypes.Workload) bool {
	if len(w1) != len(w2) {
		return false
	}

	names := make(map[string]bool)
	for _, w := range w1 {
		names[string(w.Name)] = true
	}

	for _, w := range w2 {
		if _, ok := names[string(w.Name)]; !ok {
			return false
		}
	}

	return true
}

// ConstructWorkloadVersions returns a mapping between workload name to the workload version
func ConstructWorkloadVersions(dl *gridtypes.Deployment) map[string]uint32 {
	versions := make(map[string]uint32)

	for _, w := range dl.Workloads {
		key := string(w.Name)
		versions[key] = w.Version
	}

	return versions
}

// HasWorkload checks if a deployment contains a given workload
func HasWorkload(dl *gridtypes.Deployment, wlType gridtypes.WorkloadType) bool {
	for _, wl := range dl.Workloads {
		if wl.Type == wlType {
			return true
		}
	}
	return false
}

// Capacity returns the capacity of a deployment
func Capacity(dl gridtypes.Deployment) (gridtypes.Capacity, error) {
	cap := gridtypes.Capacity{}
	for _, wl := range dl.Workloads {
		wlCap, err := wl.Capacity()
		if err != nil {
			return cap, err
		}
		cap.Add(&wlCap)
	}
	return cap, nil
}
