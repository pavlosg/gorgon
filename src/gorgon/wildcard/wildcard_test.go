package wildcard

import "testing"

func checkMatch(t *testing.T, pattern, subject string) {
	if !Compile(pattern).Match(subject) {
		t.Errorf("pattern %q should match %q", pattern, subject)
	}
}

func checkNotMatch(t *testing.T, pattern, subject string) {
	if Compile(pattern).Match(subject) {
		t.Errorf("pattern %q should not match %q", pattern, subject)
	}
}

func TestWildcards(t *testing.T) {
	for _, pattern := range []string{
		"",
		"*",
		"**",
		"***",
	} {
		checkMatch(t, pattern, "")
	}

	for _, pattern := range []string{
		"*",
		"this*",
		"*this*",
		"this**",
		"*this**",
		"**this**",
		"*test",
		"*test*",
		"**test",
		"**test*",
		"**test**",
		"*is*is*",
		"th*is*is*",
		"*is*is*st",
		"th*is*is*st",
		"t*is*is*t",
		"*th*is*is*st*",
		"*th*is**is*st*",
		"this is a test*",
		"*this is a test",
		"*this is a test*",
		"*this i*s a test*",
	} {
		checkMatch(t, pattern, "this is a test")
		if pattern != "*" {
			checkNotMatch(t, pattern, "")
		}
	}

	for _, pattern := range []string{
		"",
		"t",
		"this",
		"test",
		"test*",
		"test**",
		"test**test",
		"*this",
		"**this",
		"this**this",
		"test**this",
		"test*this",
		"*test*this*",
		"this*this is a test",
		"this is a test*test",
		"*no*",
		"**no**",
		"**si**",
	} {
		checkNotMatch(t, pattern, "this is a test")
	}
}

func BenchmarkWildcard(b *testing.B) {
	n := b.N
	m := Compile("*quick*fox*dog")
	for i := 0; i < n; i++ {
		if !m.Match("The quick brown fox jumped over the lazy dog") {
			b.Fatalf("should match")
		}
	}
}
