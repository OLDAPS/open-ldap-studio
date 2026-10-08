//go:build integration

package containers

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// ApacheDSContainer represents the ApacheDS test fixture.
type ApacheDSContainer struct {
	testcontainers.Container
	Host string
	Port int
}

// SetupApacheDS runs an ApacheDS container with X.500 prescriptiveACI enabled.
func SetupApacheDS(ctx context.Context) (*ApacheDSContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "marcelocg/apacheds:latest",
		ExposedPorts: []string{"10389/tcp"},
		WaitingFor:   wait.ForLog("starting ApacheDS").WithStartupTimeout(60 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start ApacheDS container: %v", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get container host: %v", err)
	}

	port, err := container.MappedPort(ctx, "10389")
	if err != nil {
		return nil, fmt.Errorf("failed to get container port: %v", err)
	}

	return &ApacheDSContainer{
		Container: container,
		Host:      host,
		Port:      int(port.Num()),
	}, nil
}
