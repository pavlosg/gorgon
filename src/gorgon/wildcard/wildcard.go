package wildcard

import "strings"

type Matcher struct {
	parts []string
	empty bool
}

func Compile(pattern string) Matcher {
	return Matcher{strings.Split(pattern, "*"), len(pattern) == 0}
}

func (m Matcher) Match(subject string) bool {
	if m.empty {
		return len(subject) == 0
	}
	parts := m.parts
	if len(parts) == 1 {
		return subject == parts[0]
	}
	if n := len(parts[0]); n != 0 {
		if !strings.HasPrefix(subject, parts[0]) {
			return false
		}
		subject = subject[n:]
	}
	k := len(parts) - 1
	for i := 1; i < k; i++ {
		n := len(parts[i])
		if n == 0 {
			continue
		}
		idx := strings.Index(subject, parts[i])
		if idx < 0 {
			return false
		}
		subject = subject[idx+n:]
	}
	if len(parts[k]) == 0 {
		return true
	}
	return strings.HasSuffix(subject, parts[k])
}
