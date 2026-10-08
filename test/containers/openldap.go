//go:build integration

package containers

import (
	"context"
	"fmt"
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
		// A search that succeeds, not a log line: "slapd starting" appears
		// before the seed has been imported, and a directory that is up but
		// empty would fail the first test that reads from it.
		WaitingFor: wait.ForExec([]string{
			"ldapsearch", "-x", "-H", "ldap://127.0.0.1:1389",
			"-D", OpenLDAPAdminDN, "-w", OpenLDAPAdminPassword,
			"-b", OpenLDAPBaseDN, "-s", "base", "(objectClass=*)", "dn",
		}).WithStartupTimeout(3 * time.Minute).WithPollInterval(2 * time.Second),
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
