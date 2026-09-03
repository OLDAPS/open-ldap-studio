package profiles

// Encryption is the transport a profile uses. It is never silently downgraded:
// a StartTLS profile that cannot negotiate TLS fails, it does not continue in
// the clear (FR-009).
type Encryption string

const (
	EncryptionNone     Encryption = "none"
	EncryptionStartTLS Encryption = "startTLS"
	EncryptionLDAPS    Encryption = "ldaps"
)

// BindMethod enumerates the authentication mechanisms (FR-002).
type BindMethod string

const (
	BindAnonymous BindMethod = "anonymous"
	BindSimple    BindMethod = "simple"
	BindExternal  BindMethod = "external"
	BindGSSAPI    BindMethod = "gssapi"
	BindDigestMD5 BindMethod = "digestMD5"
	BindCramMD5   BindMethod = "cramMD5"
)

// AliasPolicy and ReferralPolicy control traversal (FR-021).
type AliasPolicy string

const (
	AliasNever  AliasPolicy = "never"
	AliasSearch AliasPolicy = "search"
	AliasFind   AliasPolicy = "find"
	AliasAlways AliasPolicy = "always"
)

type ReferralPolicy string

const (
	ReferralFollow ReferralPolicy = "follow"
	ReferralIgnore ReferralPolicy = "ignore"
	ReferralAsk    ReferralPolicy = "ask"
)

// TagProduction marks a profile as production. It triggers distinct visual
// treatment and an extra confirmation on every write (research R20).
const TagProduction = "production"

// TLSPolicy is per-profile. Verification defaults on; switching it off is an
// explicit opt-in that is surfaced in the UI whenever the profile is connected
// (FR-006).
type TLSPolicy struct {
	VerifyCertificate bool   `json:"verifyCertificate"`
	VerifyHostname    bool   `json:"verifyHostname"`
	ClientCertRef     string `json:"clientCertRef,omitempty"` // for SASL EXTERNAL
}

// Timeouts are per-profile overrides of the global defaults (FR-099).
type Timeouts struct {
	ConnectMS int `json:"connectMs"`
	ReadMS    int `json:"readMs"`
}

// Limits of 0 mean "use the server's default".
type Limits struct {
	SizeLimit int `json:"sizeLimit"`
	TimeLimit int `json:"timeLimit"`
	PageSize  int `json:"pageSize"`
}

// Profile is a stored connection.
//
// Invariant: Profile has no field capable of holding a secret. The type is the
// enforcement point — there is nowhere to put one. It carries a CredentialID
// pointing at a credentials.Credential, which itself holds only a reference
// into the platform store (FR-003, SC-008, research R10).
type Profile struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	FolderID string `json:"folderId,omitempty"`

	Host       string     `json:"host"`
	Port       int        `json:"port"`
	Encryption Encryption `json:"encryption"`
	TLS        TLSPolicy  `json:"tls"`

	BindMethod BindMethod `json:"bindMethod"`
	// BindDN is raw text, stored and transmitted exactly as typed. It is never
	// normalised (FR-020).
	BindDN string `json:"bindDn"`
	// CredentialID points at a Credential. It is never a secret.
	CredentialID string `json:"credentialId,omitempty"`

	// ReadOnly refuses a preview token at the changeset boundary, below any UI
	// check. It is the only mechanism that protects against the right command
	// in the wrong window (research R20, design gap G3).
	ReadOnly bool     `json:"readOnly"`
	Tags     []string `json:"tags,omitempty"`

	BaseDN    string         `json:"baseDn,omitempty"`
	Timeouts  Timeouts       `json:"timeouts"`
	Limits    Limits         `json:"limits"`
	Aliases   AliasPolicy    `json:"aliases"`
	Referrals ReferralPolicy `json:"referrals"`

	SchemaVersion int `json:"schemaVersion"`
}

// Summary is the list projection the connections view renders.
type Summary struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	FolderID   string     `json:"folderId,omitempty"`
	Host       string     `json:"host"`
	Port       int        `json:"port"`
	Encryption Encryption `json:"encryption"`
	ReadOnly   bool       `json:"readOnly"`
	Tags       []string   `json:"tags,omitempty"`
}

// Folder groups profiles. Cycles are rejected on save.
type Folder struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ParentID string `json:"parentId,omitempty"`
}

// IsProduction reports whether the profile carries the production tag.
func (p Profile) IsProduction() bool {
	for _, t := range p.Tags {
		if t == TagProduction {
			return true
		}
	}
	return false
}

// Summarise projects a Profile onto the list view's shape.
func (p Profile) Summarise() Summary {
	return Summary{
		ID: p.ID, Name: p.Name, FolderID: p.FolderID, Host: p.Host,
		Port: p.Port, Encryption: p.Encryption, ReadOnly: p.ReadOnly, Tags: p.Tags,
	}
}

// DefaultPort is the conventional port for an encryption mode.
func DefaultPort(e Encryption) int {
	if e == EncryptionLDAPS {
		return 636
	}
	return 389
}
