//go:build integration

package bridge

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/open-ldap-studio/open-ldap-studio/test/containers"
)

// TestMain starts one seeded OpenLDAP for the whole package run. The live tests
// only read the directory, so sharing it is safe and avoids paying the import
// of the seed once per test. A machine without Docker fails here, loudly,
// rather than skipping tests that are supposed to prove the directory works.
func TestMain(m *testing.M) {
	os.Exit(runWithDirectory(m))
}

func runWithDirectory(m *testing.M) int {
	ctx := context.Background()

	dir, err := containers.SetupOpenLDAP(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot start the OpenLDAP fixture (is Docker running?): %v\n", err)
		return 1
	}
	defer func() { _ = dir.Terminate(ctx) }()

	liveHost, livePort = dir.Host, dir.Port
	return m.Run()
}
