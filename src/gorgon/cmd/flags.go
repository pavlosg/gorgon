package cmd

import (
	"fmt"
	"sort"
	"strconv"
)

type Flags struct {
	flags map[string]flagDescr
}

type flagDescr struct {
	description string
	value       interface{}
	required    bool
}

func (flags *Flags) Optional(flag string, description string, value interface{}) {
	flags.add(flag, description, value, false)
}

func (flags *Flags) Required(flag string, description string, value interface{}) {
	flags.add(flag, description, value, true)
}

func (flags *Flags) Parse(args []string) bool {
	found := make(map[string]int)
	for i := 0; i < len(args); i++ {
		flag := args[i]
		descr, ok := flags.flags[flag]
		if !ok {
			fmt.Printf("Unexpected argument: %q\n", flag)
			flags.PrintOptions()
			return false
		}
		if _, ok := found[flag]; ok {
			fmt.Printf("Duplicate flag: %q\n", flag)
			return false
		}
		found[flag] = i
		switch value := descr.value.(type) {
		case *bool:
			*value = true
		case *int:
			i++
			v, err := strconv.Atoi(args[i])
			if err != nil {
				fmt.Printf("Error parsing int from %q %q: %v\n", flag, args[i], err)
				return false
			}
			*value = v
		case *float64:
			i++
			v, err := strconv.ParseFloat(args[i], 64)
			if err != nil {
				fmt.Printf("Error parsing float from %q %q: %v\n", flag, args[i], err)
				return false
			}
			*value = v
		case *string:
			i++
			*value = args[i]
		default:
			_ = value
		}
	}
	missing := false
	for flag, descr := range flags.flags {
		if !descr.required {
			continue
		}
		if _, ok := found[flag]; !ok {
			fmt.Printf("Not found required flag %q\n", flag)
			missing = true
		}
	}
	if missing {
		flags.PrintOptions()
		return false
	}
	return true
}

func (flags *Flags) PrintOptions() {
	var lines [][2]string
	maxWidth := 0
	for flag, descr := range flags.flags {
		left := flag
		switch v := descr.value.(type) {
		case *int:
			left += " I"
		case *float64:
			left += " F"
		case *string:
			left += " S"
		default:
			_ = v
		}
		w := len(left)
		if maxWidth < w {
			maxWidth = w
		}
		lines = append(lines, [2]string{left, descr.description})
	}
	sort.Slice(lines, func(i, j int) bool { return lines[i][0] < lines[j][0] })
	fmt.Print("Options:\n\n")
	for _, line := range lines {
		fmt.Printf("  %-*s  %s\n", maxWidth, line[0], line[1])
	}
}

func (flags *Flags) add(flag string, description string, value interface{}, required bool) {
	switch v := value.(type) {
	case *bool:
	case *int:
	case *float64:
	case *string:
	default:
		_ = v
		panic("Flag not bool/int/float64/string: " + flag)
	}
	if flag[0] != '-' {
		panic("Flag without '-': " + flag)
	}
	if _, exists := flags.flags[flag]; exists {
		panic("Flag exists: " + flag)
	}
	if flags.flags == nil {
		flags.flags = make(map[string]flagDescr)
	}
	flags.flags[flag] = flagDescr{description, value, required}
}
