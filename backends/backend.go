package backends

import (
	"errors"
	"github.com/docker/docker/api/types"

	"github.com/docker/docker/client"
)

type ContainerBackend interface {
	Register(c *types.ContainerJSON) error
	Deregister(c *types.ContainerJSON) error
	Containers(  client.APIClient) ([]*types.ContainerJSON, error)
}



type initialize func(address string , options map[string]string) (ContainerBackend, error)

var (
	initializers = make(map[string]initialize)
)

func Register(name string, f initialize) error {
	if _, exists := initializers[name]; exists {
		return errors.New("Service backend has already been registered")
	}
	initializers[name] = f
	return nil
}

func New(name string, address string , options map[string]string) (ContainerBackend, error) {
	if f, exists := initializers[name]; exists {
		return f(address , options)
	}
	return nil, errors.New("Service backend not found")
}
