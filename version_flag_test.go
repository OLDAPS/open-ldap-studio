package main

import "testing"

func TestVersionRequested(t *testing.T) {
	cases := []struct {
		args []string
		want bool
	}{
		{nil, false},
		{[]string{"--version"}, true},
		{[]string{"-version"}, true},
		{[]string{"--verbose", "--version"}, true},
		{[]string{"--verbosity"}, false},
	}
	for _, c := range cases {
		if got := versionRequested(c.args); got != c.want {
			t.Errorf("versionRequested(%q) = %v, want %v", c.args, got, c.want)
		}
	}
}
