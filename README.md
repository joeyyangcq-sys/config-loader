# config-loader

一个支持动态解析与热更新的 YAML 配置加载器示例。

## 特性
- 读取 `config.yaml` 并解析为结构体（`conf.Options`）
- 监控文件变更（基于 `fsnotify`），自动重新加载配置
- 简化示例 HTTP 服务（CloudWeGo Hertz）读取最新配置并返回欢迎语

## 快速开始
1. 安装 Go 1.21+（推荐安装最新稳定版）
2. 在项目根目录准备配置文件 `config.yaml`：

```yaml
welcome:
  message: "Hello from dynamic config"
server:
  bind: ":8080"
```

3. 运行（默认使用文件来源 `file`）：

```bash
go run . -source file -config ./config.yaml
```

4. 访问：
- `GET http://localhost:8080/` 显示欢迎语与当前绑定地址
- `GET http://localhost:8080/health` 健康检查

5. 热更新：
- 编辑 `config.yaml`（例如修改 `welcome.message` 或 `server.bind`）
- 程序会自动重新加载配置；若仅欢迎语改变，立即生效
- 若 `server.bind` 改变，日志会提示需重启以应用端口变更

## 代码结构
- `conf/load.go`：配置结构定义与解析/校验
- `conf/provider/`：文件 Provider 与通用 Manager（可选的通用解析路径）
- `main.go`：示例 HTTP 服务、文件监听与动态刷新逻辑

## 备注
- 若在本地无法运行 `go` 命令，请先安装 Go 并确保环境变量正确配置。

### 在 macOS 安装 Go
- 使用 Homebrew（推荐）：
  - 确认已安装 Homebrew：`brew -v`
  - 安装 Go：`brew install go`
  - 验证：`go version`
  - 若 `go` 仍未找到，确保 `PATH` 包含 Homebrew 的 `bin`：`echo 'export PATH="$(brew --prefix)/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc`
- 使用官方安装包：
  - 前往 https://go.dev/dl/ 下载适合架构的安装包（Apple Silicon 选 `darwin-arm64.pkg`，Intel 选 `darwin-amd64.pkg`）
  - 安装后将 Go 加入 PATH：`echo 'export PATH="/usr/local/go/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc`
  - 验证：`go version`

安装完成后，在项目根目录执行：

```bash
go mod tidy
go run . -source file -config ./config.yaml
```

## 多来源配置（可拓展）

通过 Provider 抽象可以从不同来源加载配置并订阅更新：

- 文件（默认）：
  ```bash
  go run . -source file -config ./config.yaml
  ```
- etcd：
  先将 YAML 文档写入某个 key（例如 `/app/config`），启动：
  ```bash
  go run . -source etcd \
    -etcd-endpoints 127.0.0.1:2379 \
    -etcd-key /app/config \
    -etcd-user "" -etcd-pass ""
  ```
  程序会订阅该 key 的变更并自动重新加载。
- Nacos：
  在 Nacos Config 创建配置（`dataId` + `group`），启动：
  ```bash
  go run . -source nacos \
    -nacos-servers 127.0.0.1:8848 \
    -nacos-namespace "" \
    -nacos-group DEFAULT_GROUP \
    -nacos-dataid app_config_yaml
  ```
  程序通过 `ListenConfig` 订阅更新并自动重新加载。

### 自定义来源
- 参考 `conf/provider/provider.go` 的 `Provider` 接口，实现 `Open/Watch` 即可。
- 在主程序中创建你的 Provider，并使用 `conf.LoadFromProvider` 解析为结构体。
