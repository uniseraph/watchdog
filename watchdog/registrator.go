package watchdog

import (
	"context"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	apievents "github.com/docker/docker/api/types/events"
	"github.com/docker/docker/client"
	"github.com/docker/go-events"

	"github.com/omega/watchdog/backends"
)

func EventMatch(event events.Event) bool {
	ev, ok := event.(apievents.Message)
	if !ok {
		return false
	}

	if ev.Type != apievents.ContainerEventType {
		return false
	}
	return ev.Action == StateRestore || ev.Action == StateStart || ev.Action == StateDie
}

type Watchdog struct {
	backend      backends.ContainerBackend
	dockerClient client.APIClient
	cache        *Cache
	refreshTTL   time.Duration

	//getServiceNameAndTags GetRegistratorService

	closed   chan struct{}
	shutdown chan struct{}
	adds     chan *types.ContainerJSON
	removes  chan *types.ContainerJSON
	once     sync.Once
}

func NewWatchdog(backend backends.ContainerBackend, dockerClient client.APIClient, options map[string]string) *Watchdog {
	r := Watchdog{
		backend:      backend,
		dockerClient: dockerClient,
		cache:        NewContanerCache(),
		refreshTTL:   5*time.Minute,
		closed:       make(chan struct{}),
		shutdown:     make(chan struct{}),
		adds:         make(chan *types.ContainerJSON),
		removes:      make(chan *types.ContainerJSON),
	}


	go r.run()

	return &r
}

func (r *Watchdog) run() {
	defer close(r.closed)

	r.tick()
	ticker := time.NewTicker(r.refreshTTL)


	for {
		select {
		case <-ticker.C:
			logrus.Debug("Tick to synchronize container cache...")
			if err := r.tick(); err != nil {
				logrus.Error("Tick error: ", err)
			}
		case c := <-r.adds:
			logrus.Infof("Registering container %v ", c.ID[0:6])
			if err := r.backend.Register(c); err != nil {
				logrus.Error(err)
				continue
			}
			r.cache.Add(c)
		//	logrus.Infof("Registered container successfully %v : %#v", c.ID[0:6], c)
		case c := <-r.removes:
			logrus.Infof("Deregistering container %v : %#v", c.ID[0:6], c)
			if err := r.backend.Deregister(c); err != nil {
				logrus.Error(err)
				continue
			}
			r.cache.Remove(c)
		//	logrus.Infof("Deregistered container successfully %v : %#v", c.ID[0:6], c)
		case <-r.shutdown:
			logrus.Debug("Stopping registrator ...")
			ticker.Stop()
			return
		}
	}
}

func (r *Watchdog) tick() error {
	logrus.Debug("statring tick ....")
	containers, err := r.dockerClient.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		logrus.Error("list containers error: ", err)
		return err
	}

	//logrus.Debugf("existing containers number is %#v" , len(containers))

	cs := make([]*types.ContainerJSON, 0, len(containers))
	for _, container := range containers {
		if s , err := r.dockerClient.ContainerInspect(context.Background(),container.ID) ; err==nil {
			logrus.Debugf("existing containers is id:%#v , %#v" , container.ID[0:6] , s)
			cs = append(cs, &s)
		}

	}

	services, err := r.backend.Containers(r.dockerClient)
	if err != nil {
		logrus.Error("list services error: ", err)
		return err
	}
	r.cache.Reset(services)

	added, deleted := r.cache.Diff(cs)
	for _, add := range added {
		logrus.Infof("Registering container %v", add.ID[0:6] )
		if err := r.backend.Register(add); err != nil {
			logrus.Error(err)
			return err
		}
		//logrus.Infof("Registered container successfully id:%v", add.ID[0:6])
	}
	for _, rm := range deleted {
		logrus.Infof("Deregistering container %v ", rm.ID[0:6])
		if err := r.backend.Deregister(rm); err != nil {
			logrus.Error(err)
			return err
		}
		//logrus.Infof("Deregistered container successfully id:%v , %#v",rm.ID[0:6], rm)
	}
	r.cache.Reset(cs)
	logrus.Debug("ending tick ...")
	return nil
}

func (r *Watchdog) Write(event events.Event) error {
	ev, ok := event.(apievents.Message)
	if !ok {
		return nil
	}
	logrus.Debugf("Receive an event %#v ...", ev)

	c := r.cache.Get(ev.Actor.ID)



	if c == nil {
	  	c1, err := r.dockerClient.ContainerInspect(context.Background(), ev.Actor.ID)
	  	if err != nil {
	      		return nil
	  	}

		c =&c1
	}

	select {
	case <-r.closed:
		return events.ErrSinkClosed
	default:
		switch ev.Action {
		case "start":
			r.adds <- c
		case "die":
			r.removes <- c
		}
	}

	return nil
}

func (r *Watchdog) Close() error {
	r.once.Do(func() {
		close(r.shutdown)
	})
	<-r.closed
	logrus.Info("Watchdog is stopped")
	return nil
}
