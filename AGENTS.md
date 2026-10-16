# CLIProxyAPI 工程约定

Go 代理提供 OpenAI/Gemini/Claude/Codex 兼容接口。工具链版本以 `go.mod` 为准，配置模板为 `config.example.yaml`；工作目录 `.env` 会自动加载，认证材料默认在 `auths/`。

- `internal/thinking/` 保留“后缀/请求解析 → canonical ThinkingConfig → 中央验证 → ProviderApplier”边界；suffix 优先于 body。
- Provider 执行在 `internal/runtime/executor/`，协议翻译在 `internal/translator/`，嵌入式 SDK 在 `sdk/cliproxy/`。executor 目录仅放 executor 和对应测试，辅助文件放 `helps/`。
- 仅修改 translator 的任务保留仓库权限检查：使用 `gh repo view --json viewerPermission -q .viewerPermission` 确认 WRITE/MAINTAIN/ADMIN；无权限时准备含目标、理由和代码的 Issue 草稿，报告阻塞。未授权不自动发布 Issue。
- 新代码注释及普通 Markdown 使用英文；用户可见字符串沿用所在区域语言。修改既有非英文注释时可翻译所触及内容，不扩成全文件清理。
- 不用 `log.Fatal`/`log.Fatalf` 或在 HTTP handler 中 panic；使用 logrus，日志不含凭据。defer 中处理关闭错误，避免覆盖先前错误。
- 凭据取得后不为 upstream 通信增加超时；保留现有例外：Codex WebSocket liveness、wsrelay session、management APICall，以及 `cmd/fetch_antigravity_models`。
- Go 修改只格式化所改文件；按影响运行 `go test ./path/to/pkg` / `go test -run TestName ./path/to/pkg`。服务集成或入口变化用 `go build -o <临时目录>/cli-proxy-api ./cmd/server`；全面验证可用 `go test ./...`。文档修改不要求全仓格式化或构建。
