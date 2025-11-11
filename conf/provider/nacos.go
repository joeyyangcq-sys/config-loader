package provider

import (
	"strconv"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// NacosProvider 从 Nacos Config 服务读取配置，并订阅变更。
type NacosProvider struct {
	ServerAddrs []string // host:port
	NamespaceID string
	Group       string
	DataID      string

	timeoutMs uint64
	cli       config_client.IConfigClient
}

func NewNacos(serverAddrs []string, namespaceID, group, dataID string) *NacosProvider {
	return &NacosProvider{ServerAddrs: serverAddrs, NamespaceID: namespaceID, Group: group, DataID: dataID, timeoutMs: 3000}
}

func (p *NacosProvider) ensureClient() error {
	if p.cli != nil {
		return nil
	}
	var sc []constant.ServerConfig
	for _, addr := range p.ServerAddrs {
		host, port := splitHostPort(addr)
		sc = append(sc, *constant.NewServerConfig(host, port))
	}
	cc := constant.ClientConfig{
		NamespaceId:         p.NamespaceID,
		TimeoutMs:           p.timeoutMs,
		NotLoadCacheAtStart: true,
	}
	c, err := clients.NewConfigClient(vo.NacosClientParam{ClientConfig: &cc, ServerConfigs: sc})
	if err != nil {
		return err
	}
	p.cli = c
	return nil
}

func (p *NacosProvider) Open() ([]Content, error) {
	if err := p.ensureClient(); err != nil {
		return nil, err
	}
	content, err := p.cli.GetConfig(vo.ConfigParam{DataId: p.DataID, Group: p.Group})
	if err != nil {
		return nil, err
	}
	return []Content{{ID: p.DataID, Group: p.Group, Payload: content}}, nil
}

func (p *NacosProvider) Watch(onChange func() error) error {
	if err := p.ensureClient(); err != nil {
		return err
	}
	// 使用 ListenConfig 订阅变更
	err := p.cli.ListenConfig(vo.ConfigParam{
		DataId: p.DataID,
		Group:  p.Group,
		OnChange: func(namespace, group, dataId, data string) {
			_ = onChange()
		},
	})
	// SDK 内部维护长连接，不需要主动循环
	return err
}

// splitHostPort parses "host:port" into host string and port uint64.
func splitHostPort(addr string) (string, uint64) {
	s := strings.TrimSpace(addr)
	if s == "" {
		return "", 0
	}
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return s, 0
	}
	p, _ := strconv.ParseUint(parts[1], 10, 64)
	return parts[0], p
}
