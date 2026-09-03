package ldapx

import (
	"encoding/json"
	"testing"
)

func TestSearchDefinitionJSON(t *testing.T) {
	def := SearchDefinition{
		BaseDN:     "dc=example,dc=org",
		Filter:     "(objectClass=person)",
		Scope:      ScopeSubtree,
		Attributes: []string{"cn", "mail"},
		SizeLimit:  1000,
	}

	data, err := json.Marshal(def)
	if err != nil {
		t.Fatalf("failed to marshal SearchDefinition: %v", err)
	}

	var parsed SearchDefinition
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal SearchDefinition: %v", err)
	}

	if parsed.BaseDN != def.BaseDN || parsed.Filter != def.Filter {
		t.Errorf("expected roundtrip equality, got BaseDN: %s, Filter: %s", parsed.BaseDN, parsed.Filter)
	}
	if len(parsed.Attributes) != 2 || parsed.Attributes[0] != "cn" {
		t.Errorf("expected attributes to match, got %v", parsed.Attributes)
	}
}
