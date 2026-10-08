package containers

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// OpenLDAPContainer represents the OpenLDAP test fixture.
type OpenLDAPContainer struct {
	testcontainers.Container
	Host string
	Port int
}

// SetupOpenLDAP runs a bitnamilegacy/openldap container with cn=config enabled.
// The image is pinned and matches server/docker-compose.yml: docker.io/bitnami
// no longer resolves since Bitnami moved its free catalogue in 2025.
func SetupOpenLDAP(ctx context.Context) (*OpenLDAPContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "bitnamilegacy/openldap:2.6.10",
		ExposedPorts: []string{"1389/tcp"},
		Env: map[string]string{
			"LDAP_ADMIN_USERNAME":        "admin",
			"LDAP_ADMIN_PASSWORD":        "adminpassword",
			"LDAP_ROOT":                  "dc=example,dc=org",
			"LDAP_CONFIG_ADMIN_ENABLED":  "yes",
			"LDAP_CONFIG_ADMIN_USERNAME": "configadmin",
			"LDAP_CONFIG_ADMIN_PASSWORD": "configpassword",
		},
		WaitingFor: wait.ForLog("slapd starting").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start OpenLDAP container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "1389")
	if err != nil {
		return nil, fmt.Errorf("failed to get container port: %v", err)
	}

	return &OpenLDAPContainer{
		Container: container,
		Host:      host,
		Port:      int(port.Num()),
	}, nil
}
