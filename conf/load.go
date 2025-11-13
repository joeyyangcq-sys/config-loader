package conf

import (
	"errors"
	"fmt"
	"os"

	"config-loader/conf/provider"

	"gopkg.in/yaml.v3"
)

// Load 读取并解析 YAML 配置文件。
func Load(path string, opts any) error {

	if path == "" {
		return errors.New("config path is empty")
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(b, opts); err != nil {
		return fmt.Errorf("parse yaml: %w", err)
	}

	return nil
}

// LoadFromProvider 通过 Provider 打开第一个配置文档并解析为 opts。
func LoadFromProvider(p provider.Provider, opts any) error {
	contents, err := p.Open()
	if err != nil {
		return err
	}
	if len(contents) == 0 {
		return errors.New("no config content from provider")
	}
	for _, c := range contents {
		if err := yaml.Unmarshal([]byte(c.Payload), opts); err != nil {
			return fmt.Errorf("parse yaml: %w", err)
		}
	}

	return nil
}

// LoadOptionsFromProvider 通过 Provider 读取第一个配置文档并解析为 Options。
// 会为缺省端口设置默认值。
func LoadOptionsFromProvider(p provider.Provider) (Options, error) {
	var out Options
	contents, err := p.Open()
	if err != nil {
		return out, err
	}
	if len(contents) == 0 {
		return out, errors.New("no config content from provider")
	}
	for _, c := range contents {
		if err := yaml.Unmarshal([]byte(c.Payload), &out); err != nil {
			return out, fmt.Errorf("parse yaml: %w", err)
		}
	}
	if out.Server.Bind == "" {
		out.Server.Bind = ":8080"
	}
	return out, nil
}
