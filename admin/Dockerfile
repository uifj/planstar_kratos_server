FROM golang:1.26.5 AS builder

WORKDIR /src

COPY . .

ENV GOTOOLCHAIN=local
RUN CGO_ENABLED=0 GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn go build -trimpath -ldflags "-s -w -X main.Version=docker -X main.Name=admin.api" -o /out/server ./cmd/server

FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates \
        netbase \
        curl \
        tzdata \
        && rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/server /app/server

WORKDIR /app

EXPOSE 9100 9101 9110

VOLUME /data/conf

CMD ["./server", "-conf", "/data/conf"]
