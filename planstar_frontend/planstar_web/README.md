# Planstar Web

Planstar 官方网站，采用 **pnpm workspaces + Turborepo + Next.js 15 App Router** 的 Monorepo 架构，
视觉与交互严格对齐 Flutter 客户端 `flutter_planstar_app` 的设计系统，并完整对接
`planstar_kratos_server` 的公开接口。

## 技术栈

| 层级 | 选型 | 说明 |
| --- | --- | --- |
| 包管理 / 构建 | pnpm 9 + Turborepo 2 | workspace 依赖、任务编排与缓存 |
| 框架 | Next.js 15（App Router）+ React 19 | 服务端渲染、动态路由、反向代理 |
| 语言 | TypeScript 5（strict） | 全仓统一 `tsconfig.base.json` |
| 样式 | Tailwind CSS v4 | 设计令牌通过 CSS 变量接入 |
| 规范 | ESLint 9（flat config）+ Prettier 3 | 共享规则包 `@planstar/config` |
| 测试 | Vitest 3 + Testing Library + Playwright | 单元 / 组件 / E2E |
| CI | GitHub Actions | lint → typecheck → test → build → e2e |

## 目录结构

```
planstar_web/
├── apps/
│   └── web/                     # Next.js 官网
│       ├── src/app/             # 路由：首页、企划列表/详情、榜单、关于
│       ├── src/components/      # 站点级组件（导航、页脚、企划卡片、榜单）
│       ├── src/lib/             # 服务端 API 工厂、格式化工具、站点常量
│       ├── e2e/                 # Playwright 冒烟用例
│       └── public/              # 品牌字体（woff2）与图标
├── packages/
│   ├── design-tokens/           # L0 色板 / L1 语义色 / 字号 / 间距 / 圆角
│   ├── ui/                      # 无业务 UI 组件库（Button/Card/Tag/Price/…）
│   ├── api-client/              # 类型化接口客户端（超时、错误、鉴权、金额）
│   └── config/                  # 共享 ESLint 规则
├── .github/workflows/ci.yml
├── turbo.json
├── pnpm-workspace.yaml
└── tsconfig.base.json
```

## 快速开始

```bash
# 1. 安装依赖（Node >= 20.11，pnpm >= 9）
pnpm install

# 2. 配置环境变量
cp .env.example .env.local
# 按需修改 PLANSTAR_API_BASE，例如 http://127.0.0.1:8098/

# 3. 启动官网（默认 http://localhost:3000）
pnpm dev
```

后端未启动时页面不会崩溃：聚合楼层会独立降级为提示条或空状态。

## 环境变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `PLANSTAR_API_BASE` | `http://127.0.0.1:8098/` | Kratos 服务地址，**必须以 `/` 结尾** |
| `PLANSTAR_API_TIMEOUT_MS` | `15000` | 服务端请求超时（毫秒） |
| `NEXT_PUBLIC_SITE_URL` | `http://localhost:3000` | 站点公开地址，用于 metadata |

生产环境示例：`PLANSTAR_API_BASE=https://api.planstar.top/`。

## 后端对接

### 请求链路

- **服务端渲染**：`createServerApiClient()` 直接请求 `PLANSTAR_API_BASE`，
  从入站 Cookie 读取 `psSession` 并透传，避免多用户串号。
- **浏览器端**：后端 CORS 白名单仅允许同源，因此统一走 Next rewrite
  `/api/backend/:path*` → `${PLANSTAR_API_BASE}:path*`，由 Next 转发 Cookie。
- 所有请求由 `@planstar/api-client` 统一处理：
  - 15s 超时（`AbortController`）
  - `User-Agent` / `dvid` / `Cookie: psSession=` 注入（浏览器自动跳过 forbidden header）
  - 非 2xx 读取 `{ code, message, traceId }` 并抛 `ApiError`
  - 网络错误、超时、解析错误分类，`toUserMessage()` 输出统一中文文案
  - 从 `psSession` 响应头或 `Set-Cookie` 捕获会话

### 已接入端点

| 方法 | 路径 | 用途 |
| --- | --- | --- |
| GET | `v1/home/banners` | 首页运营 banner |
| GET | `v1/categories` | 企划分类 |
| GET | `v1/projects/hot` | 热门企划 |
| GET | `v1/projects` | 企划列表（分类 / 状态 / 排序 / 分页） |
| GET | `v1/projects/{id}` | 企划详情 |
| GET | `v1/projects/{id}/reward-tiers` | 回报档位 |
| GET | `v1/projects/{id}/updates` | 进度更新 |
| GET | `v1/projects/{id}/faqs` | 常见问题 |
| GET | `v1/projects/{id}/reviews` | 支持者评价 |
| GET | `v1/rankings` | 热度 / 达成 / 冲刺榜（limit ≤ 50） |

金额字段统一为**分**，展示层只允许使用 `formatYuan` / `formatCompactYuan` 转换。
`ProjectSummary` / `ProjectDetail` 的字段与后端 `biz` 层 JSON tag 一一对应，
详见 [types.ts](packages/api-client/src/types.ts)。

### 约束

- 后端当前没有搜索、城市列表等接口，官网不提供对应入口，也不使用假数据兜底。
- 支持、下单、订单查询在 Planstar App 内完成；官网只展示公开企划与进度。

## 设计系统

令牌来源：`flutter_planstar_app/packages/infras/core_design_system`。

- **L0 原始色板 → L1 语义槽**：`packages/design-tokens/src/tokens.ts`（TS）与
  `theme.css`（CSS 变量）双份同步，组件只消费语义槽。
- 品牌色 `#1C66E5`，价格色 `#E54545`，页面底 `#F9FAFC`，卡片白底 + `1px` 细描边 + 12px 圆角，无阴影。
- 字号阶梯：10 / 12 / 14 / 16 / 20 / 32；字重 400 / 500 / 700。
- 间距 4px 基准；圆角 4 / 8 / 12 / 16 / 20 / 999。
- 品牌字体 `ChillRoundGothic`（woff2，`font-display: swap`），缺失时降级
  `PingFang SC / Noto Sans SC / Microsoft YaHei / system-ui`。
- 亮/暗主题：`<html class="dark">` 切换，首屏内联脚本避免闪烁，选择持久化到 `localStorage`。
- 价格与统计数字统一 `tabular-nums`，避免位数变化导致布局跳动。

Tailwind 工具类映射在 [globals.css](apps/web/src/app/globals.css) 的 `@theme inline` 中，
例如 `bg-brand`、`text-text-primary`、`text-title`、`rounded-md`。

## 常用脚本

```bash
pnpm dev          # 启动官网
pnpm build        # 生产构建
pnpm lint         # ESLint（全仓，--max-warnings=0）
pnpm typecheck    # TypeScript（全仓）
pnpm test         # Vitest 单元 / 组件测试
pnpm test:e2e     # Playwright（需先 pnpm build）
pnpm format       # Prettier 写入
pnpm format:check # Prettier 校验
pnpm clean        # 清理产物
```

## 测试策略

- **单元**：`api-client` 覆盖 URL 规范化、查询参数、裸 JSON、错误体映射、网络/超时/解析异常、
  `psSession` 捕获、榜单 limit 夹取；金额与达成率格式化。
- **组件**：`ProjectCard` 渲染标题、状态、价格、进度、支持人数与详情链接。
- **E2E**：首页品牌与导航、列表筛选入口、三类榜单、关于页、404。
  E2E 不依赖真实后端（使用不可达的 API 地址验证降级路径）。

## 部署

```bash
pnpm install --frozen-lockfile
pnpm build
PLANSTAR_API_BASE=https://api.planstar.top/ pnpm --filter @planstar/web start
```

- 推荐 Node 20 LTS 或更高版本，进程需常驻（Next 服务端渲染 + rewrite 代理）。
- 若部署到 Vercel，直接导入仓库并设置上述环境变量即可。
- CI 已配置 lint / typecheck / unit / build / e2e 全流程。

## 扩展指引

- **新增页面**：在 `apps/web/src/app` 下创建 `page.tsx`，数据层使用
  `createServerApiClient()` + `load()`，页面设置 `dynamic = 'force-dynamic'`。
- **新增接口**：先在 `packages/api-client/src/types.ts` 补类型，再在 `client.ts` 增加端点方法，
  并补充对应单元测试。
- **新增组件**：无业务语义的放 `packages/ui`，站点级组合放 `apps/web/src/components`；
  颜色/间距只引用令牌，不写死色值。
