package provider

import (
    "errors"
    "strings"
    "sync/atomic"

    "gopkg.in/yaml.v3"
)

// Generic 是一个通用配置结构，能够承载任意 YAML 文档经解析后的层级结构。
type Generic struct {
	ID    string
	Group string
	Doc   map[string]any
}

// Manager 负责从 Provider 加载/监听配置，并以原子方式更新当前配置。
type Manager struct {
    provider Provider
    current  atomic.Value // Generic
    onUpdate func(Generic)
}

func NewManager(p Provider) *Manager {
    m := &Manager{provider: p}
    return m
}

// Load 初始化拉取配置。
func (m *Manager) Load() error {
	contents, err := m.provider.Open()
	if err != nil {
		return err
	}
	if len(contents) == 0 {
		return errors.New("no config content from provider")
	}
	// 简化：只使用第一个文档，实际可合并多个。
	c := contents[0]
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(c.Payload), &doc); err != nil {
		return err
	}
	g := Generic{ID: c.ID, Group: c.Group, Doc: doc}
	m.current.Store(g)
	if m.onUpdate != nil {
		m.onUpdate(g)
	}
	return nil
}

// Watch 开启监听，变更时重新打开并更新。
func (m *Manager) Watch() error {
	return m.provider.Watch(func() error {
		// 变更后重新拉取并更新。
		return m.Load()
	})
}

// SetOnUpdate 设置应用在配置变更时的回调。
func (m *Manager) SetOnUpdate(fn func(Generic)) { m.onUpdate = fn }

// Current 返回当前通用配置快照。
func (m *Manager) Current() Generic {
	v := m.current.Load()
	if v == nil {
		return Generic{Doc: map[string]any{}}
	}
	return v.(Generic)
}

// Lookup 使用点号路径在当前文档中查找值，例如 "welcome.message"。
func (m *Manager) Lookup(path string) (any, bool) {
	cur := m.Current()
	parts := strings.Split(path, ".")
	var node any = cur.Doc
	for _, p := range parts {
		mm, ok := node.(map[string]any)
		if !ok {
			return nil, false
		}
		node, ok = mm[p]
		if !ok {
			return nil, false
		}
	}
	return node, true
}
