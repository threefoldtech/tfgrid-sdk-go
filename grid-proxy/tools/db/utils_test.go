package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMin(t *testing.T) {
	want := uint64(4)
	got := min(4, 5)
	assert.Equal(t, want, got)
}

func TestMax(t *testing.T) {
	want := uint64(5)
	got := max(4, 5)
	assert.Equal(t, want, got)
}

func TestObjectToTupleString(t *testing.T) {
	twin := twin{
		id:           fmt.Sprintf("twin-%d", 1),
		account_id:   fmt.Sprintf("account-id-%d", 1),
		relay:        fmt.Sprintf("relay-%d", 1),
		public_key:   fmt.Sprintf("public-key-%d", 1),
		twin_id:      1,
		grid_version: 3,
	}
	got := objectToTupleString(twin)
	want := "('twin-1', 3, 1, 'account-id-1', 'relay-1', 'public-key-1')"
	assert.Equal(t, want, got)
}
