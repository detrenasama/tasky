package ui

import (
	"testing"
)

func TestWrapText(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"одно слово", "одно\nслово"},
		{"aaa bbb", "aaa bbb"},
		{"aaa bbb ccc", "aaa bbb\nccc"},
		{"оченьдлинноеслово", "оченьдл\nинноесл\nово"},
		{"aaa\nbbb", "aaa\nbbb"},
	}
	for _, c := range cases {
		got := wrapText(c.in, 7)
		if got != c.want {
			t.Errorf("wrapText(%q, 7) = %q, ожидалось %q", c.in, got, c.want)
		}
	}
}

func TestTruncateWEnd(t *testing.T) {
	cases := []struct {
		in, want string
		w        int
	}{
		{"abc", "abc", 5},
		{"abcdef", "abcd…", 5},
		{"оченьдлинноеназвание", "оченьд…", 7},
		{"", "", 4},
		{"abc", "…", 1},
	}
	for _, c := range cases {
		got := truncateWEnd(c.in, c.w)
		if got != c.want {
			t.Errorf("truncateWEnd(%q, %d) = %q, ожидалось %q", c.in, c.w, got, c.want)
		}
	}
}
