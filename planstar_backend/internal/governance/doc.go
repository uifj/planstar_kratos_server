// Package governance 不承载运行时代码，只用于存放架构治理断言测试。
//
// 断言范围（见 docs/ADR/ADR-0001 D6）：
//   - 分层依赖方向（biz 纯 Go、service 不依赖 data）
//   - 错误码单一事实源与契约文档一致性
//   - ADR 台账索引完整性
//   - 目录/文件阈值（默认报告，PLANSTAR_GOVERNANCE_STRICT=1 时阻断）
//   - 契约怪癖防回归（httpx.Plain 零调用、旧口径不再出现在规范中）
package governance
