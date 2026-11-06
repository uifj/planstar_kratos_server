FROM golang:1.25.7 AS builder

COPY . /src
WORKDIR /src

# Build directly: the golang base image has no `make`, and all generated
# artifacts (*.pb.go, wire_gen.go, ent/**) are committed to the repository,
# so no code generation is needed inside the image.
RUN GOPROXY=https://goproxy.cn,direct go build -ldflags "-X main.Version=dev" -o ./bin/server ./cmd/server

FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
		ca-certificates \
		curl \
		netbase \
        && rm -rf /var/lib/apt/lists/ \
        && apt-get autoremove -y && apt-get autoclean -y

COPY --from=builder /src/bin /app

WORKDIR /app

EXPOSE 8000
EXPOSE 9000
VOLUME /data/conf

HEALTHCHECK --interval=10s --timeout=3s --start-period=40s --retries=10 \
	CMD curl -fsS http://127.0.0.1:8000/healthz || exit 1

CMD ["./server", "-conf", "/data/conf"]
