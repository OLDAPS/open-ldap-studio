package containers

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PoorContainer represents a capability-poor directory server fixture.
type PoorContainer struct {
	testcontainers.Container
	Host string
	Port int
}

// SetupPoorContainer runs a minimal LDAP container with no paging, no readable
// subschema, and no extended operations.
func SetupPoorContainer(ctx context.Context) (*PoorContainer, error) {
	// A lightweight, minimal LDAP server with no advanced capabilities
	req := testcontainers.ContainerRequest{
		Image:        "osixia/openldap:1.5.0",
		ExposedPorts: []string{"389/tcp"},
		Env: map[string]string{
			"LDAP_ADMIN_PASSWORD": "admin",
		},
		WaitingFor: wait.ForLog("slapd starting").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start poor container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "389")
	if err != nil {
		return nil, fmt.Errorf("failed to get container port: %v", err)
	}

	return &PoorContainer{
		Container: container,
		Host:      host,
		Port:      int(port.Num()),
	}, nil
}
