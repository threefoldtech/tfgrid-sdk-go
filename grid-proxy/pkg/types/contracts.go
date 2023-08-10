package types

import "encoding/json"

type ContractDetails interface {
	RentContractDetails | NameContractDetails | NodeContractDetails
}

type NodeContractDetails struct {
	NodeID            uint   `json:"node_id"`
	DeploymentData    string `json:"deployment_data"`
	DeploymentHash    string `json:"deployment_hash"`
	NumberOfPublicIps uint   `json:"number_of_public_ips"`
}

type NameContractDetails struct {
	Name string `json:"name"`
}

type RentContractDetails struct {
	NodeID uint `json:"node_id"`
}

// Contract represents a contract and its details after decoding to one of Details structs.
type Contract struct {
	ContractID uint        `json:"contract_id"`
	TwinID     uint        `json:"twin_id"`
	State      string      `json:"state"`
	CreatedAt  uint        `json:"created_at"`
	Type       string      `json:"type"`
	Details    interface{} `json:"details"`
}

// RawContract represents a contract and its details in json RawMessage before decoding.
type RawContract struct {
	ContractID uint            `json:"contract_id"`
	TwinID     uint            `json:"twin_id"`
	State      string          `json:"state"`
	CreatedAt  uint            `json:"created_at"`
	Type       string          `json:"type"`
	Details    json.RawMessage `json:"details"`
}

// ContractBilling is contract billing info
type ContractBilling struct {
	AmountBilled     uint64 `json:"amount_billed"`
	DiscountReceived string `json:"discount_received"`
	Timestamp        uint64 `json:"timestamp"`
}

// ContractFilter contract filters
type ContractFilter struct {
	ContractID        *uint64 `schema:"contract_id,omitempty"`
	TwinID            *uint64 `schema:"twin_id,omitempty"`
	NodeID            *uint64 `schema:"node_id,omitempty"`
	Type              *string `schema:"type,omitempty"`
	State             *string `schema:"state,omitempty"`
	Name              *string `schema:"name,omitempty"`
	NumberOfPublicIps *uint64 `schema:"number_of_public_ips,omitempty"`
	DeploymentData    *string `schema:"deployment_data,omitempty"`
	DeploymentHash    *string `schema:"deployment_hash,omitempty"`
}
