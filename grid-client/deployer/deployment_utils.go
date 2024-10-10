package deployer

import (
	"crypto/md5"

	"github.com/pkg/errors"
	zosTypes "github.com/threefoldtech/tfgrid-sdk-go/grid-client/zos"
	"github.com/threefoldtech/zos/pkg/gridtypes/zos"
)

// CountDeploymentPublicIPs counts the public IPs of a deployment
func CountDeploymentPublicIPs(dl zosTypes.Deployment) (uint32, error) {
	var res uint32
	for _, wl := range dl.Workloads {
		if wl.Type == zosTypes.PublicIPType {
			data, err := wl.Workload3().WorkloadData()
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
func HashDeployment(dl zosTypes.Deployment) (string, error) {
	md5Hash := md5.New()
	if err := dl.Challenge(md5Hash); err != nil {
		return "", err
	}
	hash := string(md5Hash.Sum(nil))
	return hash, nil
}

// GetWorkloadHashes returns a mapping between workload name to the workload hash
func GetWorkloadHashes(dl zosTypes.Deployment) (map[string]string, error) {
	hashes := make(map[string]string)

	for _, w := range dl.Workloads {
		key := w.Name
		md5Hash := md5.New()
		if err := w.Challenge(md5Hash); err != nil {
			return nil, errors.Wrapf(err, "could not get a hash for a workload %s", key)
		}
		hash := string(md5Hash.Sum(nil))
		hashes[key] = hash
	}

	return hashes, nil
}

// SameWorkloadsNames compares names of 2 deployments' workloads
func SameWorkloadsNames(d1 zosTypes.Deployment, d2 zosTypes.Deployment) bool {
	if len(d1.Workloads) != len(d2.Workloads) {
		return false
	}

	names := make(map[string]bool)
	for _, w := range d1.Workloads {
		names[w.Name] = true
	}

	for _, w := range d2.Workloads {
		if _, ok := names[w.Name]; !ok {
			return false
		}
	}
	return true
}

// ConstructWorkloadVersions returns a mapping between workload name to the workload version
func ConstructWorkloadVersions(dl zosTypes.Deployment) map[string]uint32 {
	versions := make(map[string]uint32)

	for _, w := range dl.Workloads {
		key := w.Name
		versions[key] = w.Version
	}

	return versions
}

// HasWorkload checks if a deployment contains a given workload
func HasWorkload(dl *zosTypes.Deployment, wlType string) bool {
	for _, wl := range dl.Workloads {
		if wl.Type == wlType {
			return true
		}
	}
	return false
}

// Capacity returns the capacity of a deployment
func Capacity(dl zosTypes.Deployment) (zosTypes.Capacity, error) {
	cap := zosTypes.Capacity{}
	for _, wl := range dl.Workloads {
		wlCap, err := wl.Capacity()
		if err != nil {
			return cap, err
		}
		cap.Add(&wlCap)
	}
	return cap, nil
}
