# 运行时与前端加固设计

## 背景与目标

项目目前缺少配置接口资源上限、HTTP 服务生命周期控制和可用的结构化日志；前端也缺少 lint/test 基线，并将全部页面及 Monaco 打入单个约 1.42 MB 的主 JavaScript chunk。本轮在不修改数据库结构、不改变普通 KV/Watch 全局语义的前提下，完成资源保护、运行时可观测性和前端工程/加载性能加固。

Token 仍按现状存储于 `localStorage`。用户明确要求本轮跳过 Token 存储调整，因此不引入 Cookie、内存 Token 或认证协议变更。

## 配置接口资源上限

配置导入请求体最大为 10 MiB。Handler 使用 `http.MaxBytesReader` 包装请求体，并区分 `*http.MaxBytesError`，超限时返回参数错误和稳定的提示文本；空 body、格式错误仍由现有导入解析逻辑处理。限制同时适用于 dry-run 和实际导入，避免在权限校验或解析前分配无限内存。

配置列表保持现有响应数组和 `env/prefix` 查询参数，不引入分页兼容性变更。Service 向 etcd 请求最多 501 项：

- 返回 501 项表示超过 500 项硬上限，拒绝响应并提示缩小 prefix。
- 在组装列表时累计 key 与 value 字节数，超过 10 MiB 同样拒绝响应。
- 不静默截断，避免用户误以为列表完整。

新增类型化的列表超限错误，Handler 映射为明确的业务错误码。限制常量集中定义并由单元测试覆盖。

## HTTP 生命周期、超时与日志

使用 Go 标准库 `log/slog`，不增加第三方日志依赖。`log.level` 支持 `debug`、`info`、`warn`、`error`，未知值回退到 `info` 并记录警告；日志输出为 JSON，启动、关闭和请求日志使用统一 logger。

新增 Gin 中间件：

- Request ID：接受非空的 `X-Request-ID`，否则生成 UUID；写回响应头，同时放入 Gin 和 request context。
- Access log：请求结束后记录 request_id、method、path、status、latency_ms 和 client_ip，不记录 query/body/token。
- Recovery：捕获 panic，记录结构化错误和 request_id，返回 500，替代 `gin.Default()` 的非结构化 logger/recovery。
- 普通响应写超时：在进入 Gin 前使用 `http.ResponseController` 设置 30 秒写 deadline；`/api/v1/watch` SSE 路由排除该 deadline，继续使用现有 30 分钟连接上限。

HTTP Server 设置 Header 读取 5 秒、请求读取 15 秒、空闲连接 60 秒。全局 `WriteTimeout` 保持为 0，由上述路由感知的写 deadline 处理，避免破坏 SSE。

主程序使用 `signal.NotifyContext` 监听 `SIGINT/SIGTERM`。收到信号后给 `http.Server.Shutdown` 最多 15 秒；停止接收新请求后关闭 etcd 与底层 SQL 连接。`http.ErrServerClosed` 视为正常退出，其他监听/关闭错误使用结构化日志报告并返回非零状态。

配置增加 HTTP timeout 与 shutdown timeout 字段及合理默认值，可通过环境变量覆盖；不增加或修改数据库表/字段。

## 前端质量基线

引入 ESLint 9 flat config、TypeScript ESLint、React Hooks/Refresh 规则，新增：

- `npm run lint`
- `npm run typecheck`
- `npm test`（Vitest，单次执行）
- `npm run test:watch`

首批 Vitest 测试覆盖不依赖 DOM 的菜单权限与默认路由逻辑，建立可持续扩展的测试入口。lint 配置以现有代码可通过为基线，不通过关闭核心正确性规则来掩盖问题。

## 路由与 Monaco 延迟加载

`App.tsx` 使用 `React.lazy` 动态导入 Login、MainLayout 和所有业务页面，路由区域用统一 `Suspense` loading fallback 包裹。这样首次登录页或默认页面只加载当前路径所需代码。

`MonacoEditor` 保留现有对外 Props，但将 `@monaco-editor/react` 放入内部动态 import；只有编辑器实际渲染时才下载 Monaco。配置、KV、Gateway 和 gRPC 页面不需要改调用语义。

Vite 对 React、Ant Design 和 Monaco 依赖做稳定 vendor 分组。验收关注：生产构建不再产生单个约 1.42 MB 的主入口 chunk；Monaco 独立为按需 chunk。Monaco 自身 chunk 可能仍触发体积提示，但不会阻塞普通页面首屏。

## 错误处理与兼容性

- 导入/列表超限使用类型化错误与稳定业务码，不返回 Go 内部实现细节。
- 配置列表未超限时响应结构不变，前端无需分页适配。
- Authorization Header、JWT、Token 存储和 SSE 请求方式本轮不变。
- Request ID 会新增响应头和结构化日志字段，不改变业务响应 JSON。
- HTTP timeout 默认值写入配置；缺少新配置项时由代码默认值补齐，旧配置文件可继续启动。

## 测试与验收

后端：

- 超过 10 MiB 的导入被拒绝，边界内请求继续进入导入服务。
- 列表超过 500 项或 10 MiB 时返回类型化错误，未超限时保持完整数组。
- log level 解析、request ID 透传/生成、访问日志字段和 panic recovery 有测试。
- HTTP timeout 默认值和 SSE 写 deadline 排除逻辑有测试。
- 优雅关闭通过可取消 context 与测试 server 验证。
- `go test -count=1 ./...`、`go test -race -count=1 ./...`、`go vet ./...` 通过。

前端：

- `npm run lint`、`npm run typecheck`、`npm test` 通过。
- `npm run build` 通过并生成路由/Monaco 分离 chunk，主入口不再是约 1.42 MB 单文件。
- Helm lint 继续通过。

## 非目标

- 不修改 Token 存储或认证协议。
- 不引入 OpenTelemetry、链路追踪、metrics 或外部日志平台。
- 不实现配置列表游标分页；本轮采用显式硬上限和拒绝策略。
- 不修改数据库表、字段或迁移逻辑。
