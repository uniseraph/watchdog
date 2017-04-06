package watchdog

import (
	"context"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/client"
	"github.com/docker/go-events"
)

type Manager struct {
	registrator  *Watchdog
	dockerClient client.APIClient
	cancel       context.CancelFunc
}

func NewManager(r *Watchdog, dockerClient client.APIClient) *Manager {
	return &Manager{registrator: r, dockerClient: dockerClient}
}

func (m *Manager) serve(ctx context.Context) error {
	ctx, m.cancel = context.WithCancel(ctx)
	defer m.cancel()

	if err := NewDockerEventWatch(
		events.NewFilter(m.registrator, events.MatcherFunc(EventMatch)),
	).Run(ctx, m.dockerClient); err != nil {
		if err == context.Canceled {
			return nil
		}
		logrus.Error("list watch error: ", err)
		return err
	}

	return nil
}

func (m *Manager) Run(ctx context.Context) error {
	return m.serve(ctx)
}
