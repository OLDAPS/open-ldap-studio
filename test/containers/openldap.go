//go:build integration

package containers

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// What the seeded OpenLDAP fixture is configured with. Tests bind and search
// with these rather than repeating the literals; they match the credentials in
// server/docker-compose.yml, so a developer's manual fixture and the test
// fixture are interchangeable.
const (
	OpenLDAPBaseDN        = "dc=example,dc=org"
	OpenLDAPAdminDN       = "cn=admin,dc=example,dc=org"
	OpenLDAPAdminPassword = "adminpassword"
)

// openLDAPImage is pinned, and is bitnamilegacy/ rather than bitnami/:
// Bitnami moved its free catalogue to the legacy namespace in 2025 and
// docker.io/bitnami/openldap no longer resolves. It matches
// server/docker-compose.yml, so both fixtures seed the same directory.
const openLDAPImage = "bitnamilegacy/openldap:2.6.10"

// OpenLDAPContainer represents the OpenLDAP test fixture.
type OpenLDAPContainer struct {
	testcontainers.Container
	Host string
	Port int
}

// seedDir is server/seed, found relative to this source file so the fixture
// starts from any package's working directory.
func seedDir() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("cannot locate the test/containers source directory")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "server", "seed"), nil
}

// countSeedEntries is how many entries the seed files add: one per line that
// starts with "dn:" (which also matches the base64 form, "dn::"). Folded
// continuation lines start with a space and comments with "#", so neither
// counts.
func countSeedEntries(files []string) (int, error) {
	n := 0
	for _, f := range files {
		fh, err := os.Open(f)
		if err != nil {
			return 0, err
		}
		sc := bufio.NewScanner(fh)
		sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
		for sc.Scan() {
			if bytes.HasPrefix(sc.Bytes(), []byte("dn:")) {
				n++
			}
		}
		err = sc.Err()
		_ = fh.Close()
		if err != nil {
			return 0, fmt.Errorf("reading %s: %w", f, err)
		}
	}
	return n, nil
}

// readyScript succeeds only when the directory holds every seeded entry and
// still does a moment later. The image loads the seed LDIFs into a temporary
// slapd, then restarts it as the real server: a probe that only asks whether
// the base DN answers passes after the first, five-entry file, and the tests
// then see a half-loaded tree (204 of 205 people) or a connection reset by the
// restart. Counting every entry, three times a second apart, rules out both.
func readyScript(entries int) string {
	return fmt.Sprintf(`for i in 1 2 3; do
  n=$(ldapsearch -x -H ldap://127.0.0.1:1389 -D %s -w %s -b %s -s sub -LLL dn 2>/dev/null | grep -c '^dn:')
  [ "$n" -eq %d ] || exit 1
  sleep 1
done`, OpenLDAPAdminDN, OpenLDAPAdminPassword, OpenLDAPBaseDN, entries)
}

// SetupOpenLDAP runs OpenLDAP seeded from server/seed with cn=config enabled,
// and returns once the base DN answers a search. Plain LDAP only: StartTLS and
// LDAPS need the certificate setup that lives in the compose file.
func SetupOpenLDAP(ctx context.Context) (*OpenLDAPContainer, error) {
	dir, err := seedDir()
	if err != nil {
		return nil, err
	}
	seeds, err := filepath.Glob(filepath.Join(dir, "*.ldif"))
	if err != nil || len(seeds) == 0 {
		return nil, fmt.Errorf("no seed LDIF files in %s (err: %v)", dir, err)
	}

	entries, err := countSeedEntries(seeds)
	if err != nil {
		return nil, fmt.Errorf("counting seed entries: %w", err)
	}

	// Applied by the image in name order, which is why the files are numbered.
	files := make([]testcontainers.ContainerFile, 0, len(seeds))
	for _, s := range seeds {
		files = append(files, testcontainers.ContainerFile{
			HostFilePath:      s,
			ContainerFilePath: "/ldifs/" + filepath.Base(s),
			FileMode:          0o644, // slapd runs as uid 1001 and must be able to read them
		})
	}

	req := testcontainers.ContainerRequest{
		Image:        openLDAPImage,
		ExposedPorts: []string{"1389/tcp"},
		Files:        files,
		Env: map[string]string{
			"LDAP_ROOT":                  OpenLDAPBaseDN,
			"LDAP_ADMIN_USERNAME":        "admin",
			"LDAP_ADMIN_PASSWORD":        OpenLDAPAdminPassword,
			"LDAP_CONFIG_ADMIN_ENABLED":  "yes",
			"LDAP_CONFIG_ADMIN_USERNAME": "configadmin",
			"LDAP_CONFIG_ADMIN_PASSWORD": "configpassword",
			"LDAP_SKIP_DEFAULT_TREE":     "yes",
			"LDAP_CUSTOM_LDIF_DIR":       "/ldifs",
			"LDAP_PORT_NUMBER":           "1389",
		},
		// Ready means seeded and stable, not merely answering; see readyScript.
		WaitingFor: wait.ForExec([]string{"sh", "-c", readyScript(entries)}).
			WithStartupTimeout(3 * time.Minute).
			WithPollInterval(2 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start OpenLDAP container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, "1389")
	if err != nil {
		_ = container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get container port: %w", err)
	}

	return &OpenLDAPContainer{
		Container: container,
		Host:      host,
		Port:      int(port.Num()),
	}, nil
}
