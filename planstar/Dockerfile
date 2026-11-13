ARG GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct
ARG VERSION=unknown

FROM golang:1.27.0 AS builder

ARG GOPROXY
ARG VERSION

WORKDIR /src

# 先拷依赖清单，命中构建缓存，避免每次改代码都重新下载模块。
COPY go.mod go.sum ./
RUN GOPROXY=$GOPROXY go mod download

COPY . .

RUN CGO_ENABLED=0 GOPROXY=$GOPROXY go build -trimpath \
        -ldflags "-s -w -X main.Version=$VERSION" \
        -o /src/bin/planstar ./cmd/planstar

FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        tzdata \
        netbase \
        && rm -rf /var/lib/apt/lists/ \
        && apt-get autoremove -y && apt-get autoclean -y

ENV TZ=Asia/Shanghai

# 非 root 运行，降低容器逃逸风险。
RUN groupadd -r planstar && useradd -r -g planstar -d /app -s /sbin/nologin planstar

COPY --from=builder /src/bin /app

WORKDIR /app
RUN chown -R planstar:planstar /app
USER planstar

EXPOSE 8098 9108
VOLUME /data/conf

HEALTHCHECK --interval=30s --timeout=3s --start-period=20s --retries=3 \
    CMD curl -fsS http://127.0.0.1:9108/healthz || exit 1

CMD ["./planstar", "-conf", "/data/conf/config.yaml"]
