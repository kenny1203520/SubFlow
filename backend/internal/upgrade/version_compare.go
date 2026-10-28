package upgrade

import (
	"fmt"
	"strconv"
	"strings"
)

func compareVersions(left, right string) int {
	a, errA := parseVersion(left)
	b, errB := parseVersion(right)
	if errA != nil || errB != nil {
		return strings.Compare(left, right)
	}
	for i := range a.parts {
		if a.parts[i] < b.parts[i] {
			return -1
		}
		if a.parts[i] > b.parts[i] {
			return 1
		}
	}
	if a.pre == b.pre {
		return 0
	}
	if a.pre == "" {
		return 1
	}
	if b.pre == "" {
		return -1
	}
	return strings.Compare(a.pre, b.pre)
}

type parsedVersion struct {
	parts [3]int
	pre   string
}

func parseVersion(value string) (parsedVersion, error) {
	var result parsedVersion
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	if before, after, found := strings.Cut(value, "-"); found {
		value, result.pre = before, after
	}
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return result, fmt.Errorf("invalid semantic version")
	}
	for i, part := range parts {
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return result, fmt.Errorf("invalid semantic version")
		}
		result.parts[i] = number
	}
	return result, nil
}
