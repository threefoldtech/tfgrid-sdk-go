package zos

import (
	"io"

	"github.com/threefoldtech/zos_base/pkg/gridtypes"
)

type ZosDeployment interface {
	Sign(twin uint32, sk Signer) error
	Valid() error
	ChallengeHash() ([]byte, error)
	Challenge(w io.Writer) error
}

type Signer interface {
	Sign(msg []byte) ([]byte, error)
	Type() string
}

// Deployment structure
type Deployment struct {
	// Version must be set to 0 on deployment creation. And then it has to
	// be incremented with each call to update.
	Version uint32 `json:"version"`
	// TwinID is the id of the twin sending the deployment. A twin then can only
	// `get` status about deployments he owns.
	TwinID uint32 `json:"twin_id"`
	// ContractID the contract must be "pre created" on substrate before the deployment is
	// sent to the node. The node will then validate that this deployment hash, will match the
	// hash attached to this contract.
	// the flow should go as follows:
	// - fill in ALL deployment details (metadata, and workloads)
	// - calculate the deployment hash (by calling ChallengeHash method)
	// - create the contract with the right hash
	// - set the contract id on the deployment object
	// - send deployment to node.
	ContractID uint64 `json:"contract_id"`
	// Metadata is user specific meta attached to deployment, can be used to link this
	// deployment to other external systems for automation
	Metadata string `json:"metadata"`
	// Description is human readable description of the deployment
	Description string `json:"description"`
	// Expiration [deprecated] is not used
	Expiration int64 `json:"expiration"`
	// SignatureRequirement specifications
	SignatureRequirement SignatureRequirement `json:"signature_requirement"`
	// Workloads is a list of workloads associated with this deployment
	Workloads []Workload `json:"workloads"`
}

// this means that twin3 must sign + one of either (twin1 or twin2) to have the right signature weight
type SignatureRequirement struct {
	Requests       []SignatureRequest `json:"requests"`
	WeightRequired uint               `json:"weight_required"`
	Signatures     []Signature        `json:"signatures"`
	SignatureStyle string             `json:"signature_style"`
}

// SignatureRequest struct a signature request of a twin
type SignatureRequest struct {
	TwinID   uint32 `json:"twin_id"`
	Required bool   `json:"required"`
	Weight   uint   `json:"weight"`
}

// Signature struct
type Signature struct {
	TwinID        uint32 `json:"twin_id"`
	Signature     string `json:"signature"`
	SignatureType string `json:"signature_type"`
}

// WorkloadWithID wrapper around workload type
// that holds the global workload ID
// Note: you never need to construct this manually
type WorkloadWithID struct {
	*Workload
	ID WorkloadID
}

type WorkloadID string

func NewGridDeployment(twin uint32, workloads []Workload) Deployment {
	return Deployment{
		Version:   0,
		TwinID:    twin, // LocalTwin,
		Workloads: workloads,
		SignatureRequirement: SignatureRequirement{
			WeightRequired: 1,
			Requests: []SignatureRequest{
				{
					TwinID: twin,
					Weight: 1,
				},
			},
		},
	}
}

func (d *Deployment) Valid() error {
	return d.zosDeployment().Valid()
}

func (d *Deployment) Sign(twin uint32, sk Signer) error {

	dl := d.zosDeployment()
	err := dl.Sign(twin, sk)
	if err != nil {
		return err
	}

	d.copySignature(dl)
	return nil
}

func (d *Deployment) ChallengeHash() ([]byte, error) {
	return d.zosDeployment().ChallengeHash()
}

func (d *Deployment) Challenge(w io.Writer) error {
	return d.zosDeployment().Challenge(w)
}

// Get a workload by name
func (d *Deployment) Get(name string) (*WorkloadWithID, error) {

	w, err := d.zosDeployment().Get(gridtypes.Name(name))
	if err != nil {
		return nil, err
	}

	workload := NewWorkloadFromZosWorkload(*w.Workload)
	return &WorkloadWithID{
		Workload: &workload,
		ID:       WorkloadID(w.ID),
	}, nil
}

func (d *Deployment) ByType(typ ...string) []*WorkloadWithID {
	var workloadsWithID []*WorkloadWithID

	var types []gridtypes.WorkloadType
	for _, t := range typ {
		types = append(types, gridtypes.WorkloadType(t))
	}
	wls := d.zosDeployment().ByType(types...)

	for _, w := range wls {
		workload := NewWorkloadFromZosWorkload(*w.Workload)
		workloadsWithID = append(workloadsWithID, &WorkloadWithID{
			Workload: &workload,
			ID:       WorkloadID(w.ID),
		})
	}

	return workloadsWithID
}

func (d *Deployment) zosDeployment() *gridtypes.Deployment {
	var requests []gridtypes.SignatureRequest
	var signatures []gridtypes.Signature
	var workloads []gridtypes.Workload

	for _, wl := range d.Workloads {
		workloads = append(workloads, *wl.Workload())
	}

	for _, req := range d.SignatureRequirement.Requests {
		requests = append(requests, gridtypes.SignatureRequest(req))
	}

	for _, sign := range d.SignatureRequirement.Signatures {
		signatures = append(signatures, gridtypes.Signature(sign))
	}

	return &gridtypes.Deployment{
		Version:     d.Version,
		TwinID:      d.TwinID,
		ContractID:  d.ContractID,
		Metadata:    d.Metadata,
		Description: d.Description,
		Expiration:  gridtypes.Timestamp(d.Expiration),
		SignatureRequirement: gridtypes.SignatureRequirement{
			Requests:       requests,
			WeightRequired: d.SignatureRequirement.WeightRequired,
			Signatures:     signatures,
			SignatureStyle: gridtypes.SignatureStyle(d.SignatureRequirement.SignatureStyle),
		},
		Workloads: workloads,
	}
}

func (d *Deployment) copySignature(dl *gridtypes.Deployment) {
	for _, sign := range dl.SignatureRequirement.Signatures {
		d.SignatureRequirement.Signatures = append(
			d.SignatureRequirement.Signatures, Signature{
				TwinID:        sign.TwinID,
				Signature:     sign.Signature,
				SignatureType: sign.SignatureType,
			})
	}
}
