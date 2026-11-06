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

.PHONY: test
# test
test:
	go test -race ./...

.PHONY: vet
# vet
vet:
	go vet ./...

.PHONY: check
# check 本地提交前的静态检查集合
check: vet test

.PHONY: docker
# docker 构建生产镜像
docker:
	docker build -t planstar-api:dev .

.PHONY: sql
# sql 导入（需本地 docker 的 ks-mysql 容器）
sql:
	docker cp ../docs/sql/00_all.sql ks-mysql:/tmp/00_all.sql && docker exec ks-mysql sh -c "mysql -uroot -proot < /tmp/00_all.sql"
	docker cp ../docs/sql/14_mock_seed.sql ks-mysql:/tmp/14_mock_seed.sql && docker exec ks-mysql sh -c "mysql -uroot -proot < /tmp/14_mock_seed.sql"
	docker cp ./sql/15_auth_support.sql ks-mysql:/tmp/15_auth_support.sql && docker exec ks-mysql sh -c "mysql -uroot -proot < /tmp/15_auth_support.sql"
