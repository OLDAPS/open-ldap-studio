package history

import (
	"testing"
)

func TestAddSearchRecord(t *testing.T) {
	record := SearchRecord{
		ID:        "123",
		ProfileID: "prof-1",
		Filter:    "(cn=*)",
		BaseDN:    "dc=example,dc=org",
		Timestamp: 1000000,
	}

	err := AddSearch(record)
	if err != nil {
		t.Fatalf("AddSearch failed: %v", err)
	}
}
