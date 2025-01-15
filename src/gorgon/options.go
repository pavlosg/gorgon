package gorgon

import "time"

type Object = map[string]interface{}

type Options struct {
	Extras           Object
	Concurrency      int
	WorkloadDuration time.Duration
	Nodes            []string
}

func (opt *Options) GetExtraString(key string) (s string, ok bool) {
	o, ok := opt.Extras[key]
	if !ok {
		return
	}
	s, ok = o.(string)
	return
}
