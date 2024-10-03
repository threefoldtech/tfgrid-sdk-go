package parser

import (
	"fmt"
	"strconv"
	"strings"
)

func ParseStorage(storage string) (uint64, error) {
	var multiplier uint64 = 1
	var numString, unit string

	storage = strings.ToLower(strings.TrimSpace(storage))

	if !strings.HasSuffix(storage, "mb") && !strings.HasSuffix(storage, "gb") && !strings.HasSuffix(storage, "tb") {
		unit = "mb"
		numString = storage
	} else {
		unit = storage[len(storage)-2:]
		numString = storage[:len(storage)-2]
	}

	number, err := strconv.ParseUint(numString, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number format %s", numString)
	}

	switch unit {
	case "mb":
		multiplier = 1
	case "gb":
		multiplier = 1024
	case "tb":
		multiplier = 1024 * 1024
	default:
		return 0, fmt.Errorf("unsupported storage unit %s", unit)
	}

	return (number * multiplier), nil
}
