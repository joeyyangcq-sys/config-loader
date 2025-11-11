package conf

import (
    "errors"
    "fmt"
    "os"

    provider "config-loader/conf/provider"
    "gopkg.in/yaml.v3"
)

// Options 定义最小配置结构：欢迎信息与服务器绑定地址。
type Options struct {
    Welcome WelcomeOptions `yaml:"welcome"`
    Server  ServerOptions  `yaml:"server"`
}

type WelcomeOptions struct {
    Message string `yaml:"message"`
}

type ServerOptions struct {
    Bind string `yaml:"bind"`
}


// Load 读取并解析 YAML 配置文件。
func Load(path string) (Options, error) {
    var opts Options

	if path == "" {
		return opts, errors.New("config path is empty")
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return opts, fmt.Errorf("read config: %w", err)
	}

    if err := yaml.Unmarshal(b, &opts); err != nil {
        return opts, fmt.Errorf("parse yaml: %w", err)
    }

    // 默认值
    if opts.Server.Bind == "" {
        opts.Server.Bind = ":8080"
    }

	// 基础校验
	if err := Validate(opts); err != nil {
		return opts, err
	}

	return opts, nil
}

// Validate 对关键字段进行最小校验。
func Validate(o Options) error {
    if o.Server.Bind == "" {
        return errors.New("server.bind can't be empty")
    }
    return nil
}

// LoadFromProvider 通过 Provider 打开第一个配置文档并解析为 Options。
func LoadFromProvider(p provider.Provider) (Options, error) {
    var opts Options
    contents, err := p.Open()
    if err != nil {
        return opts, err
    }
    if len(contents) == 0 {
        return opts, errors.New("no config content from provider")
    }
    if err := yaml.Unmarshal([]byte(contents[0].Payload), &opts); err != nil {
        return opts, fmt.Errorf("parse yaml: %w", err)
    }
    if opts.Server.Bind == "" {
        opts.Server.Bind = ":8080"
    }
    if err := Validate(opts); err != nil {
        return opts, err
    }
    return opts, nil
}
