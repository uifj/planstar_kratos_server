# Repository Guidelines

Planstar is a single Kratos v3 service implementing the Flutter client API. The
service lives in `planstar/` as an independent Go module (`module planstar`);
the repository root is a legacy kratos-layout template kept for reference and is
**not** built by CI.

## Project structure

```text
planstar/cmd/planstar/            entrypoint + manual dependency assembly (no Wire)
planstar/internal/conf/           config structs + env overrides (PLANSTAR_*)
planstar/internal/biz/            usecases, domain objects (DO), repo ports, domain events
planstar/internal/data/           repo impls (MySQL database/sql, Redis, Elasticsearch)
planstar/internal/service/        HTTP handlers (DTO <-> DO)
planstar/internal/server/         business HTTP server + admin server (/metrics /healthz /readyz)
planstar/internal/mq/             RabbitMQ publisher / consumer (DLX, trace propagation)
planstar/internal/observability/  Prometheus metrics + Jaeger tracing + trace logging
planstar/internal/pkg/            captcha / secret / session / httpx
planstar/configs/                 config.yaml / config.docker.yaml / config.prod.yaml
planstar/deploy/                  Prometheus / Alertmanager / Loki / Promtail / Traefik / Grafana
planstar/docker-compose.yml       single-host full-stack compose
docs/api/API_CONTRACT.md          client API contract (source of truth)
docs/sql/                         schema + seed for the 9 planstar_* databases
```

## Layering & dependency rules

Three shapes flow through three layers: DTO (HTTP), DO (domain), PO (storage).

```text
client -> DTO -> service -> DO -> biz -> DO -> data -> PO -> storage
                            ^              ^
                            | declares     | implements
                            +-- repo port -+
```

- `service` parses the HTTP request into a DO, validates at the boundary, and
  returns `biz` errors. Never imports `data` or storage drivers.
- `biz` is pure Go: usecases, DO, typed errors, and repo interfaces (the
  inversion seam). Never imports `net/http`, `database/sql`, redis, amqp or elastic.
- `data` implements the repo interfaces with the long-lived clients held by
  `*Data` (a `map[string]*sql.DB` for the 9 databases, plus one Redis client and
  one Elasticsearch client). Maps driver errors to `biz` errors.
- `cmd` is the only assembly point. There is no Wire generation.

## Contract quirks (must preserve)

- Auth uses `Cookie: psSession=...`, not Bearer.
- Login failure: `5xx` + `text/plain`. Send-code: always `200` +
  `{success,needCaptcha,errorMsg}`.
- Money: DB stores yuan (`DECIMAL`), API returns cents (×100).
- Pagination: `page` for activities/feeds, `pageIndex` for messages/orders.
- Enums resolved from `dict_enum`.

## Build & test

```bash
cd planstar
go build ./...
go vet ./...
go test ./...
```

## CI/CD

Workflows mirror the kratos-shop pipeline set:

- `CI` (`.github/workflows/ci.yml`) — build + vet + test in `planstar/` on push/PR.
- `Release` (`.github/workflows/release.yml`) — tag `v*.*.*` → semver image in
  GitHub Packages + GitHub Release + SSH deploy (GitHub Deployments).
- `GitHub Pages` (`.github/workflows/pages.yml`) — publish `site/` to Pages.
- `Dependency Graph` — GitHub-managed (enabled in repository settings).
- `Dependabot Updates` (`.github/dependabot.yml`) — weekly dependency PRs.

GitHub secrets used by deploy: `SERVER_HOST`, `SERVER_USER`, `SERVER_SSH_KEY`,
`SERVER_PORT`, `APP_DIR`, optional `GHCR_TOKEN`.

## Commits & security

- Conventional Commits: `feat:`, `fix:`, `refactor:`, `chore(deps):`, `docs:`, `test:`.
- Never commit real credentials in `configs/`; use `PLANSTAR_*` env overrides.
- Do not hand-edit generated artifacts; Planstar has none (hand-written handlers).
