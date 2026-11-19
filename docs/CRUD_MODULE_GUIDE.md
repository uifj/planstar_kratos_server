# 新增 CRUD 业务域约定（ADR-0010 P2⑦）

> 借鉴 `.archive/go-wind` 的「CRUD 模块 11 步」，剔除 proto / Wire / Ent 代码生成，
> 落到 planstar 的手写分层（DTO → DO → PO）。新增一个可读写的业务域时按本清单逐项
> 推进，防止漏层、漏装配、漏契约、漏测试。上层约束见 `AGENTS.md`（分层/依赖方向/阈值）。

## 一、定契约（先写文档）

1. 在 `docs/api/API_CONTRACT.md`（C 端）或 `docs/api/ADMIN_CONTRACT.md`（9108 运维端）
   登记端点：方法、路径、请求字段、响应模型、错误码。
2. 遵守既有怪癖：裸 JSON 成功、`{code,message,traceId}` 错误、金额分/元、分页
   `page` vs `pageIndex`、`pageSize<=0→20`、硬编码枚举、`/v1`/`/v3`/`/v4` 不“统一”。

## 二、定 DO 与 repo 端口（biz 层）

3. 在 `internal/biz/<domain>.go` 定义 DO（领域对象）与业务枚举/错误；纯 Go，
   不 import `net/http` / `database/sql` / redis / amqp / elastic。
4. 在同文件声明 repo 端口（接口，`NewXxx` 不在此定义），由 **消费方** 声明、`cmd` 装配。
5. 跨域调用走端口，不 import 其它域的 `data` 包。

## 三、落 PO 与实现（data 层）

6. 在 `internal/data/<domain>.go` 实现 repo：`NewXxxRepo(d *Data)` 构造器注入
   `*Data` 连接池（`d.DB("domain")`）；PO 与 DO 在边界做映射（元/分转换在此完成）。
7. 驱动错误统一经 `data/shared.MapQueryError` 映射为 `biz` 错误；事务用
   `data/shared.WithTx`，勿手写 Begin/Commit/Rollback。

## 四、接 HTTP（service 层）

8. 在 `internal/service/<domain>.go` 定义 handler：解析请求 → 校验边界 → 组装 DO →
   调 biz 用例 → 返回裸 JSON；错误经 `service.fail` / `httpx.Error` 输出统一错误体。
9. 在 `internal/service/routes.go`（或对应路由注册处）登记路由；注意 `routes_test.go`
   的 57/56 路由清单断言——新增 C 端端点需同步更新该常量口径。

## 五、装配与验证（cmd + governance）

10. 在 `cmd/planstar/wiring.go` 按「基础设施 → 仓储 → 服务 → 传输」单向装配
    `NewXxxRepo` → `NewXxxUsecase` → handler 注册。
11. 验证：`go build ./...`、`go vet ./...`、`go test ./... -count=1`、`make governance`
    （含 `TestADRIndexComplete` / 依赖方向 / 路由清单断言）；涉及对外契约时补 ADR。
