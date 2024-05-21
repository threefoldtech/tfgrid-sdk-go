package test

import "testing"

func TestGetRandomSliceLengthInts(t *testing.T) {
	inputSlice := []int{1, 2, 3, 4, 5}
	length := 3
	result := getRandomSliceFrom(inputSlice, length)
	if len(result) != length {
		t.Errorf("expected length %d, got %d", length, len(result))
	}
}

func TestGetRandomSliceLengthEmpty(t *testing.T) {
	inputSlice := []string{}
	length := 0
	result := getRandomSliceFrom(inputSlice, length)
	if len(result) != length {
		t.Errorf("expected length %d, got %d", length, len(result))
	}
}
