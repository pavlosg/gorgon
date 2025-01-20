package cmd

import "fmt"

func EscapeFileName(name string) string {
	n := len(name)
	esc := make([]byte, 0, n)
	for i := 0; i < n; i++ {
		c := name[i]
		if r, ok := specialChars[c]; ok {
			esc = append(esc, r...)
		} else {
			esc = append(esc, c)
		}
	}
	return string(esc)
}

var specialChars map[byte]string

func init() {
	specialChars = make(map[byte]string, 60)
	for i := byte(0); i < 127; i++ {
		if i < 32 || i == '^' || i == '<' || i == '>' || i == ':' || i == '"' ||
			i == '/' || i == '\\' || i == '|' || i == '?' || i == '*' {
			specialChars[i] = fmt.Sprintf("^%02X", i)
		}
	}
}
