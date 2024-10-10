package zos

// ZDB namespace creation info
type ZDB struct {
	Size     uint64 `json:"size"`
	Mode     string `json:"mode"`
	Password string `json:"password"`
	Public   bool   `json:"public"`
}

// ZDBResult is the information return to the BCDB
// after deploying a 0-db namespace
type ZDBResult struct {
	Namespace string
	IPs       []string
	Port      uint
}
