package main

import (
	"context"
	"flag"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	conf "config-loader/conf"
	provider "config-loader/conf/provider"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
)

var welcome atomic.Value // string

// main 负责解析参数、选择 Provider，并启动 HTTP 服务与监听。
// 配置解析逻辑由 conf 包提供；provider 包仅负责配置来源接口。
func main() {
	source := flag.String("source", "file", "config source: file|etcd|nacos")
	cfgPath := flag.String("config", "./config.yaml", "config file path (for file source)")
	etcdEndpoints := flag.String("etcd-endpoints", "", "comma-separated etcd endpoints (for etcd source)")
	etcdKey := flag.String("etcd-key", "", "etcd key holding YAML config (for etcd source)")
	etcdUser := flag.String("etcd-user", "", "etcd username (optional)")
	etcdPass := flag.String("etcd-pass", "", "etcd password (optional)")
	nacosServers := flag.String("nacos-servers", "", "comma-separated nacos server addrs host:port (for nacos source)")
	nacosNS := flag.String("nacos-namespace", "", "nacos namespace id (optional)")
	nacosGroup := flag.String("nacos-group", "DEFAULT_GROUP", "nacos group")
	nacosDataID := flag.String("nacos-dataid", "", "nacos dataId holding YAML config")
	flag.Parse()

	// 初始化 Provider
	var p provider.Provider
	switch *source {
	case "file":
		p = provider.NewFile(*cfgPath)
	case "etcd":
		eps := strings.Split(strings.TrimSpace(*etcdEndpoints), ",")
		p = provider.NewEtcd(nonEmpty(eps), *etcdKey, *etcdUser, *etcdPass)
	case "nacos":
		eps := strings.Split(strings.TrimSpace(*nacosServers), ",")
		p = provider.NewNacos(nonEmpty(eps), *nacosNS, *nacosGroup, *nacosDataID)
	default:
		slog.Error("unknown source", "source", *source)
		return
	}

	// 加载配置
	opts, err := conf.LoadOptionsFromProvider(p)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		return
	}
	var optsVal atomic.Value
	optsVal.Store(opts)
	// 初始化欢迎（用于承载完整配置的 JSON 字符串），防止首次请求取值为 nil
	if jsonStr, err := sonic.MarshalString(opts); err == nil {
		welcome.Store(jsonStr)
	} else {
		welcome.Store("{}")
	}

	h := server.New(
		server.WithHostPorts(opts.Server.Bind),
		server.WithDisableDefaultDate(true),
		server.WithDisablePrintRoute(true),
		server.WithExitWaitTime(1*time.Second),
	)

	// 监听来源变更，动态刷新 opts
	if err := p.Watch(func() error {
		newOpts, err := conf.LoadOptionsFromProvider(p)
		if err != nil {
			slog.Error("reload config failed", "error", err)
			return nil
		}

		optsVal.Store(newOpts)
		if newOptsStr, err := sonic.MarshalString(newOpts); err == nil {
			welcome.Store(newOptsStr)
		}
		return nil
	}); err != nil {
		slog.Error("start config watch failed", "error", err)
	}

	h.GET("/", func(ctx context.Context, c *app.RequestContext) {
		v := welcome.Load()
		if v != nil {
			c.JSON(200, v.(string))
			return
		}
		// 回退：直接返回当前配置对象
		cur := optsVal.Load().(conf.Options)
		c.JSON(200, cur)
	})

	h.GET("/health", func(ctx context.Context, c *app.RequestContext) {
		c.JSON(200, map[string]string{"status": "ok"})
	})

	h.Spin()
}

// nonEmpty 过滤空字符串元素
func nonEmpty(items []string) []string {
	var out []string
	for _, it := range items {
		s := strings.TrimSpace(it)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}
