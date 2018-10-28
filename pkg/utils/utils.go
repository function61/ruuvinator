package utils

import (
	"regexp"
	"strings"
)

var splitStringIntoGroupsOfTwoRe = regexp.MustCompile("(.{2})")

func SplitStringIntoGroupsOfTwo(input string, join string) string {
	matchesOnly := []string{}
	matches := splitStringIntoGroupsOfTwoRe.FindAllStringSubmatch(input, -1)

	for _, item := range matches {
		matchesOnly = append(matchesOnly, item[0])
	}

	return strings.Join(matchesOnly, join)
}
