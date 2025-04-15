package test

import (
	"strings"
	"testing"
)

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

func TestGetRandomItemSubstring(t *testing.T) {
	testItems := []string{"egypt", "belgium", "united states"}

	for i := 0; i < 10; i++ {
		result := getRandomItemSubstring(testItems)
		if result == "" {
			continue
		}

		// is result a substring of any item in testItems? check case insensitive
		found := false
		for _, item := range testItems {
			if strings.Contains(strings.ToLower(item), strings.ToLower(result)) {
				found = true
				break
			}
		}

		if !found {
			t.Errorf("result '%s' is not a substring of any input item", result)
		}
	}

	emptyResult := getRandomItemSubstring([]string{})
	if emptyResult != "" {
		t.Errorf("expected empty string, got '%s'", emptyResult)
	}
}
