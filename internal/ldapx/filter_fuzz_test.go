package ldapx

import (
	"testing"
)

func FuzzValidateFilter(f *testing.F) {
	f.Add("(objectClass=*)")
	f.Add("(&(cn=a)(sn=b))")
	f.Add("(|(cn=a)(!(sn=b)))")

	f.Fuzz(func(t *testing.T, input string) {
		ValidateFilter(input)
	})
}
