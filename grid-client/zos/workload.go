package zos

import (
	"encoding/json"
	"io"
	"slices"

	"github.com/threefoldtech/zos/pkg/gridtypes"
	gridtypes4 "github.com/threefoldtech/zos4/pkg/gridtypes"
)

type Workload struct {
	// Version is version of reservation object. On deployment creation, version must be 0
	// then only workloads that need to be updated must match the version of the deployment object.
	// if a deployment update message is sent to a node it does the following:
	// - validate deployment version
	// - check workloads list, if a version is not matching the new deployment version, the workload is untouched
	// - if a workload version is same as deployment, the workload is "updated"
	// - if a workload is removed, the workload is deleted.
	Version uint32 `json:"version"`
	//Name is unique workload name per deployment  (required)
	Name string `json:"name"`
	// Type of the reservation (container, zdb, vm, etc...)
	Type string `json:"type"`
	// Data is the reservation type arguments.
	Data json.RawMessage `json:"data"`
	// Metadata is user specific meta attached to deployment, can be used to link this
	// deployment to other external systems for automation
	Metadata string `json:"metadata"`
	//Description human readable description of the workload
	Description string `json:"description"`
	// Result of reservation, set by the node
	Result Result `json:"result"`
}

// Result is the struct filled by the node
// after a reservation object has been processed
type Result struct {
	// Time when the result is sent
	Created int64 `json:"created"`
	// State of the deployment (ok,error)
	State ResultState `json:"state"`
	// if State is "error", then this field contains the error
	// otherwise it's nil
	Error string `json:"message"`
	// Data is the information generated by the provisioning of the workload
	// its type depend on the reservation type
	Data json.RawMessage `json:"data"`
}

// Unmarshal a shortcut for json.Unmarshal
func (r *Result) Unmarshal(v interface{}) error {
	return json.Unmarshal(r.Data, v)
}

// ResultState type
type ResultState string

func (s ResultState) IsAny(state ...ResultState) bool {
	return slices.Contains(state, s)
}

func (s ResultState) IsOkay() bool {
	return s.IsAny(StateOk, StatePaused)
}

// MustMarshal is a utility function to quickly serialize workload data
func MustMarshal(data interface{}) json.RawMessage {
	bytes, err := json.Marshal(data)
	if err != nil {
		panic(err)
	}

	return json.RawMessage(bytes)
}

func NewWorkloadFromZosWorkload(wl gridtypes.Workload) Workload {
	return Workload{
		Version:     wl.Version,
		Name:        wl.Name.String(),
		Type:        wl.Type.String(),
		Data:        wl.Data,
		Metadata:    wl.Metadata,
		Description: wl.Description,
		Result: Result{
			Created: int64(wl.Result.Created),
			State:   ResultState(wl.Result.State),
			Error:   wl.Result.Error,
			Data:    wl.Result.Data,
		},
	}
}

func NewWorkloadFromZosWorkload4(wl gridtypes4.Workload) Workload {
	return Workload{
		Version:     wl.Version,
		Name:        wl.Name.String(),
		Type:        wl.Type.String(),
		Data:        wl.Data,
		Metadata:    wl.Metadata,
		Description: wl.Description,
		Result: Result{
			Created: int64(wl.Result.Created),
			State:   ResultState(wl.Result.State),
			Error:   wl.Result.Error,
			Data:    wl.Result.Data,
		},
	}
}

func (wl *Workload) Workload3() *gridtypes.Workload {
	return &gridtypes.Workload{
		Version:     wl.Version,
		Name:        gridtypes.Name(wl.Name),
		Type:        gridtypes.WorkloadType(wl.Type),
		Data:        wl.Data,
		Metadata:    wl.Metadata,
		Description: wl.Description,
		Result: gridtypes.Result{
			Created: gridtypes.Timestamp(wl.Result.Created),
			State:   gridtypes.ResultState(wl.Result.State),
			Error:   wl.Result.Error,
			Data:    wl.Result.Data,
		},
	}
}

func (wl *Workload) Workload4() *gridtypes4.Workload {
	return &gridtypes4.Workload{
		Version:     wl.Version,
		Name:        gridtypes4.Name(wl.Name),
		Type:        gridtypes4.WorkloadType(wl.Type),
		Data:        wl.Data,
		Metadata:    wl.Metadata,
		Description: wl.Description,
		Result: gridtypes4.Result{
			Created: gridtypes4.Timestamp(wl.Result.Created),
			State:   gridtypes4.ResultState(wl.Result.State),
			Error:   wl.Result.Error,
			Data:    wl.Result.Data,
		},
	}
}

func (wl *Workload) Challenge(w io.Writer) error {
	if wl.Type == NetworkLightType || wl.Type == ZMachineLightType {
		return wl.Workload4().Challenge(w)
	}

	return wl.Workload3().Challenge(w)
}

// Capacity returns the used capacity by this workload
func (wl *Workload) Capacity() (Capacity, error) {
	if wl.Type == NetworkLightType || wl.Type == ZMachineLightType {
		cap, err := wl.Workload4().Capacity()
		return Capacity{
			CRU:   cap.CRU,
			SRU:   uint64(cap.SRU),
			HRU:   uint64(cap.HRU),
			MRU:   uint64(cap.MRU),
			IPV4U: cap.IPV4U,
		}, err
	}

	cap, err := wl.Workload3().Capacity()
	return Capacity{
		CRU:   cap.CRU,
		SRU:   uint64(cap.SRU),
		HRU:   uint64(cap.HRU),
		MRU:   uint64(cap.MRU),
		IPV4U: cap.IPV4U,
	}, err
}

func (w Workload) WithResults(result Result) Workload {
	w.Result = result
	return w
}
