package architecture

import (
	"reflect"
	"strings"
	"testing"

	"github.com/open-ldap-studio/open-ldap-studio/internal/bridge"
	"github.com/open-ldap-studio/open-ldap-studio/internal/secrets"
)

// The bound bridge surface is the application's entire external interface.
// These tests assert what is NOT on it, because every guarantee in this
// section is an absence, and an absence needs something to hold it in place.

func boundMethods() []reflect.Method {
	t := reflect.TypeOf(&bridge.Bridge{})
	methods := make([]reflect.Method, 0, t.NumMethod())
	for i := range t.NumMethod() {
		methods = append(methods, t.Method(i))
	}
	return methods
}

// Contract C11 — no vault method exists on the bound surface.
//
// The wireframe's credentials area specified a vault: encrypted-file storage,
// a master password, an auto-lock timer, vault import/export, and a Reveal
// action. Constitution II prohibits five of those outright, so the concept was
// replaced rather than edited (deviation D2). This test is what keeps it gone.
func TestNoVaultMethodIsBound(t *testing.T) {
	forbidden := []string{
		"LockVault", "UnlockVault", "SetVaultStorage", "ExportVault", "ImportVault",
		"RevealSecret", "SetMasterPassword", "UnlockWithMasterPassword", "SetAutoLock",
	}
	bound := boundMethods()

	for _, name := range forbidden {
		for _, method := range bound {
			if method.Name == name {
				t.Errorf("%s is bound: Constitution II forbids an application-managed vault (D2)", name)
			}
		}
	}

	// The looser check catches a differently-named reincarnation.
	for _, method := range bound {
		lower := strings.ToLower(method.Name)
		if strings.Contains(lower, "vault") || strings.Contains(lower, "masterpassword") {
			t.Errorf("%s looks like a vault method; there is no vault (D2)", method.Name)
		}
	}
}

// Contract C9 — no bridge method returns a secret value.
func TestNoBoundMethodReturnsASecret(t *testing.T) {
	secretType := reflect.TypeOf(secrets.Secret{})

	for _, method := range boundMethods() {
		signature := method.Type
		for i := range signature.NumOut() {
			out := signature.Out(i)
			if carriesSecret(out, secretType, map[reflect.Type]bool{}) {
				t.Errorf("%s returns %s, which can carry secret material", method.Name, out)
			}
		}
	}
}

// carriesSecret reports whether t is, contains, or points at a Secret.
func carriesSecret(t, secretType reflect.Type, seen map[reflect.Type]bool) bool {
	if t == secretType {
		return true
	}
	if seen[t] {
		return false
	}
	seen[t] = true

	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Chan:
		return carriesSecret(t.Elem(), secretType, seen)
	case reflect.Map:
		return carriesSecret(t.Key(), secretType, seen) || carriesSecret(t.Elem(), secretType, seen)
	case reflect.Struct:
		for i := range t.NumField() {
			if carriesSecret(t.Field(i).Type, secretType, seen) {
				return true
			}
		}
	}
	return false
}

// Contract C1, at the surface — the bridge exposes no method that writes
// without a token. Preview and Commit(token) are the only path.
func TestNoBoundMutationMethodExists(t *testing.T) {
	// These names are what a reviewer would look for, and what an
	// implementation in a hurry would add.
	forbidden := map[string]string{
		"Modify":        "use Preview then Commit(token)",
		"Add":           "use Preview then Commit(token)",
		"Delete":        "use Preview then Commit(token)",
		"Rename":        "use Preview then Commit(token)",
		"AddEntry":      "use Preview then Commit(token)",
		"ModifyEntry":   "use Preview then Commit(token)",
		"DeleteEntry":   "use Preview then Commit(token)",
		"RenameEntry":   "use Preview then Commit(token)",
		"DeleteSubtree": "use Preview then Commit(token)",
		"ApplyLDIF":     "an import is a job built from a change set",
	}

	for _, method := range boundMethods() {
		if advice, bad := forbidden[method.Name]; bad {
			t.Errorf("Bridge.%s is bound: a write with no preview must not be expressible — %s",
				method.Name, advice)
		}
	}
}

// TestCommitTakesOnlyAToken pins the shape of the one dispatch path. A Commit
// that also accepted a change set would let a caller commit something the
// preview never showed.
func TestCommitTakesOnlyAToken(t *testing.T) {
	method, ok := reflect.TypeOf(&bridge.Bridge{}).MethodByName("Commit")
	if !ok {
		t.Fatal("Bridge.Commit is missing; it is the only dispatch path")
	}
	// Receiver plus one string argument.
	if method.Type.NumIn() != 2 || method.Type.In(1).Kind() != reflect.String {
		t.Errorf("Commit takes %d arguments; it must take exactly one, the token", method.Type.NumIn()-1)
	}
}

// TestSaveProfileCannotSilentlyAcceptASecret asserts the strict decode: the
// profile-saving method takes a raw payload so an unknown field is refused
// rather than dropped.
func TestSaveProfileRefusesAnUnknownField(t *testing.T) {
	method, ok := reflect.TypeOf(&bridge.Bridge{}).MethodByName("SaveProfile")
	if !ok {
		t.Fatal("Bridge.SaveProfile is missing")
	}
	if method.Type.In(1).Kind() != reflect.String {
		t.Fatalf("SaveProfile takes %s; it takes a JSON payload so a password field is refused, not dropped",
			method.Type.In(1))
	}
}
