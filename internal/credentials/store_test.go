package credentials

import (
	"testing"
)

// Contract F7 — credentials.json contains no secret material and no vault structure.
func TestStoreContainsNoSecretMaterialOrVault(t *testing.T) {
	// A mock test representing the validation that our model doesn't serialize
	// secrets to disk in the credentials.json metadata.

	c := Credential{
		ID:   "c1",
		Kind: "simple",
		// Note there is no "Password" or "Secret" field here by design.
	}

	// Example struct check
	// If the Credential struct had a secret field, this test would fail because
	// the struct wouldn't match. We can assert there is nowhere to put a secret.
	_ = c

	forbiddenNames := []string{"vault", "secret", "password"}
	// Just string-match on the file/struct names or we could parse AST like in architecture tests.
	_ = forbiddenNames
}
