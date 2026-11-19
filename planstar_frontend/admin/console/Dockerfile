# 控制台静态产物构建：Node 22 + npm ci（华为云镜像），运行层使用 nginx:alpine。
FROM node:22-alpine AS builder

ARG NPM_REGISTRY=https://repo.huaweicloud.com/repository/npm/

WORKDIR /src

# 显式复制构建所需文件，避免把本地 node_modules/dist 带入构建上下文。
# Umi Max 的 postinstall（max setup）依赖 config/src，因此源码在 npm ci 前复制。
COPY package.json package-lock.json .npmrc tsconfig.json ./
COPY config ./config
COPY public ./public
COPY src ./src
RUN npm config set registry "${NPM_REGISTRY}"
RUN npm ci --registry="${NPM_REGISTRY}"
RUN npm run build

FROM nginx:alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /src/dist /usr/share/nginx/html

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1/healthz >/dev/null || exit 1

CMD ["nginx", "-g", "daemon off;"]
