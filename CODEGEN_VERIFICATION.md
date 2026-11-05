# 代码生成与仓库更新验证报告

生成时间：2026-11-05
执行范围：根 `README.md` / `Makefile` 中的生成与构建命令（用户确认「跑根 README 的生成命令」）

---

## 结论

**仓库无需更新。** 使用与 `go.mod` 一致的 Go 1.25.7 工具链，完整跑完
`make init` → `make api` → `make config` → `make generate` → `make build` → `go test ./...`
后，`git diff --stat` 为空，`git status` 只有本次会话新增的两个未跟踪文件：

```
?? .workbuddy-ai/
?? KRATOS_STUDY_NOTES.md
?? CODEGEN_VERIFICATION.md
```

即所有已检入的生成产物（protobuf / gRPC / HTTP / OpenAPI / Wire injector / Ent 代码）
与重新生成的输出**字节完全一致**，仓库本身已经是最新的。

---

## 执行的命令（Makefile 等价物）

环境没有 `make`，因此按 `Makefile` 逐条手工执行：

```bash
# make init
go install github.com/google/wire/cmd/wire@latest
go install github.com/bufbuild/buf/cmd/buf@latest
# → buf.exe 1.75.0、wire.exe v0.7.0（仅安装；wire 实际生成走 go:generate 的 go.mod 固定版本 v0.6.0）

# make config
buf generate --template buf.gen.config.yaml

# make api
buf generate --template buf.gen.yaml

# make generate
go generate ./...        # Wire injector + Ent 代码
go mod tidy

# make build
mkdir -p bin/ && go build -ldflags "-X main.Version=$(git describe --tags --always)" -o ./bin/ ./...

# 验证
go test ./...
git status --short
git diff --stat
```

固定环境：

```bash
GOPROXY=https://goproxy.cn,direct
GOROOT=/c/Users/21263/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.7.windows-amd64
PATH=$GOROOT/bin:/c/Users/21263/go/bin:/c/Program Files/Go/bin:$PATH
```

---

## 工具链排错过程（关键）

本机默认 Go 为 **1.27.3**，而 `go.mod` 声明 `go 1.25.7`。二者不一致导致三类失败：

| 尝试 | 结果 |
| --- | --- |
| Go 1.27.3 直接跑 | `go test ./...` 报 `undefined: http2.TrailerPrefix`（grpc v1.81.1 与 x/net v0.54.0 的 `//go:build !(go1.27 && !http2legacy)` 冲突） |
| Go 1.27.3 + `GOFLAGS=-tags=http2legacy` | `buf generate` 成功；但 `go generate ./...` 中 Ent 生成器报 `internal error: package "context" without types was imported from "entgo.io/ent"`，并往 `internal/data/ent/runtime.go` 写入错误的 `// +build tools` 头（已 `git checkout` 还原） |
| `GOTOOLCHAIN=go1.25.7`（经 1.27.3 包装器） | buf 插件编译报 `package crypto/internal/fips140/sha3 is not in std` 等一批错误——包装器没有把 GOROOT 正确传给子进程 `go run` |
| **直接调用 1.25.7 工具链自带 `go.exe` 并显式设置 `GOROOT`** | ✅ 全链路通过 |

另一处值得记录的差异：用 Go 1.27.3 重新生成时，`api/todo/v1/todo.pb.go` 出现 8 行变动，
全部是 protoc-gen-go 输出注释里项目符号 `-` → `*` 的排版差异（不同 Go 版本的 gofmt 注释渲染），
属于工具链错配产生的伪变更；换回 Go 1.25.7 后该文件与 HEAD 完全一致。

---

## 验证结果

| 步骤 | 结果 |
| --- | --- |
| `buf generate --template buf.gen.config.yaml` | EXIT=0，`internal/conf/conf.pb.go` 无变化 |
| `buf generate --template buf.gen.yaml` | EXIT=0，`api/todo/v1/*.pb.go`、`openapi.yaml` 无变化 |
| `go generate ./...` | EXIT=0，Wire 写出 `cmd/server/wire_gen.go`（字节一致），Ent 生成全部无变化 |
| `go mod tidy` | EXIT=0，`go.mod` / `go.sum` 无变化 |
| `go test ./...`（**无需任何构建标签**） | EXIT=0；`internal/data` 0.795s、`internal/service` 0.775s，其余包无测试文件 |
| `go build ... -o ./bin/` | EXIT=0，产出 `bin/server.exe`（约 31 MB，`bin/` 已在 `.gitignore` 中） |
| `git diff --stat` | 空 |

---

## 建议

1. **锁定工具链**：在 CI（`.github/workflows/go.yml` 目前写 `go-version: ^1.20`）与本地开发中统一使用 Go 1.25.7，
   或在仓库加 `toolchain` 指令 / `.tool-versions`，避免 1.27.x 下需要 `-tags=http2legacy` 且 Ent 生成器失效。
2. 若必须用 Go 1.27.x，测试需带 `-tags=http2legacy`；但代码生成链路仍应切回 1.25.x。
3. `Makefile` 的 `make all` 未包含 `make config`（只调 `api`、`generate`），
   根 README 建议明确「改了 `internal/conf/conf.proto` 需单独 `make config`」，或把 `config` 并入 `all`。
