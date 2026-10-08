package ldif

import "testing"

func FuzzLDIFReader(f *testing.F) {
	f.Add("dn: cn=admin")
	f.Fuzz(func(t *testing.T, input string) {
		// Just a placeholder
	})
}
