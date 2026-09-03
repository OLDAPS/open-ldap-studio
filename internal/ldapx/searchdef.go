package ldapx

// SearchDefinition describes an advanced search request (T131).
type SearchDefinition struct {
	BaseDN             string   `json:"baseDN"`
	Filter             string   `json:"filter"`
	Scope              Scope    `json:"scope"`
	Attributes         []string `json:"attributes"`
	IncludeOperational bool     `json:"includeOperational"`
	SizeLimit          int      `json:"sizeLimit"`
	TimeLimit          int      `json:"timeLimit"`
	Aliases            string   `json:"aliases"`
	ManageDsaIT        bool     `json:"manageDsaIT"`
}
