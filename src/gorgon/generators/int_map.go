package generators

import (
	"fmt"
	"strings"
)

type IntMap struct {
	m map[string]int
}

func (im IntMap) Get(key string) (i int, ok bool) {
	if im.m == nil {
		return 0, false
	}
	i, ok = im.m[key]
	return
}

func (im IntMap) Put(key string, value int) IntMap {
	ret := make(map[string]int, len(im.m))
	for k, v := range im.m {
		ret[k] = v
	}
	ret[key] = value
	return IntMap{ret}
}

func (im IntMap) Equals(other IntMap) bool {
	a := im.m
	b := other.m
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; ok {
			if v != w {
				return false
			}
		} else {
			return false
		}
	}
	for k, v := range b {
		if w, ok := a[k]; ok {
			if v != w {
				return false
			}
		} else {
			return false
		}
	}
	return true
}

func (im IntMap) String() string {
	var sb strings.Builder
	sb.WriteByte('{')
	first := true
	for k, v := range im.m {
		if first {
			first = false
		} else {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%q: %d", k, v))
	}
	sb.WriteByte('}')
	return sb.String()
}
