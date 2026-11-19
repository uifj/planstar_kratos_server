# Repository Guidelines

Planstar is a single Kratos v3 service implementing the Flutter client API. The
service lives in `planstar_backend/` as an independent Go module (`module planstar`);
the repository root is a legacy kratos-layout template kept for reference and is
**not** built by CI.

## Project structure

```text
planstar_backend/cmd/planstar/            main.go (thin) + wiring.go + probes.go (no Wire)
planstar_backend/internal/platform/       conf / mq / observability / ops / pkg / authn / authz
planstar_backend/internal/biz/            usecases, DO, repo ports (+ biz/shared)
planstar_backend/internal/data/           repos (+ data/shared driver error mapping)
planstar_backend/internal/service/        HTTP handlers + route registry (+ service/events)
planstar_backend/internal/server/         business HTTP + admin server (JWT + Casbin)
planstar_backend/internal/governance/     CI-gated architecture assertions
planstar_backend/internal/testutil/       opt-in integration harness
planstar_backend/configs/                 config.yaml / config.local.yaml / config.lite.yaml / config.prod.yaml
planstar_backend/deploy/                  Prometheus / Alertmanager / Traefik / Grafana (+ loki/promtail/otel placeholders)
planstar_backend/docker-compose*.yml      base / local / prod / server overlay (4 files)
docs/api/API_CONTRACT.md          client API contract (source of truth); docs/api/ADMIN_CONTRACT.md for ops port
docs/ADR/                        architecture decision records (see docs/ADR/README.md index)
docs/sql/                         schema + seed for the 9 baseline planstar_* databases
planstar_frontend/admin/                   separate Go module (Wire + Ent + proto) and Umi console; NOT part of planstar
planstar_frontend/planstar_web/            Next.js read-only web; site/ static landing page
.archive/go-wind/                 read-only reference snapshots; excluded from build/CI/deploy
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
  Shared DO/errors/events/money helpers live in `biz/shared` (ADR-0001 P1).
- `data` implements the repo interfaces with the long-lived clients held by
  `*Data` (a `map[string]*sql.DB` pool for **10** domains, plus one Redis client
  and one Elasticsearch client). Runtime has **11** `planstar_*` databases:
  9 baseline + `planstar_shop` + `planstar_admin` (the latter belongs to the
  separate admin module). Maps driver errors to `biz` errors.
- `cmd` is the only assembly point. There is no Wire generation in `planstar_backend/`.
  `planstar_frontend/admin/` is a **different** Go module and does use Wire/Ent/proto.
- New read/write business domains follow the checklist in `docs/CRUD_MODULE_GUIDE.md`
  (ADR-0010 P2⑦); data-layer transactions go through `data/shared.WithTx`.

## Contract quirks (must preserve)

Verified against `planstar_backend/internal/service/` and `internal/server/` on 2026-11-19.
These are guarded by `internal/governance` tests — do not "fix" them by hand.

- Auth uses `Cookie: psSession=...`, **not** Bearer. Login writes only a HttpOnly
  `Set-Cookie`; it must **not** write a JS-readable `psSession` response header.
  `dvid` is the only value also emitted as a response header.
- Login failure: `400 BAD_REQUEST` for malformed input, otherwise `401 UNAUTHORIZED`,
  both with the uniform JSON error body `{code,message,traceId}`.
  **The old `5xx + text/plain` behaviour was removed**; `httpx.Plain` has no callers.
- Send-code: always `200` + `{success,needCaptcha,errorMsg}` for parameter,
  captcha and send failures. **Exception**: the two rate-limit gates return
  `429 RATE_LIMITED` (in-process IP limiter + Redis `SETNX` cooldown).
- Success responses are **bare JSON**, no `{code,message,data}` envelope.
- Money: DB stores yuan (`DECIMAL(12,2)`), API returns cents (×100, rounded).
  `priceCoins` / `todayReward` are counts of star coins (1 coin = 100 cents).
- Pagination: `page` for activities/feeds/projects/pledges, `pageIndex` for
  messages/orders; chat uses `limit`+`before`, ideas use `cursor`+`size`.
  `pageSize<=0 or >100` **falls back to 20** (not clamped to 100).
  `nextPageKey` is a page number, not a cursor.
- Enums are **hard-coded whitelists** in `biz/project.go`, `biz/payment.go`,
  `biz/project_idea.go` (e.g. `hot|new|ending|achievement`,
  `preheating|...|failed`, `wechat|alipay|unionpay|balance`). `planstar_meta.dict_enum`
  exists as seed data but is **not** read at runtime.
- Version numbers are intentionally inconsistent (`/v1`, `/v3`, `/v4`); never
  "normalize" them without a client-side release.

## Architecture governance

Decisions are recorded in `docs/ADR/` (index: `docs/ADR/README.md`). Current
top-level decision: **modular monolith** (`ADR-0001`) — no process split, no
proto/Wire in `planstar_backend/`; `ADR-0002` defines the target directory layout.

- Keep `biz` / `data` / `service` as first-level directories; business domains
  are second-level packages inside each layer (P2, not yet applied).
- Cross-domain calls go through a port declared by the **consuming** domain and
  wired in `cmd/planstar`; never import another domain's `data` package.
- Thresholds: one package ≤ 40 `.go` files and ≤ 1 domain; one file ≤ 30 KB or
  ≤ 800 lines; `main.go` ≤ 80 lines; one wiring function ≤ 30 arguments.
- `make check` is the local gate; CI additionally runs `internal/governance`
  tests for dependency direction, route inventory, error-code table and ADR index.
  Layout thresholds are strict by default (`PLANSTAR_GOVERNANCE_STRICT=0` only
  downgrades to report mode during P2 file moves).
- 9108 ops domain uses JWT (jwt/v5 HS256) + Casbin RBAC (ADR-0007) in
  internal/platform/{authn,authz}. The C-side psSession contract is unchanged.
  The shared ops token is an explicit `superadmin` policy, not a code bypass:
  deny-by-default, 401 vs 403, fail-closed. Secrets only via
  PLANSTAR_SECURITY_JWT_SECRET / PLANSTAR_SECURITY_USERS_JSON.
- `docs/OPEN_ITEMS.md` is the closed governance ledger (33 items); the work that
  still needs action is in `docs/FOLLOWUP_WORK.md` (TODO / DEBT / VERIFY / ENV / TRIGGER).
  Check both before starting architecture work, and update them (plus
  `docs/ADR/README.md`) when an item closes.
- `docs/research/archive-reference-integration.md` (registered as **ADR-0010**) is
  the phased reference-integration plan distilled from `.archive/go-wind` and
  `.archive/zenith-admin`. Execute its P0→P2 items in order; close each in
  `docs/FOLLOWUP_WORK.md`, and record a new ADR if a change touches the client contract.
  Excluded (never adopt): Ent ORM, proto/Wire codegen, Kratos v2 multi-service split,
  Node/Hono/Drizzle code, Semi Design, zenith `pageOffset` pagination semantics.

## Build & test

```bash
cd planstar_backend
go build ./...
go vet ./...
go test ./...          # unit tests, colocated, no infrastructure needed
make test-integration  # real-MySQL tests (build tag "integration")
```

Unit tests stay colocated (`*_test.go`) and are table-driven; shared harnesses
live in `internal/testutil`. Integration tests are guarded by
`//go:build integration` and read `PLANSTAR_TEST_MYSQL_DSN`; they auto-skip when
it is unset or unreachable, and clean up seeded rows via `testutil.Cleanup`.
`make check` runs gofmt + vet + `make governance` (architecture assertions in
`internal/governance`) + `go test -race`. Use `make lint` for the optional
golangci-lint pass, and `make ops-check` for probe/authz/alerting checks.
On Windows without gcc, `go test -race` requires CGO; CI covers it on Linux.

## CI/CD

Workflows mirror the kratos-shop pipeline set:

- `CI` (`.github/workflows/ci.yml`) — build + vet + test in `planstar_backend/` on push/PR.
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
