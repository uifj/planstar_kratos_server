# Kratos 服务模板 · 源码学习笔记

> 结合 `https://github.com/go-kratos/kratos` 最新代码
>
> - 学习日期：2026-11-05
> - 本地仓库：`C:/Users/21263/WorkBuddy/Worktrees/planstar_kratos_server/main-2eae8665`
> - 本地模块：`github.com/go-kratos/kratos-layout`，`go 1.25.7`
> - 上游基准：`go-kratos/kratos` **main 分支快照**（2026-11-05 抓取），最新正式版 **v3.0.0**（2026-06-26 发布），main 领先 v3.0.0 **7 个提交**；同时抓取 `go-kratos/kratos-layout` main 作对照
> - 上游源码落地位置（临时）：`C:/Users/21263/AppData/Local/Temp/kratos-upstream/kratos-main` 与 `.../kratos-layout-main`

---

## 0. 结论速览

1. **本地仓库不是自研代码，而是官方 `go-kratos/kratos-layout` 模板的一份完整快照。** 逐文件比对（62 个文件）后发现：除 `CLAUDE.md` 外，所有源码、proto、生成代码、配置、Makefile、Dockerfile、CI 与官方 layout main **字节级一致**。
2. 上游 `CLAUDE.md` 是指向 `AGENTS.md` 的**符号链接**；Windows 检出后变成内容为 `AGENTS.md` 的文本桩。所以 `AGENTS.md` 才是唯一的分层契约源。
3. 项目已完全对齐 **Kratos v3**（`/v3` 导入路径、`log/slog`、`*slog.Logger`、v3 生成器），并使用了 2026 年较新的生态：`ent v0.14.6`、`grpc v1.81.1`、`go.einride.tech/aip v0.86.3`、`go-kratos/aip-go/ents`、`UUIDv7`、SSE / WebSocket 流式 HTTP。
4. **测试全部通过**：`go test -tags=http2legacy ./...` 下 `internal/data` 与 `internal/service` 均 `ok`。但本机 Go 1.27.3 直接编译失败（`grpc v1.81.1` 与 `x/net v0.54.0` 在 Go 1.27 的构建标签冲突），应使用 go.mod 声明的 **Go 1.25.x**。详见 §13。
5. 对照上游 v3 迁移指南与 main 分支源码，发现 **13 处可改进/不一致点**（README 过时、CI Go 版本、JSON codec 未显式注册、生成器版本头、流式 RPC 语义等），详见 §12。

---

## 1. 学习范围与上游基准

### 1.1 本地文件全图（62 个文件，全部通读）

| 目录 | 文件 | 性质 |
|---|---|---|
| 根 | `AGENTS.md`（分层契约）、`CLAUDE.md`（链接桩）、`README.md`、`LICENSE` | 文档 |
| 根 | `Makefile`、`go.mod`、`go.sum`、`Dockerfile`、`openapi.yaml` | 构建/契约 |
| 根 | `buf.yaml`、`buf.gen.yaml`、`buf.gen.config.yaml`、`buf.lock` | 代码生成 |
| `.github/workflows/go.yml`、`.gitignore`、`.gitattributes` | CI / 仓库配置 |
| `api/todo/v1/` | `todo.proto`、`error_reason.proto`；生成的 `todo.pb.go`、`todo_grpc.pb.go`、`todo_http.pb.go`、`error_reason.pb.go` | **DTO 公共契约** |
| `cmd/server/` | `main.go`、`wire.go`、`wire_gen.go` | 入口 + Wire 装配 |
| `configs/` | `config.yaml` | 运行时配置 |
| `internal/conf/` | `conf.proto`、`conf.pb.go` | 配置结构（proto 定义） |
| `internal/server/` | `server.go`、`grpc.go`、`http.go` | 传输层装配 |
| `internal/service/` | `service.go`、`todo.go`、`todo_test.go`、`README.md` | DTO ↔ DO |
| `internal/biz/` | `biz.go`、`todo.go`、`README.md` | DO / 用例 / Repo 接口 |
| `internal/data/` | `data.go`、`todo.go`、`todo_test.go`、`README.md` | Repo 实现 |
| `internal/data/ent/` | `schema/{todo,mixin}.go` + **24 个 ent 生成文件**（client/migrate/todo_create/query/update/delete/where/tx/runtime/enttest/hook/predicate…） | **PO / 存储** |

> 生成文件（`*.pb.go`、`*_grpc.pb.go`、`*_http.pb.go`、`wire_gen.go`、`internal/data/ent/**` 除 `schema/` 与 `generate.go`）均按 `AGENTS.md` 约定**不可手改**，只能 `make api` / `make config` / `go generate` / `make all` 重新生成。

### 1.2 上游基准事实（已核实）

| 项 | 值 |
|---|---|
| 默认分支 | `main` |
| 最新 release | `v3.0.0`，发布于 2026-06-26 |
| main 相对 v3.0.0 | `ahead_by = 7`，`total_commits = 7`，最后一次代码提交 2026-10-21，最后提交 2026-10-26 |
| 上游根 `go.mod` | `module github.com/go-kratos/kratos/v3`，`go 1.25.0` |
| 上游 `version.go` | `const Release = "v3.0.0"` |
| `cmd/protoc-gen-go-http/version.go` | `release = "v3.0.0"` |
| 仓库规模 | 26k stars，MIT，Go 1.25+ |
| 7 个后续提交 | `fix(config): resolve bool placeholders to bool (#3891)`、`feat(contrib): add oauth2 middleware (#3901)`、`fix(transport): escape path values and fix form encoding (#3902)` + 4 个 CI 依赖升级 |

上游顶层包：`app.go` / `options.go` / `version.go`、`config/`、`log/`、`errors/`、`middleware/`、`transport/{http,grpc}/`、`encoding/`、`registry/`、`selector/`、`metadata/`、`internal/`、`cmd/{kratos,protoc-gen-go-http,protoc-gen-go-errors}/`、`contrib/`、`docs/`、`third_party/`。

`kratos-layout` main 最近提交为 2026-09-01（`chore(deps): bump aip-go/ents to latest`），本地即该状态。

---

## 2. 项目定位与技术栈

这是一个**用于创建新 Kratos 服务的模板仓库**，示例资源 `Todo` 只是演示 API 形状、分层、代码生成与测试，README 明确要求建真实项目时替换掉。

| 能力 | 选型 | 版本/位置 |
|---|---|---|
| 框架 | `github.com/go-kratos/kratos/v3` | v3.0.0 |
| 可观测 | `contrib/otel/v3/tracing`（日志注入 trace_id/span_id） | 伪版本 `v3.0.0-20260515082355-…` |
| 传输 | HTTP(:8000) + gRPC(:9000) | `internal/server/` |
| API | Protobuf + `google.api.http` + `field_behavior` | `api/todo/v1/todo.proto` |
| 列表查询 | AIP `filtering` / `ordering` / `pagination` / `fieldmask` | `go.einride.tech/aip v0.86.3` |
| SQL 翻译 | `github.com/go-kratos/aip-go/ents` | 把 AIP Filter/OrderBy 编译成 ent 谓词 |
| ORM | `entgo.io/ent v0.14.6` | MySQL 运行 / `modernc.org/sqlite` 测试 |
| ID | `uuid.NewV7()`（UUIDv7，时间有序） | `internal/data/ent/schema/mixin.go` |
| DI | `github.com/google/wire v0.6.0` | `cmd/server/wire*.go` |
| 代码生成 | buf + protoc-gen-go / go-grpc / **protoc-gen-go-http/v3** / gnostic-openapi | `buf.gen*.yaml` |
| 运行时 | `go.uber.org/automaxprocs`、Go 1.25.7 | `main.go`、`go.mod` |

---

## 3. 分层架构：DTO → DO → PO

`AGENTS.md` 定义的契约（也是理解全部代码的骨架）：

```
client ──► DTO ──► service ──► DO ──► biz ──► DO ──► data ──► PO ──► storage
                         ▲                  ▲
                         │ declares         │ implements
                         └──── repo IF ─────┘
```

| 层 | 拥有 | 边界语言 | 禁止 | 本地实现 |
|---|---|---|---|---|
| `service` | — | DTO ↔ DO | PO、storage client | `internal/service/todo.go`（`convertTodo` 入，`convertTodoReply` 出） |
| `biz` | **DO** `biz.Todo`、用例、Repo 接口 | DO | DTO / PO / storage | `internal/biz/todo.go` |
| `data` | **PO** `ent.Todo` | DO ↔ PO | DTO | `internal/data/todo.go` + `internal/data/ent/**` |
| `server` | HTTP/gRPC Server | 注册 + 中间件 | 业务/转换 | `internal/server/{http,grpc}.go` |
| `cmd` | 装配 | 全部 | — | `cmd/server/wire.go` → `wire_gen.go` |

**依赖倒置的落点**：`biz.TodoRepo`（`internal/biz/todo.go:49`）声明 5 个方法；`data.NewTodoRepo` 返回**接口类型** `biz.TodoRepo` 而非具体 `todoRepo`（`internal/data/todo.go:36`）；`cmd` 是唯一同时 import 所有层的地方。

三个模型字段对照：

| 概念 | DTO `v1.Todo` | DO `biz.Todo` | PO `ent.Todo` |
|---|---|---|---|
| ID | `string` | `uuid.UUID` | `uuid.UUID`（`field.UUID`，UUIDv7 默认） |
| 时间 | `*timestamppb.Timestamp` | `time.Time` | `time.Time`（`Immutable` / `UpdateDefault`） |
| 状态 | `v1.TodoStatus` 枚举 | `biz.TodoStatus int32` | `int32` + `GoType(biz.TodoStatus(0))` |
| 其余 | `title/content/completed` | 同名 | 同名 + 默认值 |

- `convertTodo` **故意丢弃** `id/created_at/updated_at/status`（服务端只读字段），只把可变字段送进 DO。
- `convertTodoReply` 反向组装，时间戳用 `timestamppb.New`，状态用显式 `switch`（`convertTodoStatus`）避免枚举漂移。
- `toBiz(po)` 把 ent PO 转回 DO；**没有** `newTodo(do)`，因为写入直接在 repo 里用 ent builder 逐字段 `SetXxx`。

---

## 4. 启动链路：从 `main` 到 `app.Run()`

### 4.1 `cmd/server/main.go` 逐段

```go
logger := log.NewLogger(slog.NewTextHandler(os.Stdout, ...), log.WithExtractor(tracing.TraceAttrs))
log.SetDefault(logger)                       // v3: 直接接管 slog.Default()
c := config.New(config.WithSource(
    file.NewSource(flagconf),               // -conf，默认 "../../configs"
    env.NewSource("KRATOS"),                // KRATOS_ 前缀环境变量，后加载覆盖前加载
))
c.Load(); c.Scan(&bc)                       // bc 为 conf.Bootstrap（proto 生成）
app, cleanup, err := wireApp(bc.Server, bc.Data, logger)
app.Run()
```

- **上游对照**：`log/log.go` 只保留 `slog` 门面（`SetDefault = slog.SetDefault`、`Info/Error/…` 直接构造 `slog.Record` 并交给默认 handler）。`contrib/otel/tracing/tracing.go:94` 的 `TraceAttrs(ctx)` 返回 `trace_id` / `span_id` 两个 `slog.Attr`，由 `log.WithExtractor` 注入每条日志。
- **配置占位符**：`configs/config.yaml` 里的 `${DATABASE_SOURCE:root:root@tcp(...)}`、`${REDIS_ADDR:127.0.0.1:6379}` 由上游 `config/options.go` 的 `defaultResolver` / `placeholderRegexp = \${(.*?)}` 解析；`env.NewSource("KRATOS")` 由 `config/env/env.go` 去掉 `KRATOS_` 前缀后作为 key 注入，因此 `KRATOS_DATABASE_SOURCE` 可覆盖文件值。该能力来自 layout 提交 `feat: allow env vars to override config values (#172)`（2026-08-16）。
- `conf.proto` 的 `Bootstrap{Server,Data}` 用 `make config`（`buf.gen.config.yaml` → `protoc-gen-go`）生成 `conf.pb.go`；`google.protobuf.Duration` 字段用 `.AsDuration()` 传给 server option。
- `_ "go.uber.org/automaxprocs"`：按容器 CPU quota 自动设 `GOMAXPROCS`。

### 4.2 Wire 装配（`wire.go` / `wire_gen.go`）

```
wireApp(conf.Server, conf.Data, logger)
  └─ wire.Build(server.ProviderSet, data.ProviderSet, biz.ProviderSet, service.ProviderSet, newApp)
```

生成结果（`wire_gen.go:26-40`）：

```
NewData(*conf.Data) → (*Data, cleanup, error)
NewTodoRepo(*Data) → biz.TodoRepo
NewTodoUsecase(biz.TodoRepo) → *biz.TodoUsecase
NewTodoService(*biz.TodoUsecase) → *service.TodoService
NewGRPCServer(*conf.Server, *TodoService) → *grpc.Server
NewHTTPServer(*conf.Server, *TodoService) → *http.Server
newApp(logger, gs, hs) → *kratos.App
```

四个 `ProviderSet`：`server.NewSet(NewGRPCServer, NewHTTPServer)`、`data.NewSet(NewData, NewTodoRepo)`、`biz.NewSet(NewTodoUsecase)`、`service.NewSet(NewTodoService)`。`wire.go` 带 `//go:build wireinject` 构建标签，只在生成时存在；`wire_gen.go` 带 `!wireinject`，是最终编译用的产物。

### 4.3 `kratos.App` 生命周期（上游 `app.go` / `options.go`）

`newApp` 用 `kratos.ID/Name/Version/Metadata/Logger/Server` 六个 option 构造。上游 `App.Run()` 的实际行为：

1. `buildInstance()`：从各 `transport.Endpointer`（HTTP/gRPC server）收集 endpoint，构造 `registry.ServiceInstance`；
2. `errgroup`：为每个 server 起两个 goroutine —— 一个 `server.Start(octx)`，一个 `<-ctx.Done()` 后 `server.Stop(stopCtx)`（可用 `kratos.StopTimeout` 限制）；
3. 若配置了 `kratos.Registrar`，`Register`（默认超时 10s）；
4. `BeforeStart` / `AfterStart` / `BeforeStop` / `AfterStop` 钩子；
5. `signal.Notify(SIGTERM/SIGQUIT/SIGINT)` → `a.Stop()` → `Deregister` → `cancel()` → 各 server 优雅退出。

> 本地**未使用** `Registrar`、钩子、StopTimeout、`kratos.Context`，这是模板保持精简的选择，也是接服务发现/优雅停机时的扩展点。

### 4.4 Server 装配（`internal/server/`）

- `grpc.go`：`recovery.Recovery()` 中间件 + `Network/Address/Timeout` 条件 option，然后 `v1.RegisterTodoServiceServer(srv, todo)`。
- `http.go`：`recovery.Recovery()` + `validate.Validator(func(req any) error { fieldbehavior.ValidateRequiredFields(msg) })`，然后 `v1.RegisterTodoServiceHTTPServer(srv, todo)`。
- 上游 `middleware/middleware.go`：`type Middleware func(Handler) Handler`，`Chain` 逆序包裹；`middleware/validate/validate.go` 会先调用请求自身的 `Validate()`（protovalidate 生成），再跑自定义 validator，失败统一转 `errors.BadRequest("VALIDATOR", …).WithCause(err)`；`middleware/recovery/recovery.go` 用 `slog` 记录 panic 栈，默认返回 `errors.InternalServer("UNKNOWN", …)`。
- 上游 `transport/http/server.go` / `transport/grpc/server.go`：默认 `:0`、`timeout 1s`、mux 路由、matcher 中间件、`health`/`reflection`/`admin`（gRPC）。

---

## 5. API 契约与代码生成流水线

### 5.1 `api/todo/v1/todo.proto`

7 个 RPC（5 个 unary + 2 个 stream）：

| RPC | HTTP 规则 | 说明 |
|---|---|---|
| `CreateTodo` | `POST /v1/todos/create`，`body: "todo"` | 返回带服务端 id/时间戳的 Todo |
| `ListTodos` | `GET /v1/todos/list` | `page_size/page_token/filter/order_by` |
| `GetTodo` | `GET /v1/todos/{id}` | 路径变量 |
| `WatchTodos` | `GET /v1/todos/watch`，`returns (stream TodoEvent)` | 服务端流 |
| `UpdateTodo` | `PUT /v1/todos/update`，`body: "todo"` | `FieldMask` 局部更新 |
| `DeleteTodo` | `DELETE /v1/todos/{id}`，返回 `google.protobuf.Empty` | 软删除 |
| `SyncTodos` | `POST /v1/todos/sync`，`body: "*"`，双向流 | 双向同步 |

消息：`Todo`、`TodoSet`（含 `next_page_token`）、`TodoStatus` 枚举（UNSPECIFIED/ACTIVE/DELETED）、`TodoEvent`（action/todo/event_time）、`SyncTodoRequest`（action/todo/id/update_mask）。`google.api.field_behavior = REQUIRED` 标注必填字段，由 HTTP 侧 `fieldbehavior.ValidateRequiredFields` 校验。

`error_reason.proto`：`ErrorReason{ TODO_UNSPECIFIED=0, TODO_NOT_FOUND=1, TODO_INVALID_ARGUMENT=2 }`，在 `biz` 中以 `.String()` 作为错误 reason。

### 5.2 生成流水线（`Makefile` + `buf.gen*.yaml`）

```
make init   → go install wire@latest, buf@latest
make api    → buf generate --template buf.gen.yaml         (inputs: api/)
make config → buf generate --template buf.gen.config.yaml  (inputs: internal/)
make generate → go generate ./... (ent 生成 + wire 生成) + go mod tidy
make all    → api + config + generate
make build  → go build -ldflags "-X main.Version=$(git describe)" -o ./bin/ ./...
```

`buf.gen.yaml` 的 4 个插件：

| 插件 | 版本 | 产物 |
|---|---|---|
| `protoc-gen-go` | v1.36.11 | `todo.pb.go`、`error_reason.pb.go` |
| `protoc-gen-go-grpc` | v1.6.2 | `todo_grpc.pb.go`（泛型 `grpc.ServerStreamingClient[T]`） |
| `protoc-gen-go-http/v3` | `v3.0.0-20260526000039-30da04b769dc` | `todo_http.pb.go` |
| `protoc-gen-openapi`（gnostic） | v0.7.1 | `openapi.yaml`（OpenAPI 3.0.3） |

`buf.yaml` 是 v2 格式，模块 `api` + `internal`，依赖 `buf.build/googleapis/googleapis`（锁在 `buf.lock`）。

### 5.3 `protoc-gen-go-http` 模板（上游 `cmd/protoc-gen-go-http/httpTemplate.tpl`）

生成代码的关键形态，与本地 `todo_http.pb.go` 一一对应：

- `const OperationTodoServiceXxx = "/todo.v1.TodoService/Xxx"`：供中间件按 operation 匹配（上游 `matcher` + `Server.Use(selector, …)`）。
- `RegisterTodoServiceHTTPServer`：`r := s.Route("/")` 后逐条 `r.Handle(METHOD, PATH, handler)`；handler 内 `ctx.Bind`/`ctx.BindQuery`/`ctx.BindVars` → `http.SetOperation(ctx, …)` → `ctx.Middleware(chain)` → `ctx.Result(200, reply)`。
- `const _ = http.SupportPackageIsVersion3`（上游 `transport/http/codec.go:21` 定义）编译期版本断言。
- **流式 RPC**：模板为服务端流生成 `TodoService_WatchTodosHTTPServer{ http.ServerStream }`，为双向流生成 `SyncTodosHTTPServer`；服务端流走 `http.NewServerSentEventServerStream(ctx)`（SSE），双向流走 `http.NewWebSocketServerStream(ctx)`（WebSocket，模板中支持 `WithStreamBodyField` 处理命名 body 字段）。客户端侧对应 `ServerSentEvent` / `cc.WebSocket`。
- 上游 `transport/http/stream.go` 定义 `ServerStream` 接口（`grpc.ServerStream` + `Send/Recv/SendAndClose/Close/SetContext/SetReadDeadline/SetWriteDeadline`），并用 `detachStreamContext` 把长连接从每请求 1s 超时中剥离，只保留 context value（trace/metadata）。

> **注意**：生成器模板对流式 RPC **硬编码注册成 `GET`**（`r.Handle("GET", "{{.Path}}", …)`），所以尽管 proto 给 `SyncTodos` 写的是 `post: "/v1/todos/sync" body:"*"`，生成的路由实际是 `GET /v1/todos/sync`（见本地 `todo_http.pb.go:70`）。这是上游生成器的行为，不是本地写错。

### 5.4 生成器版本差异

本地 `todo_http.pb.go` 头部写 `protoc-gen-go-http v2.9.2`，但 `buf.gen.yaml` 调用的是 `/v3` 伪版本。原因：上游 `30da04b`（`break: update kratos modules to v3`，2026-05-26）改了模块路径却尚未把 `version.go` 的 `release` 从 `v2.9.2` 改成 `v3.0.0`（后者在 `668db92` 才做）。**官方 layout main 同样如此**，属于上游遗留痕迹；升级到 `protoc-gen-go-http/v3@v3.0.0` 后重新 `make api` 可消除，并顺带获得 v3.0.0 的 `HttpBody` 响应、ServerStream 读写 deadline 等新特性。

---

## 6. 请求全链路：7 个 RPC 逐一拆解

### 6.1 `CreateTodo`

`service.CreateTodo` → `convertTodo(req.Todo)`（丢弃只读字段）→ `uc.CreateTodo` → `biz.validateTodo`（`strings.TrimSpace(title) != ""`）→ `repo.CreateTodo` → ent builder `SetTitle/SetContent/SetCompleted/SetStatus(Active)` + `Save(ctx)` → `toBiz(po)` → `convertTodoReply`。ID 由 ent schema 的 `Default(newUUID)`（`uuid.Must(uuid.NewV7())`）生成，时间戳由 `TimeMixin` 的 `Default(time.Now)` / `UpdateDefault(time.Now)` 生成。

### 6.2 `GetTodo`

`parseTodoID` 在 **service 边界**用 `uuid.Parse` 校验，非法 → `biz.ErrTodoInvalidArgument`；`biz.GetTodo` 再挡一次 `uuid.Nil`；`repo.FindByID` 用 `Where(todo.IDEQ(id), todo.StatusEQ(Active)).Only(ctx)`，`ent.IsNotFound` → `biz.ErrTodoNotFound`。软删除记录读不到。

### 6.3 `ListTodos`（最复杂的一条链路）

1. `service` 用 `filtering.NewDeclarations` 声明 6 个可过滤字段（`id/title/content` 字符串、`completed` 布尔、`created_at/updated_at` 时间戳）与标准函数；
2. `filtering.ParseFilter(req, decls)`、`pagination.ParsePageToken(req)`、`ordering.ParseOrderBy(req)` 解析并校验；`orderBy.ValidateForPaths("id","title","created_at","updated_at")` 白名单（**`content` 不能排序**）；
3. `page_size <= 0` → 默认 20；
4. 组装 `biz.ListFilter / ListOrderBy / ListLimit / ListOffset` 四个 functional option（`internal/biz/todo.go:57-94`），这样 `biz` 不泄漏存储原语；
5. `uc.ListTodos` 只做转发；`repo.ListTodos` 把 option 折成 `biz.ListOptions`，校验 `Offset >= 0 && Limit > 0`；
6. 关键 SQL 构造（`internal/data/todo.go:66-72`）：

```go
.Where(todo.StatusEQ(biz.TodoStatusActive))
.Where(ents.ApplyFilter(options.Filter))                 // AIP Filter → ent 谓词
.Order(ents.ApplyOrderBy(options.OrderBy), todo.ByID()) // 追加 id 作为全序兜底
.Offset(options.Offset).Limit(options.Limit).All(ctx)
```

`github.com/go-kratos/aip-go/ents` 负责把 AIP 过滤/排序表达式翻译成 ent 查询。`id` 是 UUIDv7（时间有序），既是无 `order_by` 时的默认顺序，也保证**偏移分页在并列值下不重不漏**（`todo_test.go` 的 `TestTodoRepoListDefaultOrderIsStable` / `TestTodoRepoListOrderByTiesBrokenByID` 专门验证）。

7. 回包：`TodoSet{Todos, NextPageToken}`；仅当 `len(todos) >= page_size` 才用 `pageToken.Next(req).String()` 生成下一页 token（`page_size=2` 且剩 1 条时不再给 token，测试覆盖了这点）。

### 6.4 `UpdateTodo`

- service 先校验 `todo.id != ""`、`update_mask != nil && len(paths) > 0`；
- **复用 `s.GetTodo`** 取当前完整 DTO（这一步同时完成存在性与 UUID 校验）；
- `fieldmask.Update(mask, current, req.Todo)` 把请求中列出的字段覆盖到当前对象；未列出的字段保持不变；`*` 单路径替换全部可变字段；
- 再 `convertTodo(current)` → `uc.UpdateTodo`（再次 `validateTodo`）→ `repo.UpdateTodo`：`UpdateOneID(...).Where(status=Active)` 且**只 Set 三字段**，`status` 不可通过 update 改（测试 `TestTodoServiceCRUD` 断言）。
- 代价：一次 update = 一次读 + 一次写；换来了 fieldmask 语义与「只读字段不被客户端污染」。

### 6.5 `DeleteTodo`（软删除）

`repo.DeleteTodo` 是 `Todo.Update().Where(id, status=Active).SetStatus(Deleted).Save(ctx)`，返回 `affected`；为 0 → `biz.ErrTodoNotFound`。行保留可审计，所有读路径 `status=Active` 过滤，所有写路径也带 `status=Active`。`todo_test.go` 的 `TestTodoRepoDeleteIsSoft` 用 `client.Todo.Get` 验证行仍在、状态已翻转，并验证二次删除/更新/读取都返回 NotFound。

### 6.6 `WatchTodos`（服务端流）

service 复用 `ListTodos` 的解析逻辑，取一页快照后对每个 todo 发一个 `action="snapshot"` 的 `TodoEvent`，然后返回 `nil` 关闭流。**注意**：proto 注释写的是「持续推送每次 create/update/delete」，但实现只是「一次性快照」，没有事件订阅/广播——这是模板示例与契约注释的语义差距。HTTP 侧由 SSE（`text/event-stream`）承载。

### 6.7 `SyncTodos`（双向流）

`for { Recv() → switch action(create/update/delete) → 调用同一 service 的 unary 方法 → Send(TodoEvent) }`，`io.EOF` 正常结束，未知 action → `biz.ErrTodoInvalidArgument`。`newTodoEvent` 直接接收已经构造好的 `*v1.Todo` 回复，避免回灌 DO 丢失服务端时间戳（测试 `TestTodoServiceSyncTodos` 专门断言 `created_at/updated_at/event_time` 非零）。HTTP 侧由 WebSocket 承载。

---

## 7. 数据层：Ent 与持久化设计

- `internal/data/data.go`：`Data{db *ent.Client}` 是长生命周期客户端；`NewData` 用 `ent.Open(driver, source)`，按配置开 `Debug()` 与 `Schema.Create()`（auto migrate），返回 `cleanup` 由 Wire 挂到 `app` 生命周期。
- `schema/mixin.go`：`IDMixin`（`field.UUID("id", uuid.Nil).Default(newUUID).Immutable()`，注释说明**有意不写 SchemaType**，让 ent 按方言映射 UUID）、`TimeMixin`（`created_at` Immutable、`updated_at` UpdateDefault、`updated_at` 索引）。
- `schema/todo.go`：`title/content/completed` 带默认值，`status` 用 `GoType(biz.TodoStatus(0))` 把 PO 直接绑定领域类型，`status` 建索引。
- `migrate/schema.go`（生成）：`todos` 表 7 列，主键 `id`，索引 `todo_updated_at`、`todo_status`。
- 运行用 MySQL（`configs/config.yaml`），测试用 `modernc.org/sqlite` 内存库 + `entsql.OpenDB(dialect.SQLite, db)` + `client.Schema.Create`（`todo_test.go:22-36`）。

---

## 8. 横切关注点：错误、中间件、校验（上游对照）

| 本地用法 | 上游最新实现 | 要点 |
|---|---|---|
| `errors.NotFound` / `errors.BadRequest` / `IsNotFound` / `IsBadRequest` | `errors/types.go`、`errors/errors.go` | `*errors.Error` 内嵌 `Status{Code,Reason,Message,Metadata}` + `cause`；`Code/Reason/FromError` 支持 `errors.As` 解包；`GRPCStatus()` 把 HTTP code 转 gRPC code 并附 `errdetails.ErrorInfo`；`WithCause/WithMetadata/Clone` |
| `v1.ErrorReason_*.String()` | `errors/errors.proto` + `cmd/protoc-gen-go-errors` | 上游提供生成器，可从 proto 枚举直接生成类型化错误构造函数；**本地未使用**，而是手写 `biz.ErrTodoXxx` |
| `errors.Is/As/Unwrap/Join` | `errors/wrap.go`（v3.0.0 新增，`feat(errors): add standard library errors wrappers #3844`） | v3 补齐标准库错误链门面；另有 `errors.TooManyRequests`（#3787）等新错误类型 |
| `validate.Validator` | `middleware/validate/validate.go` | 先 `Validate()` 接口（protovalidate），再自定义 validator；失败统一 400 |
| `fieldbehavior.ValidateRequiredFields` | `go.einride.tech/aip/fieldbehavior`（第三方） | AIP-203 必填字段校验 |
| `recovery.Recovery()` | `middleware/recovery/recovery.go` | v3 用 `*slog.Logger`（`WithLogger`），默认 `slog.Default()`；64KB 栈 |
| `log.NewLogger / WithExtractor / SetDefault` | `log/log.go` + `contrib/otel/tracing` | v3 完全转向 `log/slog`，`log.Logger/Helper/Valuer` 已移除 |
| `config.New/WithSource/Scan` | `config/config.go` | `Config` 接口 `Load/Scan/Value/Watch/Close`；`Load` 后启动 watcher goroutine 做热更新；v3 新增泛型 `config.Get[T](c, key)`（#3778） |

---

## 9. 测试策略（三层隔离）

| 层 | 文件 | 做法 |
|---|---|---|
| service | `internal/service/todo_test.go` | `fakeTodoRepo`（内存 map + `order []uuid.UUID` 保序）实现 `biz.TodoRepo`；`fakeServerStream` / `watchTodosStream` / `syncTodosStream` 伪造 gRPC 流接口。覆盖 CRUD、分页 token、filter/order 校验、必填/非法参数、Watch/Sync 事件 |
| data | `internal/data/todo_test.go` | 真 SQLite + 真 ent 查询：`newTestRepo` 每个用例一个内存库；覆盖 CRUD、软删除、分页、默认顺序稳定、并列值用 id 兜底、desc 排序 |
| biz | 无独立测试 | 通过 service 层 fake repo 间接覆盖；`AGENTS.md` 建议 biz 测试应 fake repo |

测试通过记录见 §13。

---

## 10. 本地代码 ↔ 上游 Kratos 最新源码 映射表

| 本地引用 | 上游文件 | 作用 |
|---|---|---|
| `kratos.New/ID/Name/Version/Metadata/Logger/Server` | `app.go`、`options.go` | 应用生命周期、option、`AppInfo` context |
| `kratos.App.Run` | `app.go:83-151` | errgroup 启停、signal、registrar、钩子 |
| `config.New / WithSource / file.NewSource / env.NewSource / Scan` | `config/config.go`、`config/options.go`、`config/reader.go`、`config/file/*`、`config/env/*` | 多源合并、`${K:default}` 解析、watcher 热更新、protojson Scan |
| `log.NewLogger / SetDefault / WithExtractor` | `log/log.go`、`log/handler.go`、`log/builder.go` | slog 门面 |
| `tracing.TraceAttrs` | `contrib/otel/tracing/tracing.go:94` | 日志注入 trace_id/span_id |
| `errors.NotFound / BadRequest / IsNotFound / IsBadRequest` | `errors/types.go`、`errors/errors.go`、`errors/wrap.go` | 统一 HTTP/gRPC 错误 |
| `middleware/recovery.Recovery` | `middleware/recovery/recovery.go` | panic 恢复 |
| `middleware/validate.Validator` | `middleware/validate/validate.go` | 请求校验 |
| `transport/http.NewServer / Middleware / Address / Timeout / Network` | `transport/http/server.go` | HTTP server、mux、`matcher` 中间件 |
| `transport/grpc.NewServer / Middleware / …` | `transport/grpc/server.go` | gRPC server、health/reflection/admin |
| `http.Context.Bind/BindQuery/BindVars/Result/SetOperation` | `transport/http/context.go`、`codec.go` | 请求/响应编解码、operation 注入 |
| `http.SupportPackageIsVersion3` | `transport/http/codec.go:21` | 生成代码版本断言 |
| `http.ServerStream` / SSE / WebSocket | `transport/http/stream.go` | 流式 HTTP 适配 gRPC 流接口 |
| `RegisterTodoServiceHTTPServer` | `cmd/protoc-gen-go-http/httpTemplate.tpl` | 生成路由与 handler |
| `RegisterTodoServiceServer` | `google.golang.org/grpc` 生成 | gRPC 注册 |
| `protoc-gen-openapi` → `openapi.yaml` | `github.com/google/gnostic` | OpenAPI 3.0.3 |
| **未使用但可用** | `registry/`、`selector/`、`metadata/`、`middleware/{logging,metadata,ratelimit,circuitbreaker,selector}/`、`contrib/*`、`encoding/{json,protojson,form,xml,yaml}/`、`cmd/kratos`（`new/proto/run/upgrade/change`） | 服务发现、负载均衡、可观测、编解码、CLI 脚手架 |

上游 `cmd/kratos/main.go` 注册 `new / proto / upgrade / change / run` 五个子命令，官方推荐 `kratos new <svc>` → `go mod tidy` → `kratos run`。

---

## 11. v3 迁移指南逐条核对（`docs/migration/v2-to-v3_zh.md`）

| # | 上游要求 | 本地状态 |
|---|---|---|
| 1 | 导入路径 `/v2` → `/v3` | ✅ 全部 `kratos/v3`，contrib 也是 `contrib/otel/v3` |
| 2 | **显式选择 JSON Codec**（`encoding/json` / `encoding/protojson` / `contrib/encoding/json/v3`） | ⚠️ **未显式导入任何 encoding 包**，见 §12-2 |
| 3 | 日志迁移到 `log/slog`，`kratos.Logger(*slog.Logger)` | ✅ `log.NewLogger(slog.NewTextHandler(...))` |
| 4 | JWT 移到 contrib | ✅ 未使用 JWT |
| 5 | 自定义熔断器改 `WithBreakerFactory` | ✅ 未使用熔断器 |
| 6 | 移除 `transport/http/binding`，用 `http.BuildPath` / `Context.Bind*` | ✅ 生成代码已是新 API |
| 7 | 重新生成代码（`go generate ./...`） | ⚠️ 生成头仍标 `v2.9.2`，见 §12-6 |
| 8 | `go test` / `go vet` / `go mod tidy` 验证 | ✅ 测试通过（需正确 Go 版本），见 §13 |

---

## 12. 发现的问题 / 不一致 / 改进建议

按影响排序：

1. **【构建】Go 版本不匹配导致编译失败。** `go.mod` 要求 `go 1.25.7`，本机为 `go1.27.3`，直接 `go test ./...` 报 `undefined: http2.TrailerPrefix`（`grpc@v1.81.1/internal/transport/handler_server.go:271`）。根因：`x/net@v0.54.0/http2/server.go` 的构建标签是 `//go:build !(go1.27 && !http2legacy)`，Go 1.27 默认排除了定义 `TrailerPrefix` 的 legacy 实现，而 grpc v1.81.1 仍引用它。**建议**：用 `GOTOOLCHAIN=go1.25.7`（或 Go 1.25/1.26）构建；临时可加 `-tags=http2legacy`。CI 也应固定 Go 版本而非 `^1.20`。
2. **【v3 语义】proto 消息的 HTTP JSON 编解码未显式配置。** v3 把 `json` codec 拆成：核心 `encoding/json`（标准库，注册名 `json`）、`encoding/protojson`（注册名 `protojson`）、`contrib/encoding/json/v3`（v2 兼容、注册名 `json`、对 `proto.Message` 走 protojson）。本地只因 `import config` 间接注册了标准库 `json` codec；`transport/http/codec.go:160` 的 `CodecForRequest` 在无匹配时回落 `GetCodec("json")`。结果是 `Todo.created_at`（`Timestamp`）会输出成 `{"seconds":…,"nanos":…}`、`status` 输出数字而非 RFC3339 字符串/枚举名。**建议**：按迁移指南显式 `_ "github.com/go-kratos/kratos/v3/encoding/json"` + `_ "…/encoding/protojson"`，或引入 `contrib/encoding/json/v3` 覆盖 `json` 名以获得 proto 语义（注意不要同时注册两个 `json` 实现）。
3. **【文档】README 与代码不符。** `README.md:38` 写「a lightweight in-memory repository」、`:69` 写「The in-memory data layer intentionally stays simple」；实际数据层是 Ent + MySQL/SQLite，还有 `aip-go/ents` 的 SQL 过滤翻译。官方 layout main 同样如此，属于上游文档债。
4. **【CI】`.github/workflows/go.yml` 用 `go-version: ^1.20`**，`setup-go@v5.0.0`，与 `go.mod` 的 `go 1.25.7` 不匹配；且 CI 只跑 `go build`/`go test`，未跑 `make all` 校验生成代码一致性。
5. **【配置】`configs/config.yaml` 的 `data.redis` 段无任何消费方。** `conf.proto` 定义了 `Data.Redis`，但 `NewData` 只读 `GetDatabase()`。要么删掉，要么补 Redis 客户端（也是演示 `*Data` 持有多个长连接客户端的好例子）。
6. **【代码生成】生成器版本头滞后。** `todo_http.pb.go` 头为 `protoc-gen-go-http v2.9.2`，而 `buf.gen.yaml` 用 `/v3` 伪版本 `v3.0.0-20260526000039-30da04b769dc`。上游 main 的生成器已是 `v3.0.0`。**建议**把插件升到 `v3.0.0` 并 `make api`，同时可拿到 `HttpBody` 响应、ServerStream 读写 deadline 等新特性。
7. **【API 语义】流式 RPC 契约与实现/路由不一致。** (a) 生成器把 streaming RPC 硬编码注册为 `GET`，`SyncTodos` 在 proto 中声明的 `POST /v1/todos/sync body:"*"` 实际不生效；(b) `WatchTodos` 注释承诺「持续推送每次变更」，实现只发一页 `snapshot` 事件后关闭流；(c) `openapi.yaml` 对流式接口仍生成普通 200 响应描述。
8. **【错误处理】未使用 `protoc-gen-go-errors`。** `error_reason.proto` 只用 `.String()` 取 reason，`biz` 手写两个 `var`。上游提供 `cmd/protoc-gen-go-errors` 可从枚举生成 `ErrXxx` 与 `IsXxx`，减少手写漂移（迁移指南与 `AGENTS.md` 的「Error reasons」小节都指向这个方向）。
9. **【生产安全】`configs/config.yaml` 默认 `debug: true` + `auto_migrate: true`。** 注释已提示生产要关，但模板默认值对直接拿去部署的人有风险；建议默认 false 或用环境变量控制。
10. **【健壮性】`ListTodos` 的 `page_size` 只有默认值 20，没有上限。** 客户端可传极大值造成大查询；AIP 建议设 `max_page_size`。`filter` 复杂度也没有限制。
11. **【可观测】中间件偏少。** 只挂了 `recovery` + `validate`，未启用 `logging`/`metrics`/`tracing.Server()`/`ratelimit`/`circuitbreaker`；日志靠 `TraceAttrs` 注入但没有 trace middleware 生成 span。上游 `contrib/otel/tracing.Server()`、`middleware/logging`、`middleware/metrics` 都是现成扩展点。
12. **【仓库】`CLAUDE.md` 在本地是文本桩（`AGENTS.md`），上游是符号链接。** Windows 检出符号链接的常见结果；不影响构建，但 `AGENTS.md` 才是契约源，改契约时不要改 `CLAUDE.md`。
13. **【跟进上游】main 比 v3.0.0 多 7 个提交**，其中 3 个值得关注：`fix(config): resolve bool placeholders to bool (#3891)`（若配置里用 `${FLAG:true}` 这类布尔占位符，v3.0.0 可能有类型问题）、`fix(transport): escape path values and fix form encoding (#3902)`、以及新增 `contrib` oauth2 middleware。若后续升级 `kratos/v3` 到 main 之后的正式版，可一并获得。

---

## 13. 验证记录

| 命令 | 结果 |
|---|---|
| 逐文件比对本地 vs `kratos-layout` main | 仅 `CLAUDE.md` 不同（符号链接 vs 文本桩），其余 61 文件完全一致；`AGENTS.md` 哈希相同 |
| `go test ./...`（GOPROXY=https://proxy.golang.org） | 失败：代理返回 `Bad Gateway`，依赖无法下载（环境问题，非代码问题） |
| `go test ./...`（GOPROXY=https://goproxy.cn,direct，Go 1.27.3） | 编译失败：`undefined: http2.TrailerPrefix`（见 §12-1） |
| `go test -tags=http2legacy ./...` | ✅ **`internal/data` 0.798s ok；`internal/service` 0.602s ok；其余包无测试文件，全部编译通过** |
| 上游源码获取 | `git clone` 被代理阻断；改用 `curl -sL https://codeload.github.com/go-kratos/kratos/tar.gz/refs/heads/main` + `api.github.com` 取 release/commits/compare，成功落地完整 main 源码 |

复现命令：

```bash
# 推荐：与 go.mod 一致的 Go 版本
GOTOOLCHAIN=go1.25.7 GOPROXY=https://goproxy.cn,direct go test ./...

# 或在本机 Go 1.27 上临时绕过
go test -tags=http2legacy ./...
```

---

## 14. 速查：新增一个资源（把 `AGENTS.md` 的 checklist 落到本仓库）

1. `api/<domain>/v1/<res>.proto` 定义 `Create/Get/List/Update/Delete<Resource>` + `error_reason.proto`，`make api`（会同时刷新 `openapi.yaml`）。
2. `internal/biz/<res>.go`：定义 DO、`<Resource>Repo` 接口、`<Resource>Usecase`、`ListOption` 助手、`Err<Resource><Cause>`；注册进 `biz.ProviderSet`。
3. `internal/data/<res>.go`：实现 Repo，构造函数返回 `biz.<Resource>Repo`；必要时在 `internal/data/ent/schema/` 加 schema 并 `go generate ./...` 重生成 ent；注册进 `data.ProviderSet`。
4. `internal/service/<res>.go`：`convert<Resource>` + 内联 reply、嵌入 `Unimplemented<Resource>ServiceServer`；注册进 `service.ProviderSet`。
5. `internal/server/{grpc,http}.go`：`v1.Register<Resource>ServiceServer` / `...HTTPServer`。
6. `make all`（buf + wire + ent + `go mod tidy`），`go test ./...`。
7. 测试：service 层 fake `biz.Repo`，data 层用 SQLite 真库。

---

## 附录 A：本地完整文件清单（62 个）

```
.gitattributes  .github/workflows/go.yml  .gitignore
AGENTS.md  CLAUDE.md  Dockerfile  LICENSE  Makefile  README.md
buf.gen.config.yaml  buf.gen.yaml  buf.lock  buf.yaml
go.mod  go.sum  openapi.yaml
api/todo/v1/{todo.proto,error_reason.proto,todo.pb.go,todo_grpc.pb.go,todo_http.pb.go,error_reason.pb.go}
cmd/server/{main.go,wire.go,wire_gen.go}
configs/config.yaml
internal/conf/{conf.proto,conf.pb.go}
internal/server/{server.go,grpc.go,http.go}
internal/service/{service.go,todo.go,todo_test.go,README.md}
internal/biz/{biz.go,todo.go,README.md}
internal/data/{data.go,todo.go,todo_test.go,README.md}
internal/data/ent/{client.go,ent.go,enttest/enttest.go,generate.go,hook/hook.go,
  migrate/{migrate.go,schema.go},mutation.go,predicate/predicate.go,runtime.go,
  runtime/runtime.go,schema/{mixin.go,todo.go},todo.go,todo/todo.go,todo/where.go,
  todo_create.go,todo_delete.go,todo_query.go,todo_update.go,tx.go}
```

## 附录 B：上游 Kratos 核心包速查

```
app.go options.go version.go           应用生命周期 / Option / Release=v3.0.0
config/                               Config 接口、${K:default}、file/env source、watcher、Get[T]
log/                                  slog 门面、handler/filter/level/builder
errors/                               *Error、types.go、wrap.go（标准库包装）、errors.proto
middleware/                           Chain + recovery/validate/logging/metadata/ratelimit/
                                      circuitbreaker/selector
transport/http/                       server/codec/context/router/binding/stream(SSE+WS)/status
transport/grpc/                       server/client/interceptor/codec/resolver
encoding/                             json / protojson / proto / form / xml / yaml
registry/ selector/ metadata/          服务注册发现、负载均衡、元数据
internal/                             endpoint/host/matcher/httputil/group/...
cmd/kratos/                           new / proto / run / upgrade / change
cmd/protoc-gen-go-http/               HTTP 生成器（release=v3.0.0）+ httpTemplate.tpl
cmd/protoc-gen-go-errors/             错误码生成器
contrib/                              otel / config / registry / middleware / encoding / polaris ...
docs/migration/v2-to-v3{,_zh}.md     官方 v3 迁移指南
ROADMAP.md                           已完成/规划能力清单
```
