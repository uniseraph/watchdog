package watchdog

import (
	"context"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/go-events"
)

const (
	StateRestore = "restore"
	StateStart   = "start"
	StateDie     = "die"
)

type DockerEventWatch struct {
	sink events.Sink
}

func NewDockerEventWatch(sink events.Sink) *DockerEventWatch {
	l := &DockerEventWatch{
		sink: sink,
	}
	return l
}

func (l *DockerEventWatch) Run(ctx context.Context, dockerClient client.APIClient) error {
	logrus.Info("Listen docker events ...")

	messageCh, errorCh := dockerClient.Events(ctx, types.EventsOptions{})

	for {
		select {
		case message := <-messageCh:
			if err := l.sink.Write(message); err != nil {
				logrus.Errorf("fail to deliver message %#v, error: %v", message, err)
			}
		case err := <-errorCh:
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				logrus.Error("get docker events failed, error: ", err)
				messageCh, errorCh = dockerClient.Events(ctx, types.EventsOptions{})
			}
		}
	}
}
