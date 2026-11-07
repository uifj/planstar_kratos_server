> 适用版本：planstar 单体 · 最后核对：2026-11-07 · 维护：后端组
>
> **Deprecated**：本目录只存放历史资料，不代表当前运行代码，不得作为架构、接口或部署依据。

# Legacy 历史文档

本目录用于隔离与当前 `planstar/` 单体不一致的历史内容，避免新成员被误导。

## 目录内容

| 文件 | 原用途 | 当前状态 |
| --- | --- | --- |
| [architecture-microservices.md](architecture-microservices.md) | kratos-shop 微服务拆分、BFF、Consul/gRPC 调用关系 | **已废弃**，仅作历史对照 |

## 仍留在上级目录的历史文档

以下文档同样描述 kratos-shop 微服务模板（PostgreSQL、`shop_*` 库、`shop`/`admin`/`service/*` 多进程），**不适用于当前代码**，尚未迁移到本目录：

- `../configuration.md`
- `../database.md`
- `../development.md`
- `../evolution-plan.md`
- `../feasibility-p1.md`

当前有效文档请回到 [docs/README.md](../README.md)。

## 维护约定

- 不再修改本目录内容，除非需要补充归档说明。
- 新发现的失真文档应先标注 `Deprecated`，再迁入本目录，并在本索引登记。
- 当前唯一运行服务是 `planstar/`，业务端口 8098、运维端口 9108；任何与本事实冲突的旧描述均以代码为准。
