package provider

// Content 表示一个配置单元（例如一个 YAML 文档）。
type Content struct {
	ID      string
	Group   string
	Payload string
}

// Provider 是一个最小的配置源接口，支持打开与监听。
type Provider interface {
	Open() ([]Content, error)
	Watch(onChange func() error) error
}
