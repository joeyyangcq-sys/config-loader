package conf

// Options 表示应用的核心配置结构。
// 目前仅包含欢迎语与服务绑定端口，可按需扩展。
type Options struct {
	Welcome struct {
		Title    string   `yaml:"title"`
		Messages []string `yaml:"messages"`
		Tail     string   `yaml:"tail"`
	} `yaml:"welcome"`
	Server struct {
		Bind string `yaml:"bind"`
	} `yaml:"server"`
}
