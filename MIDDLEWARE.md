# Middleware 接入说明

依据 <https://go-kratos.dev/zh-cn/docs/component/middleware/overview/> 的全部中间件文档，
结合 `AGENTS.md` 的分层契约，把文档里出现的中间件**全部**接入本模板。

## 1. 文档清单

| 文档页 | 主题 |
| --- | --- |
| `/zh-cn/docs/component/middleware/overview/` | 概览、执行顺序、Chain、Use、selector |
| `/zh-cn/docs/component/middleware/auth/` | JWT 认证（contrib） |
| `/zh-cn/docs/component/middleware/circuitbreaker/` | 熔断器 |
| `/zh-cn/docs/component/middleware/logging/` | 日志 |
| `/zh-cn/docs/component/middleware/metrics/` | 指标 |
| `/zh-cn/docs/component/middleware/ratelimit/` | 限流 |
| `/zh-cn/docs/component/middleware/recovery/` | panic 恢复 |
| `/zh-cn/docs/component/middleware/tracing/` | 链路追踪 |
| `/zh-cn/docs/component/middleware/validate/` | 校验 |

## 2. 已接入的全部中间件

核心模块 `github.com/go-kratos/kratos/v3/middleware/...`：

| 包 | 位置 | 用途 | 本项目落点 |
| --- | --- | --- | --- |
| `metadata` | 两端 | 传递 `x-md-*` 请求元数据 | Server / Stream / Client 链首 |
| `logging` | 两端 | 结构化记录 operation、耗时、状态、错误 | Server / Stream / Client |
| `recovery` | 服务端 | panic 转 Kratos 错误 | Server / Stream |
| `validate` | 服务端 | 调用请求校验（`Validate()` + 自定义 ValidatorFunc） | Server / Stream 最内层 |
| `ratelimit` | 服务端 | 准入控制 | Server（流式跳过） |
| `selector` | 两端 | 只对选定 operation 应用中间件 | JWT 白名单、`Use()` 路由 |
| `circuitbreaker` | 客户端 | 依赖持续失败时拒绝调用 | Client |
| `middleware.Chain` | 两端 | 组合中间件 | `Middlewares.ClientMiddleware()` |

Contrib 模块：

| 包 | 模块 | 用途 | 本项目落点 |
| --- | --- | --- | --- |
| `jwt` | `github.com/go-kratos/kratos/contrib/middleware/jwt/v3` | Bearer Token 校验 / 签发 | Server / Stream（selector 白名单）、Client（可选） |
| `tracing` | `github.com/go-kratos/kratos/contrib/otel/v3/tracing` | W3C 传播、span | Server / Stream / Client |
| `metrics` | `github.com/go-kratos/kratos/contrib/otel/v3/metrics` | 请求计数与耗时直方图 | Server / Client |

`main.go` 原本已使用 `tracing.TraceAttrs` 作为日志 extractor，现在 tracing/metrics
middleware 与它共用同一套 OTel 上下文。

## 3. Clean Arch 分层映射

严格遵循 `AGENTS.md`：**依赖向内**，`server` 只组装、不含业务规则，`cmd` 是唯一装配点。

```text
configs/config.yaml ──► conf.Middleware (DTO，internal/conf)
                              │
                              ▼
  ┌──────────────── internal/data ────────────────┐
  │ 实现 biz 端口：                                 │
  │   concurrencyLimiter  → biz.RateLimiter        │
  │   sreBreakerFactory   → biz.CircuitBreaker*   │
  │   configAuthKeyProvider → biz.AuthKeyProvider  │
  └───────────────────────────────────────────────┘
                              ▲ 实现
                              │ 声明（纯 Go，零框架依赖）
  ┌──────────────── internal/biz ────────────────┐
  │ RateLimiter / CircuitBreaker /                 │
  │ CircuitBreakerFactory / AuthKeyProvider        │
  └───────────────────────────────────────────────┘
                              ▲ 依赖接口
                              │
  ┌──────────────── internal/server ─────────────┐
  │ NewMiddlewares：组装 3 条链 + Use 路由          │
  │ http.go / grpc.go：只应用链 + 注册服务          │
  └───────────────────────────────────────────────┘
                              ▲ Wire 注入
                              │
                     cmd/server (wire.go / wire_gen.go)
```

要点：

- `biz` 只声明端口，不 import Kratos、proto、HTTP/gRPC；与 `TodoRepo` 的反转缝保持一致。
- `data` 是唯一读取配置值/凭据的地方；签名密钥来自 `${JWT_SIGNING_KEY:}`，仓库不含真实密钥。
- `server` 只做 `limiterAdapter` / `breakerAdapter` 这类纯接口适配，不含策略判断。
- `service` / `api` / `data/ent` 均未改动，DTO↔DO↔PO 三层不受影响。

## 4. 链的顺序与理由

**Server（HTTP 一元 + gRPC 一元）**

```
metadata → tracing → metrics → logging → recovery → ratelimit → jwt(selector) → validate → handler
```

- `metadata` / `tracing` 先执行，后续链才能拿到元数据与 span 上下文（overview 页示例顺序）。
- `logging` 在 `recovery` 外层，recover 后的 panic 会以 error 出现在完成日志里。
- `ratelimit` 靠近 handler，被拒绝的请求不会消耗昂贵工作；同时仍在 logging 内层，拒绝可被记录。
- `jwt` 用 `selector.Server(...).Match(...)` 包一层，只对**非公开** operation 生效；白名单按
  `/todo.v1.TodoService/ListTodos` 这类规范 operation 名匹配，不是 HTTP 路径。
- `validate` 紧贴 handler（AIP `fieldbehavior.ValidateRequiredFields`）。

**Stream（gRPC 流式）**

```
metadata → tracing → logging → recovery → jwt(selector) → validate → handler
```

长连接不按「每次请求」限流，也跳过 metrics；通过 `grpc.StreamMiddleware` 单独安装。

**Client（出站）**

```
metadata → tracing → metrics → logging → circuitbreaker → jwt.Client(可选) → handler
```

由 `Middlewares.ClientMiddleware()` 用 `middleware.Chain` 组合，供未来的
`grpc.NewClient(..., grpc.WithMiddleware(mw.ClientMiddleware()))` 使用。
注意：`circuitbreaker.Client` 会直接读取 client transport context 的 `Operation()`，
**必须**安装在 HTTP/gRPC 客户端上；装到没有 client transport context 的 handler 会得到
nil 指针（文档明确列为不支持用法），`middleware_test.go` 用 client context 覆盖了这一点。

**Use 路由中间件**

`http.Server.Use` / `grpc.Server.Use` 安装自定义 `audit` 中间件到
`/todo.v1.TodoService/*`；它演示 overview 页「编写中间件」范式：只调用一次 `next`，
用 `log.ContextWithAttrs` 注入 request 级属性。

## 5. 配置

`internal/conf/conf.proto` 新增 `Bootstrap.middleware`，`configs/config.yaml` 的 `middleware:`
段逐项开关。全部开关默认关闭，模板配置按需打开：

| 字段 | 默认 | 说明 |
| --- | --- | --- |
| `logging` / `recovery` / `validate` / `metadata` | `true` | 基础链 |
| `ratelimit` / `metrics` / `tracing` / `circuit_breaker` | `true` | 可观测性与韧性 |
| `jwt` | **`false`** | 需先提供签名密钥 |
| `jwt_client` | `false` | 出站签发 token，默认关闭 |
| `propagated_prefix` | `x-md-` | metadata 传播前缀 |
| `public_operations` | `ListTodos` | JWT 白名单（规范 operation 名） |
| `jwt_signing_key` | `${JWT_SIGNING_KEY:}` | 从环境注入，不提交真实密钥 |
| `tracer_name` / `meter_name` | `kratos-layout` | OTel instrument 名称 |
| `rate_limit` | `64` | 最大并发请求数 |
| `circuit_breaker_threshold` | `5` | 连续失败次数阈值 |

启用 JWT：

```bash
export JWT_SIGNING_KEY='<由部署环境提供>'
# 把 configs/config.yaml 里 middleware.jwt 改为 true
```

指标/追踪只使用全局 OTel provider；provider、exporter、sampling、shutdown 属于应用职责，
Kratos middleware 不会替应用创建或关闭它们（见 tracing/metrics 文档页）。

## 6. 生成与验证

改动了 `conf.proto` 与 Wire 签名，已按 `AGENTS.md` 重新生成（未手工编辑生成文件）：

```bash
buf generate --template buf.gen.config.yaml   # conf.pb.go
buf generate --template buf.gen.yaml          # api/* + openapi.yaml（无变化）
go generate ./...                             # wire_gen.go + Ent
go mod tidy
go test ./...
go build -o ./bin/ ./...
```

新增测试：

- `internal/data/middleware_test.go`：限流器准入/释放幂等、熔断开合与 operation 隔离、密钥返回值防篡改。
- `internal/server/middleware_test.go`：全开/全关链长度、`Chain` 组合、`audit` 单次 next、
  两种适配器、selector + JWT 白名单放行与拦截。

验证结果：`go test ./...` 全部通过，`go build` 成功。

## 7. 依赖变化

`go.mod` 新增：

```
github.com/go-kratos/kratos/contrib/middleware/jwt/v3 v3.0.0-20261026152050-dd0fb7df1e32
github.com/golang-jwt/jwt/v5 v5.3.1
go.opentelemetry.io/otel v1.43.0   # 由 indirect 转为直接依赖
```

> 工具链提示：本机默认 Go 1.27.3 与 `go.mod` 的 `go 1.25.7` 不一致，会让 Ent 生成器失败。
> 请使用 Go 1.25.7（或 `GOTOOLCHAIN` 配合正确的 GOROOT），详见 `CODEGEN_VERIFICATION.md`。
