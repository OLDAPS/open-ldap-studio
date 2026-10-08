package logging

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Contract F8 — logs rotate at 10 MB x 3 and redact before writing.

func TestRedactLDIFValueAndItsContinuationLines(t *testing.T) {
	r := NewRedactor()
	in := "dn: cn=jrivera,ou=people,dc=example,dc=com\n" +
		"changetype: modify\n" +
		"replace: userPassword\n" +
		"userPassword:: e1NTSEF9c2VjcmV0dmFsdWU=\n" +
		" IGZvbGRlZCBjb250aW51YXRpb24=\n" +
		"-\n" +
		"result: 0\n"

	got := string(r.Redact([]byte(in)))

	if strings.Contains(got, "e1NTSEF9") || strings.Contains(got, "IGZvbGRlZA") {
		t.Fatalf("secret survived redaction:\n%s", got)
	}
	if !strings.Contains(got, "userPassword: "+Placeholder) {
		t.Fatalf("expected a redacted userPassword line, got:\n%s", got)
	}
	// The DN and the result code must survive — redacting them would defeat
	// the log's purpose (error-model.md § Redaction).
	if !strings.Contains(got, "dn: cn=jrivera,ou=people,dc=example,dc=com") {
		t.Errorf("DN was redacted; it must not be:\n%s", got)
	}
	if !strings.Contains(got, "result: 0") {
		t.Errorf("result code was redacted; it must not be:\n%s", got)
	}
}

func TestRedactIgnoresAttributeOptionsAndCase(t *testing.T) {
	r := NewRedactor()
	got := string(r.Redact([]byte("UserPassword;binary:: c2VjcmV0\n")))
	if strings.Contains(got, "c2VjcmV0") {
		t.Fatalf("option-bearing sensitive attribute was not redacted: %s", got)
	}
	if !strings.Contains(got, "UserPassword;binary: "+Placeholder) {
		t.Fatalf("attribute description was not preserved: %s", got)
	}
}

func TestRedactJSONEventPayload(t *testing.T) {
	r := NewRedactor()
	in := `{"profileId":"p1","bindDN":"cn=admin","password":"hunter2","saslCredentials":"AAEC"}`
	got := string(r.Redact([]byte(in)))
	for _, secret := range []string{"hunter2", "AAEC"} {
		if strings.Contains(got, secret) {
			t.Fatalf("secret %q survived JSON redaction: %s", secret, got)
		}
	}
	if !strings.Contains(got, `"bindDN":"cn=admin"`) {
		t.Errorf("bind DN was redacted; it must not be: %s", got)
	}
}

func TestUserMarkedAttributeIsRedacted(t *testing.T) {
	r := NewRedactor("employeeNumber")
	got := string(r.Redact([]byte("employeeNumber: 12345\n")))
	if strings.Contains(got, "12345") {
		t.Fatalf("user-marked attribute was not redacted: %s", got)
	}
}

func TestSinkRedactsBeforeTheBytesReachDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "modification.log")
	s, err := OpenSink(path, NewRedactor())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Write([]byte("userPassword: hunter2\n")); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}

	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(onDisk, []byte("hunter2")) {
		t.Fatalf("raw secret reached disk: %s", onDisk)
	}
}

func TestSinkWriteReportsTheCallersLength(t *testing.T) {
	s, err := OpenSink(filepath.Join(t.TempDir(), "a.log"), NewRedactor())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	p := []byte("userPassword: hunter2\n")
	n, err := s.Write(p)
	if err != nil {
		t.Fatal(err)
	}
	// Redaction shortens the record; reporting the shorter count would look
	// like a short write to every io.Writer caller, slog included.
	if n != len(p) {
		t.Fatalf("Write returned %d, want %d", n, len(p))
	}
}

func TestSinkRotatesAtItsLimitAndKeepsThreeBackups(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "application.log")
	s, err := OpenSink(path, NewRedactor())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()

	// Shrink the limit rather than write 40 MB; the policy under test is the
	// rotation and retention behaviour, not the constant.
	s.maxBytes = 1024

	record := bytes.Repeat([]byte("x"), 200)
	for range 40 {
		if _, err := s.Write(append(record, '\n')); err != nil {
			t.Fatal(err)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != KeptBackups+1 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("got %d files %v, want %d (current + %d backups)",
			len(entries), names, KeptBackups+1, KeptBackups)
	}
	if _, err := os.Stat(path + ".4"); !os.IsNotExist(err) {
		t.Errorf("a fourth backup was kept; retention is %d", KeptBackups)
	}
}

func TestSinkConstantsMatchThePolicy(t *testing.T) {
	if MaxBytes != 10<<20 {
		t.Errorf("MaxBytes = %d, want 10 MB", MaxBytes)
	}
	if KeptBackups != 3 {
		t.Errorf("KeptBackups = %d, want 3", KeptBackups)
	}
}
