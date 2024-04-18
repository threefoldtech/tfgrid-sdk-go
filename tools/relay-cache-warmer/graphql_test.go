package main

import (
	"reflect"
	"testing"
)

func TestInvalidateTwins(t *testing.T) {
	twins := []graphqlTwin{
		{
			Relay: nil,
		},
		{
			Relay: strPointer("192.168.1.1"),
		},
		{
			Relay: strPointer("invalid"),
		},
		{
			Relay: strPointer(".."),
		},
		{
			Relay: strPointer("::1_302:9e63:7d43:b742:2442:f506:5aa4:d5c5"),
		},
		{
			Relay: strPointer("example.com_relay.grid.tf_relay.bknd1.ninja.com_relay.02.grid.tf"),
		},
	}
	expected := []graphqlTwin{
		{
			Relay: nil,
		},
		{
			Relay: nil,
		},
		{
			Relay: nil,
		},
		{
			Relay: nil,
		},
		{
			Relay: nil,
		},
		{
			Relay: strPointer("example.com_relay.grid.tf_relay.bknd1.ninja.com_relay.02.grid.tf"),
		},
	}
	invalidateTwins(twins)
	if !reflect.DeepEqual(twins, expected) {
		t.Fatalf("expected %+v got %+v", expected, twins)
	}
}
func strPointer(s string) *string {
	return &s
}
