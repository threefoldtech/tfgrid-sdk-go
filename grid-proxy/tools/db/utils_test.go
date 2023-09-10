package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMin(t *testing.T) {
	want := uint64(4)
	got := min(4, 5)
	assert.Equal(t, want, got)
}

func TestRnd(t *testing.T) {
	_, got := rnd(2, 1)
	assert.Error(t, got)
}

func TestMax(t *testing.T) {
	want := uint64(5)
	got := max(4, 5)
	assert.Equal(t, want, got)
}

func TestObjectToTupleString(t *testing.T) {

	t.Run("test without null values", func(t *testing.T) {
		twin := twin{
			id:           fmt.Sprintf("twin-%d", 1),
			account_id:   fmt.Sprintf("account-id-%d", 1),
			relay:        fmt.Sprintf("relay-%d", 1),
			public_key:   fmt.Sprintf("public-key-%d", 1),
			twin_id:      1,
			grid_version: 3,
		}
		got, err := objectToTupleString(twin)
		assert.Nil(t, err)
		want := "('twin-1', 3, 1, 'account-id-1', 'relay-1', 'public-key-1')"
		assert.Equal(t, want, got)
	})

	t.Run("test with null values", func(t *testing.T) {
		createdAt := uint64(time.Now().Unix())
		contract := node_contract{
			id:                    fmt.Sprintf("node-contract-%d", 1),
			twin_id:               1,
			contract_id:           1,
			state:                 "Created", // Simulating a nil value
			created_at:            createdAt,
			node_id:               1,
			deployment_data:       "deployment-data-1",
			deployment_hash:       "deployment-hash-1",
			number_of_public_i_ps: 0,
			grid_version:          3,
			resources_used_id:     "",
		}

		got, err := objectToTupleString(contract)
		assert.Nil(t, err)
		want := fmt.Sprintf("('node-contract-1', 3, 1, 1, 1, 'deployment-data-1', 'deployment-hash-1', 0, 'Created', %d, NULL)", createdAt)
		assert.Equal(t, want, got)

	})
}
