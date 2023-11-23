// Package models for farmerbot models.
package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var cap = Capacity{
	CRU: 1,
	SRU: 1,
	MRU: 1,
	HRU: 1,
}

func TestCapacityModel(t *testing.T) {
	assert.False(t, cap.isEmpty())

	resultSub := cap.subtract(cap)
	assert.True(t, resultSub.isEmpty())

	cap.Add(cap)
	assert.Equal(t, cap.CRU, uint64(2))
}
