package registry

import (
	"fmt"
	"net"

	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"
)

type consul struct {
	client *api.Client
}

var Reg Register

var _ Register = (*consul)(nil)

func Init(addr string) (err error) {
	cfg := api.DefaultConfig()
	cfg.Address = addr
	c, err := api.NewClient(cfg)
	if err != nil {
		return err
	}
	Reg = &consul{c}
	zap.L().Info("Init consul success!")
	return
}

// GetOutboundIP 获取本机的出口IP
func GetOutboundIP() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP, nil
}

func (c *consul) RegisterService(serviceName string, ip string, port int, tags []string) error {
	outIP, _ := GetOutboundIP()
	zap.L().Sugar().Infof("OutIP is: %s", outIP)
	check := &api.AgentServiceCheck{
		GRPC:                           fmt.Sprintf("%s:%d", outIP, port), //外部可访问的ip
		Timeout:                        "10s",
		Interval:                       "10s",
		DeregisterCriticalServiceAfter: "20s",
	}
	srv := &api.AgentServiceRegistration{
		ID:      fmt.Sprintf("%s-%s-%d", serviceName, ip, port), //服务ID
		Name:    serviceName,                                    //服务名
		Tags:    tags,                                           //服务标签
		Address: ip,
		Port:    port,
		Check:   check,
	}
	return c.client.Agent().ServiceRegister(srv)
}

func (c *consul) ListService(serviceName string) (map[string]*api.AgentService, error) {
	return c.client.Agent().ServicesWithFilter(fmt.Sprintf("Service==`%s`", serviceName))
}

func (c *consul) Deregister(serviceID string) error {
	return c.client.Agent().ServiceDeregister(serviceID)
}
