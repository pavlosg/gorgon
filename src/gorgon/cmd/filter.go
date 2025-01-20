package cmd

import (
	"strings"

	"github.com/pavlosg/gorgon/src/gorgon/wildcard"
)

type Filter struct {
	match   []wildcard.Matcher
	exclude []wildcard.Matcher
}

func MakeFilter(match, exclude string) (filter Filter) {
	for _, p := range strings.Split(match, "|") {
		filter.match = append(filter.match, wildcard.Compile(p))
	}
	if len(exclude) != 0 {
		for _, p := range strings.Split(exclude, "|") {
			filter.exclude = append(filter.exclude, wildcard.Compile(p))
		}
	}
	return
}

func (filter Filter) Match(subject string) bool {
	matched := false
	for _, m := range filter.match {
		if m.Match(subject) {
			matched = true
			break
		}
	}
	if !matched {
		return false
	}
	for _, m := range filter.exclude {
		if m.Match(subject) {
			return false
		}
	}
	return true
}
