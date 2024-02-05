package registry

import "github.com/hashicorp/consul/api"

type Register interface {
	RegisterService(serviceName string, ip string, port int, tags []string) error

	ListService(serviceName string) (map[string]*api.AgentService, error)

	Deregister(serviceID string) error
}
