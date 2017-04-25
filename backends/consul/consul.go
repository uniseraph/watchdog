package consul

import (
	"github.com/hashicorp/consul/api"

	"github.com/omega/watchdog/backends"
	"github.com/docker/docker/api/types"
	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/client"
	"context"
)

func init() {
	if err := backends.Register("consul", New); err != nil {
		panic(err)
	}
}

type backend struct {
	client *api.Client
	getServiceNameAndTags GetRegistratorService
}

func New(address string,options map[string]string) (backends.ContainerBackend, error) {
	client, err := api.NewClient(&api.Config{Address: address})
	if err != nil {
		return nil, err
	}

	b:=&backend{
		client: client ,
		getServiceNameAndTags:GetRegistratorService(getComposeServiceNameAndTags),
	}

	if options != nil {
		if v, exists := options["registrator.service.getter"]; exists {
			switch v {
			case "name":
				b.getServiceNameAndTags = GetRegistratorService(getContainerServiceNameAndTags)
			case "docker-compose":
				b.getServiceNameAndTags = GetRegistratorService(getComposeServiceNameAndTags)
			default:
				logrus.Warnf("invalid service getter %v, use 'docker-compose' by default", v)
			}
		}
	}

	return b, nil
}

func (b *backend) Register(c *types.ContainerJSON) error {

	logrus.Debugf("consul backend:registering container %v",  c.ID[0:6] )

	s := b.ContainerToService(c)

	if s==nil {

		//logrus.Infof("consul backend:ignore the container %v ", c.ID[0:6] )
		return nil
	}


	err := b.client.Agent().ServiceRegister(BackendServiceToAgentService( s))

	if err!=nil {
		logrus.Infof("consul backend : registery to consul error : %s" , err.Error())
	}

	return err
}

func (b *backend) Deregister(c *types.ContainerJSON) error {
	logrus.Infof("consul backend:deregistering container %v",  c.ID[0:6] )

	services, err := b.client.Agent().Services()
	if err != nil {
		return err
	}
	if _, exists := services[c.ID]; exists {
		return b.client.Agent().ServiceDeregister(c.ID)
	}
	return nil
}

func (b *backend) Containers(cli client.APIClient) ([]*types.ContainerJSON, error) {
	services, err := b.client.Agent().Services()
	if err != nil {
		return nil, err
	}

	result := make([]*types.ContainerJSON, 0, len(services))
	for _, service := range services {
		if service.Service == "consul" { // default consul raft service
			continue
		}


		if c , err := cli.ContainerInspect(context.Background(),service.ID) ; err==nil {
			result = append(result, &c)
		}
	}
	return result, nil
}

func BackendServiceToAgentService(s *Service) *api.AgentServiceRegistration {
	result := &api.AgentServiceRegistration{
		ID:      s.ID,
		Name:    s.Name,
		Address: s.Address,
		Tags:    s.Tags,
	}


	logrus.Debugf("Service to consul is %#v" , result)

	return result
}

