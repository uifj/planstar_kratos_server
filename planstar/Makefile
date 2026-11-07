VERSION=$(shell git describe --tags --always 2>/dev/null || echo unknown)

.PHONY: build
# build
build:
	go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/planstar ./cmd/planstar

.PHONY: run
# run
run:
	go run ./cmd/planstar -conf configs/config.yaml

.PHONY: tidy
# tidy
tidy:
	go mod tidy

.PHONY: fmt
# fmt gofmt 校验（只检查不修改，失败时列出未格式化文件）
fmt:
	@command -v gofmt >/dev/null 2>&1 || { echo "未找到 gofmt，请检查 Go 工具链"; exit 1; }
	@unformatted="$$(gofmt -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "以下文件未通过 gofmt，请执行 gofmt -w 修复:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

.PHONY: test
# test
test:
	go test -race ./...

.PHONY: vet
# vet
vet:
	go vet ./...

.PHONY: check
# check 本地提交前的静态检查集合：gofmt + vet + test -race
check: fmt vet test

.PHONY: docker
# docker 构建生产镜像
docker:
	docker build -t planstar-api:dev .

.PHONY: sql-apply
# sql-apply 向已存在的 compose MySQL（服务名 mysql）补齐建库/演示/认证/生产补丁脚本
# 口令优先取环境变量 MYSQL_ROOT_PASSWORD（与 compose .env 同名），未设置时回退 root
sql-apply:
	docker compose exec -T mysql mysql -uroot -p$${MYSQL_ROOT_PASSWORD:-root} < ../docs/sql/00_all.sql
	docker compose exec -T mysql mysql -uroot -p$${MYSQL_ROOT_PASSWORD:-root} < ../docs/sql/14_mock_seed.sql
	docker compose exec -T mysql mysql -uroot -p$${MYSQL_ROOT_PASSWORD:-root} < ./sql/15_auth_support.sql
	docker compose exec -T mysql mysql -uroot -p$${MYSQL_ROOT_PASSWORD:-root} < ./sql/16_production_hardening.sql
