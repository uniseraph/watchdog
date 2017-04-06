package consul

import (
	"github.com/docker/docker/api/types"
	"strings"
	"github.com/Sirupsen/logrus"
)

type Service struct {
	ID      string
	Name    string
	Address string
	TTL     int
	Tags    []string
}


func  (b *backend)ContainerToService(cJSON *types.ContainerJSON) *Service {


	name, tags := b.getServiceNameAndTags(cJSON)

	if name == "" {
		logrus.Infof("consul  backend : container  %s  name is empty , ignore" , cJSON.ID[0:6])
		return nil
	}

	var address string
	for name, network := range cJSON.NetworkSettings.Networks {
		// 对于增强型的bridge，bridge ip是外部可见的，也需要注册consul
		if  strings.HasPrefix(name, "container:") {
			continue
		}
		if network.IPAddress != "" {
			address = network.IPAddress
		}
	}
	if address == "" {
		logrus.Debug("consul backend : container %s  address is empty , ignore" , cJSON.ID[0:6])
		return nil
	}

	return &Service{
		ID:      cJSON.ID,
		Name:    name,
		Address: address,
		Tags:    tags,
	}
}
type GetRegistratorService func(*types.ContainerJSON) (string, []string)

func getContainerName(s string) string {
	switch i := strings.LastIndex(s, "/"); i {
	case -1:
		return s
	default:
		return s[i+1:]
	}
}

func getComposeServiceNameAndTags(cJSON *types.ContainerJSON) (string, []string) {
	const (
		project = "com.docker.compose.project"
		service = "com.docker.compose.service"
	)

	namespace := cJSON.Config.Labels[project]
	if namespace == "" {
		return "", nil
	}

	tags := []string{
		cJSON.Config.Labels[service],
	}

	containerName := getContainerName(cJSON.Name)
	tags = append(tags, containerName)

	if strings.HasPrefix(containerName, namespace+"_") {
		tags = append(tags, strings.TrimLeft(containerName, namespace+"_"))
	}

	return namespace, tags
}

func getContainerServiceNameAndTags(cJSON *types.ContainerJSON) (string, []string) {
	return getContainerName(cJSON.Name), []string{}
}
