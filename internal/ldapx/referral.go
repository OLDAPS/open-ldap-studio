package ldapx

// ReferralTarget represents an LDAP URL parsed for referrals.
type ReferralTarget struct {
	Host     string
	Port     int
	BaseDN   string
	Scope    string
	Filter   string
	Extension string
}
