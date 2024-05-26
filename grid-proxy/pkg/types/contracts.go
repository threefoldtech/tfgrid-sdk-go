package types

import "encoding/json"

// ContractDetails represent details for all contract types
type ContractDetails interface {
	RentContractDetails | NameContractDetails | NodeContractDetails
}

// NodeContractDetails node contract details
type NodeContractDetails struct {
	NodeID            uint   `json:"nodeId"`
	DeploymentData    string `json:"deployment_data"`
	DeploymentHash    string `json:"deployment_hash"`
	NumberOfPublicIps uint   `json:"number_of_public_ips"`
	FarmName          string `json:"farm_name"`
	FarmId            uint64 `json:"farm_id"`
}

// NameContractDetails name contract details
type NameContractDetails struct {
	Name string `json:"name"`
}

// RentContractDetails rent contract details
type RentContractDetails struct {
	NodeID   uint   `json:"nodeId"`
	FarmName string `json:"farm_name"`
	FarmId   uint64 `json:"farm_id"`
}

// Contract represents a contract and its details after decoding to one of Details structs.
type Contract struct {
	ContractID uint        `json:"contract_id" sort:"contract_id"`
	TwinID     uint        `json:"twin_id" sort:"twin_id"`
	State      string      `json:"state" sort:"state"`
	CreatedAt  uint        `json:"created_at" sort:"created_at"`
	Type       string      `json:"type" sort:"type"`
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
	AmountBilled     uint64 `json:"amountBilled"`
	DiscountReceived string `json:"discountReceived"`
	Timestamp        uint64 `json:"timestamp"`
}

// ContractFilter contract filters
type ContractFilter struct {
	ContractID        *uint64  `schema:"contract_id,omitempty"`
	TwinID            *uint64  `schema:"twin_id,omitempty"`
	NodeID            *uint64  `schema:"node_id,omitempty"`
	Type              *string  `schema:"type,omitempty"`
	State             []string `schema:"state,omitempty"`
	Name              *string  `schema:"name,omitempty"`
	NumberOfPublicIps *uint64  `schema:"number_of_public_ips,omitempty"`
	DeploymentData    *string  `schema:"deployment_data,omitempty"`
	DeploymentHash    *string  `schema:"deployment_hash,omitempty"`
	FarmName          *string  `schema:"farm_name,omitempty"`
	FarmId            *uint64  `schema:"farm_id,omitempty"`
}
