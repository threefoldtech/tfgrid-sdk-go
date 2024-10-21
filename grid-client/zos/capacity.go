package zos

type Capacity struct {
	CRU   uint64 `json:"cru"`
	SRU   uint64 `json:"sru"`
	HRU   uint64 `json:"hru"`
	MRU   uint64 `json:"mru"`
	IPV4U uint64 `json:"ipv4u"`
}

// Zero returns true if capacity is zero
func (c *Capacity) Zero() bool {
	return c.CRU == 0 && c.SRU == 0 && c.HRU == 0 && c.MRU == 0 && c.IPV4U == 0
}

// Add increments value of capacity with o
func (c *Capacity) Add(o *Capacity) {
	c.CRU += o.CRU
	c.MRU += o.MRU
	c.SRU += o.SRU
	c.HRU += o.HRU
	c.IPV4U += o.IPV4U
}
