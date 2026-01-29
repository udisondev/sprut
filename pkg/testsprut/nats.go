package testsprut

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// natsContainer обёртка над NATS testcontainer.
type natsContainer struct {
	container testcontainers.Container
	url       string
}

// startNATS запускает NATS контейнер.
func startNATS(ctx context.Context) (*natsContainer, error) {
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "nats:latest",
			ExposedPorts: []string{"4222/tcp"},
			WaitingFor:   wait.ForListeningPort("4222/tcp").WithStartupTimeout(30 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		return nil, fmt.Errorf("start NATS container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		terminateContainer(ctx, container)
		return nil, fmt.Errorf("get NATS host: %w", err)
	}

	port, err := container.MappedPort(ctx, "4222")
	if err != nil {
		terminateContainer(ctx, container)
		return nil, fmt.Errorf("get NATS port: %w", err)
	}

	return &natsContainer{
		container: container,
		url:       fmt.Sprintf("nats://%s:%s", host, port.Port()),
	}, nil
}

// URL возвращает NATS URL для подключения.
func (n *natsContainer) URL() string {
	return n.url
}

// Terminate останавливает NATS контейнер.
func (n *natsContainer) Terminate(ctx context.Context) error {
	if n.container == nil {
		return nil
	}
	return n.container.Terminate(ctx)
}

func terminateContainer(ctx context.Context, c testcontainers.Container) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	_ = c.Terminate(ctx)
}
