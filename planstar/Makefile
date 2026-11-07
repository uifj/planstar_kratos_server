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
# 运维面校验（探针/鉴权/压测/告警配置）请单独执行 make ops-check
check: fmt vet test

.PHONY: ops-check
# ops-check 运维面校验：探针/鉴权测试 + 端点压测；Docker 可用时额外校验
# Prometheus 规则、Alertmanager 配置与 compose。Docker 缺失时跳过外部校验。
ops-check:
	go test ./internal/server -run 'TestOps|TestHealthz|TestReadyz' -count=1 -v
	@if command -v docker >/dev/null 2>&1; then \
		echo "==> promtool check rules"; \
		docker run --rm --entrypoint promtool \
			-v "$$PWD/deploy/prometheus/rules:/rules" \
			prom/prometheus:v2.53.0 check rules /rules/alerts.yml; \
		echo "==> amtool check-config"; \
		ALERTMANAGER_TEMPLATE=deploy/alertmanager/alertmanager.yml \
		ALERTMANAGER_RENDERED=/tmp/planstar-alertmanager.yml \
		ALERTMANAGER_RENDER_ONLY=1 \
		AM_SMTP_SMARTHOST=smtp.example.com:587 \
		AM_SMTP_FROM=alert@example.com \
		AM_SMTP_AUTH_USERNAME=user \
		AM_SMTP_AUTH_PASSWORD=pass \
		AM_ALERT_EMAIL_TO=ops@example.com \
			sh deploy/alertmanager/entrypoint.sh >/dev/null; \
		docker run --rm --entrypoint amtool \
			-v "/tmp/planstar-alertmanager.yml:/etc/alertmanager/alertmanager.yml:ro" \
			prom/alertmanager:v0.27.0 \
			check-config /etc/alertmanager/alertmanager.yml; \
		echo "==> docker compose config"; \
		docker compose -f docker-compose.yml config -q; \
	else \
		echo "skip promtool/amtool/compose：未找到 docker"; \
	fi

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
