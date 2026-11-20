# =====================================================================
# Planstar 官网（Next.js 15 standalone）多阶段构建
#   构建上下文：planstar_frontend/planstar_web（pnpm monorepo 根）
#   运行方式：node apps/web/server.js，容器内监听 3000，由 Traefik 反代
# =====================================================================
FROM node:22-alpine AS base
ENV PNPM_HOME=/pnpm \
    PATH=/pnpm:$PATH \
    CI=1 \
    NEXT_TELEMETRY_DISABLED=1
RUN corepack enable && corepack prepare pnpm@9.15.9 --activate
WORKDIR /repo

# 先只拷贝清单，最大化利用层缓存
FROM base AS deps
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml .npmrc turbo.json tsconfig.base.json ./
COPY apps/web/package.json apps/web/package.json
COPY packages/api-client/package.json packages/api-client/package.json
COPY packages/design-tokens/package.json packages/design-tokens/package.json
COPY packages/ui/package.json packages/ui/package.json
COPY packages/config/package.json packages/config/package.json
RUN pnpm install --frozen-lockfile

FROM deps AS builder
# rewrites()/NEXT_PUBLIC_* 在构建期求值，必须在 build 阶段注入
ARG PLANSTAR_API_BASE=http://planstar-api:8098
ARG NEXT_PUBLIC_SITE_URL=https://www.planstar.top
ENV PLANSTAR_API_BASE=$PLANSTAR_API_BASE \
    NEXT_PUBLIC_SITE_URL=$NEXT_PUBLIC_SITE_URL
COPY . .
RUN pnpm --filter @planstar/web build

FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production \
    PORT=3000 \
    HOSTNAME=0.0.0.0 \
    NEXT_TELEMETRY_DISABLED=1
RUN addgroup -S -g 10001 nodejs && adduser -S -u 10001 -G nodejs nodejs

# standalone 产物已包含裁剪后的 node_modules（outputFileTracingRoot = monorepo 根）
COPY --from=builder --chown=nodejs:nodejs /repo/apps/web/.next/standalone ./
COPY --from=builder --chown=nodejs:nodejs /repo/apps/web/.next/static ./apps/web/.next/static
COPY --from=builder --chown=nodejs:nodejs /repo/apps/web/public ./apps/web/public

USER nodejs
EXPOSE 3000
CMD ["node", "apps/web/server.js"]
