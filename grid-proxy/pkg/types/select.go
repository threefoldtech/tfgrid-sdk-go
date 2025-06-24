package types

import (
	"net/url"
	"reflect"
	"strings"
)

// parseSelectFields parses comma-separated select fields and sets corresponding boolean flags
func ParseSelectFields(params url.Values, selectStruct interface{}) error {
	selectValues, exists := params["select"]
	if !exists || len(selectValues) == 0 {
		return nil
	}

	selectedFields := strings.Split(selectValues[0], ",")
	normalizedFields := make(map[string]bool)
	for _, field := range selectedFields {
		field = strings.TrimSpace(field)
		if field != "" {
			// Convert camelCase to snake_case
			normalizedField := camelToSnake(field)
			normalizedFields[normalizedField] = true
		}
	}

	v := reflect.ValueOf(selectStruct).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		structField := t.Field(i)

		tag := structField.Tag.Get("schema")
		if tag != "" {
			fieldName := strings.Split(tag, ",")[0]
			if normalizedFields[fieldName] {
				if field.Kind() == reflect.Bool && field.CanSet() {
					field.SetBool(true)
				}
			}
		}
	}

	return nil
}

// camelToSnake converts camelCase to snake_case
func camelToSnake(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// isSelectType checks if the given interface is a select type
func IsSelectType(type_ interface{}) bool {
	t := reflect.TypeOf(type_)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	typeName := t.Name()
	return strings.HasSuffix(typeName, "Select")
}


// filterNodesResponse filters node response based on selected fields
func FilterNodesResponse(nodes []Node, nodeSelect NodeSelect) []map[string]interface{} {
	if !nodeSelect.HasSelection() {
		result := make([]map[string]interface{}, len(nodes))
		for i, node := range nodes {
			result[i] = typesNodeToMap(node)
		}
		return result
	}

	result := make([]map[string]interface{}, len(nodes))
	for i, node := range nodes {
		filteredNode := make(map[string]interface{})
		
		if nodeSelect.ID {
			filteredNode["id"] = node.ID
		}
		if nodeSelect.NodeID {
			filteredNode["nodeId"] = node.NodeID
		}
		if nodeSelect.FarmID {
			filteredNode["farmId"] = node.FarmID
		}
		if nodeSelect.FarmName {
			filteredNode["farmName"] = node.FarmName
		}
		if nodeSelect.TwinID {
			filteredNode["twinId"] = node.TwinID
		}
		if nodeSelect.Country {
			filteredNode["country"] = node.Country
		}
		if nodeSelect.GridVersion {
			filteredNode["gridVersion"] = node.GridVersion
		}
		if nodeSelect.City {
			filteredNode["city"] = node.City
		}
		if nodeSelect.Uptime {
			filteredNode["uptime"] = node.Uptime
		}
		if nodeSelect.Created {
			filteredNode["created"] = node.Created
		}
		if nodeSelect.FarmingPolicyID {
			filteredNode["farmingPolicyId"] = node.FarmingPolicyID
		}
		if nodeSelect.UpdatedAt {
			filteredNode["updatedAt"] = node.UpdatedAt
		}
		if nodeSelect.TotalResources {
			filteredNode["total_resources"] = node.TotalResources
		}
		if nodeSelect.UsedResources {
			filteredNode["used_resources"] = node.UsedResources
		}
		if nodeSelect.Location {
			filteredNode["location"] = node.Location
		}
		if nodeSelect.PublicConfig {
			filteredNode["publicConfig"] = node.PublicConfig
		}
		if nodeSelect.Status {
			filteredNode["status"] = node.Status
		}
		if nodeSelect.CertificationType {
			filteredNode["certificationType"] = node.CertificationType
		}
		if nodeSelect.Dedicated {
			filteredNode["dedicated"] = node.Dedicated
		}
		if nodeSelect.InDedicatedFarm {
			filteredNode["inDedicatedFarm"] = node.InDedicatedFarm
		}
		if nodeSelect.RentContractID {
			filteredNode["rentContractId"] = node.RentContractID
		}
		if nodeSelect.Rented {
			filteredNode["rented"] = node.Rented
		}
		if nodeSelect.Rentable {
			filteredNode["rentable"] = node.Rentable
		}
		if nodeSelect.RentedByTwinID {
			filteredNode["rentedByTwinId"] = node.RentedByTwinID
		}
		if nodeSelect.SerialNumber {
			filteredNode["serialNumber"] = node.SerialNumber
		}
		if nodeSelect.Power {
			filteredNode["power"] = node.Power
		}
		if nodeSelect.NumGPU {
			filteredNode["num_gpu"] = node.NumGPU
		}
		if nodeSelect.ExtraFee {
			filteredNode["extraFee"] = node.ExtraFee
		}
		if nodeSelect.Healthy {
			filteredNode["healthy"] = node.Healthy
		}
		if nodeSelect.Dmi {
			filteredNode["dmi"] = node.Dmi
		}
		if nodeSelect.Speed {
			filteredNode["speed"] = node.Speed
		}
		if nodeSelect.CpuBenchmark {
			filteredNode["cpu_benchmark"] = node.CpuBenchmark
		}
		if nodeSelect.GPUs {
			filteredNode["gpus"] = node.GPUs
		}
		if nodeSelect.PriceUsd {
			filteredNode["price_usd"] = node.PriceUsd
		}
		if nodeSelect.FarmFreeIps {
			filteredNode["farm_free_ips"] = node.FarmFreeIps
		}
		if nodeSelect.Features {
			filteredNode["features"] = node.Features
		}
		
		result[i] = filteredNode
	}
	
	return result
}

// filterFarmsResponse filters farm response based on selected fields
func FilterFarmsResponse(farms []Farm, farmSelect FarmSelect) []map[string]interface{} {
	if !farmSelect.HasSelection() {
		// No selection specified, return all data (convert to map for consistency)
		result := make([]map[string]interface{}, len(farms))
		for i, farm := range farms {
			result[i] = typesFarmToMap(farm)
		}
		return result
	}

	result := make([]map[string]interface{}, len(farms))
	for i, farm := range farms {
		filteredFarm := make(map[string]interface{})
		
		if farmSelect.Name {
			filteredFarm["name"] = farm.Name
		}
		if farmSelect.FarmID {
			filteredFarm["farmId"] = farm.FarmID
		}
		if farmSelect.TwinID {
			filteredFarm["twinId"] = farm.TwinID
		}
		if farmSelect.PricingPolicyID {
			filteredFarm["pricingPolicyId"] = farm.PricingPolicyID
		}
		if farmSelect.CertificationType {
			filteredFarm["certificationType"] = farm.CertificationType
		}
		if farmSelect.StellarAddress {
			filteredFarm["stellarAddress"] = farm.StellarAddress
		}
		if farmSelect.Dedicated {
			filteredFarm["dedicated"] = farm.Dedicated
		}
		if farmSelect.PublicIps {
			filteredFarm["publicIps"] = farm.PublicIps
		}
		
		result[i] = filteredFarm
	}
	
	return result
}

// typesNodeToMap converts a Node to a map for JSON serialization
func typesNodeToMap(node Node) map[string]interface{} {
	return map[string]interface{}{
		"id":                node.ID,
		"nodeId":            node.NodeID,
		"farmId":            node.FarmID,
		"farmName":          node.FarmName,
		"twinId":            node.TwinID,
		"country":           node.Country,
		"gridVersion":       node.GridVersion,
		"city":              node.City,
		"uptime":            node.Uptime,
		"created":           node.Created,
		"farmingPolicyId":   node.FarmingPolicyID,
		"updatedAt":         node.UpdatedAt,
		"total_resources":   node.TotalResources,
		"used_resources":    node.UsedResources,
		"location":          node.Location,
		"publicConfig":      node.PublicConfig,
		"status":            node.Status,
		"certificationType": node.CertificationType,
		"dedicated":         node.Dedicated,
		"inDedicatedFarm":   node.InDedicatedFarm,
		"rentContractId":    node.RentContractID,
		"rented":            node.Rented,
		"rentable":          node.Rentable,
		"rentedByTwinId":    node.RentedByTwinID,
		"serialNumber":      node.SerialNumber,
		"power":             node.Power,
		"num_gpu":           node.NumGPU,
		"extraFee":          node.ExtraFee,
		"healthy":           node.Healthy,
		"dmi":               node.Dmi,
		"speed":             node.Speed,
		"cpu_benchmark":     node.CpuBenchmark,
		"gpus":              node.GPUs,
		"price_usd":         node.PriceUsd,
		"farm_free_ips":     node.FarmFreeIps,
		"features":          node.Features,
	}
}

// typesFarmToMap converts a Farm to a map for JSON serialization
func typesFarmToMap(farm Farm) map[string]interface{} {
	return map[string]interface{}{
		"farmId":            farm.FarmID,
		"name":              farm.Name,
		"twinId":            farm.TwinID,
		"pricingPolicyId":   farm.PricingPolicyID,
		"stellarAddress":    farm.StellarAddress,
		"dedicated":         farm.Dedicated,
		"certificationType": farm.CertificationType,
		"publicIps":         farm.PublicIps,
	}
}

// filterTwinsResponse filters twin response based on selected fields
func FilterTwinsResponse(twins []Twin, twinSelect TwinSelect) []map[string]interface{} {
	result := make([]map[string]interface{}, len(twins))
	
	for i, twin := range twins {
		if !twinSelect.HasSelection() {
			result[i] = twinToMap(twin)
			continue
		}
		
		filteredTwin := make(map[string]interface{})
		
		if twinSelect.TwinID {
			filteredTwin["twinId"] = twin.TwinID
		}
		if twinSelect.AccountID {
			filteredTwin["accountId"] = twin.AccountID
		}
		if twinSelect.Relay {
			filteredTwin["relay"] = twin.Relay
		}
		if twinSelect.PublicKey {
			filteredTwin["publicKey"] = twin.PublicKey
		}
		
		result[i] = filteredTwin
	}
	
	return result
}

// twinToMap converts a Twin to a map for JSON serialization
func twinToMap(twin Twin) map[string]interface{} {
	return map[string]interface{}{
		"twinId":    twin.TwinID,
		"accountId": twin.AccountID,
		"relay":     twin.Relay,
		"publicKey": twin.PublicKey,
	}
}

// filterContractsResponse filters contract response based on selected fields
func FilterContractsResponse(contracts []Contract, contractSelect ContractSelect) []map[string]interface{} {
	result := make([]map[string]interface{}, len(contracts))
	
	for i, contract := range contracts {
		if !contractSelect.HasSelection() {
			result[i] = contractToMap(contract)
			continue
		}
		
		filteredContract := make(map[string]interface{})
		
		if contractSelect.ContractID {
			filteredContract["contractId"] = contract.ContractID
		}
		if contractSelect.TwinID {
			filteredContract["twinId"] = contract.TwinID
		}
		if contractSelect.State {
			filteredContract["state"] = contract.State
		}
		if contractSelect.CreatedAt {
			filteredContract["createdAt"] = contract.CreatedAt
		}
		if contractSelect.Type {
			filteredContract["type"] = contract.Type
		}
		if contractSelect.Details {
			filteredContract["details"] = contract.Details
		}
		
		result[i] = filteredContract
	}
	
	return result
}

// contractToMap converts a Contract to a map for JSON serialization
func contractToMap(contract Contract) map[string]interface{} {
	return map[string]interface{}{
		"contractId": contract.ContractID,
		"twinId":     contract.TwinID,
		"state":      contract.State,
		"createdAt":  contract.CreatedAt,
		"type":       contract.Type,
		"details":    contract.Details,
	}
}

// filterPublicIPsResponse filters public IP response based on selected fields
func FilterPublicIPsResponse(ips []PublicIP, publicIPSelect PublicIPSelect) []map[string]interface{} {
	result := make([]map[string]interface{}, len(ips))
	
	for i, ip := range ips {
		if !publicIPSelect.HasSelection() {
			result[i] = publicIPToMap(ip)
			continue
		}
		
		filteredIP := make(map[string]interface{})
		
		if publicIPSelect.ID {
			filteredIP["id"] = ip.ID
		}
		if publicIPSelect.IP {
			filteredIP["ip"] = ip.IP
		}
		if publicIPSelect.Gateway {
			filteredIP["gateway"] = ip.Gateway
		}
		if publicIPSelect.ContractID {
			filteredIP["contractId"] = ip.ContractID
		}
		if publicIPSelect.FarmID {
			filteredIP["farmId"] = ip.FarmID
		}
		
		result[i] = filteredIP
	}
	
	return result
}

// publicIPToMap converts a PublicIP to a map for JSON serialization
func publicIPToMap(ip PublicIP) map[string]interface{} {
	return map[string]interface{}{
		"id":         ip.ID,
		"ip":         ip.IP,
		"gateway":    ip.Gateway,
		"contractId": ip.ContractID,
		"farmId":     ip.FarmID,
	}
}