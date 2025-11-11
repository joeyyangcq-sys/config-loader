package provider

import (
	"context"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// EtcdProvider 从 etcd 指定 key 读取配置，并订阅变更。
type EtcdProvider struct {
	Endpoints   []string
	Key         string
	Username    string
	Password    string
	DialTimeout time.Duration

	cli *clientv3.Client
}

func NewEtcd(endpoints []string, key string, username, password string) *EtcdProvider {
	return &EtcdProvider{Endpoints: endpoints, Key: key, Username: username, Password: password, DialTimeout: 5 * time.Second}
}

func (p *EtcdProvider) ensureClient() error {
	if p.cli != nil {
		return nil
	}
	cfg := clientv3.Config{Endpoints: p.Endpoints, DialTimeout: p.DialTimeout}
	if p.Username != "" || p.Password != "" {
		cfg.Username = p.Username
		cfg.Password = p.Password
	}
	cli, err := clientv3.New(cfg)
	if err != nil {
		return err
	}
	p.cli = cli
	return nil
}

func (p *EtcdProvider) Open() ([]Content, error) {
	if err := p.ensureClient(); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	resp, err := p.cli.Get(ctx, p.Key)
	if err != nil {
		return nil, err
	}
	if len(resp.Kvs) == 0 {
		return []Content{}, nil
	}
	payload := string(resp.Kvs[0].Value)
	return []Content{{ID: p.Key, Group: "etcd", Payload: payload}}, nil
}

func (p *EtcdProvider) Watch(onChange func() error) error {
	if err := p.ensureClient(); err != nil {
		return err
	}
	go func() {
		wch := p.cli.Watch(context.Background(), p.Key)
		for range wch {
			// 任意事件触发重新加载
			_ = onChange()
		}
	}()
	return nil
}
