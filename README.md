# rollfuse + Go + Prometheus

A feature-flagged checkout service, instrumented with Prometheus, running
against [rollfuse](https://rollfuse.com)'s [Go SDK](https://github.com/rollfuse/go-sdk).
Clone it, run one command, and watch a rollout happen on a live Grafana
dashboard — no rollfuse account required.

```bash
git clone https://github.com/rollfuse/examples-go-prometheus.git
cd examples-go-prometheus
docker compose up --build
```

Then open **http://localhost:3000** (Grafana, no login needed) and watch
the **"rollfuse checkout example"** dashboard fill in: a synthetic load
generator hits the service continuously, so you'll see live traffic split
50/50 across the `checkout-redesign` flag's two variations within seconds
— evaluation rate, request outcomes, and p95 latency, each broken down by
variation.

## What this demonstrates

- **Local flag evaluation** (`cmd/checkout/main.go`): every request calls
  `client.Evaluate(...)`, which resolves entirely from a cached,
  versioned Configuration — no network call, no added latency, per
  [ADR 0004](https://github.com/rollfuse/go-sdk#readme) (ship a flag check
  on every request path without worrying about tail latency).
- **Prometheus instrumentation**: three metrics —
  `rollfuse_flag_evaluations_total` (by flag, variation, reason),
  `checkout_requests_total` (by variation, outcome), and
  `checkout_request_duration_seconds` (a histogram, by variation) — so a
  rollout's effect on both business outcomes and latency is visible on
  one dashboard, not just "is the flag on."
- **A rollout you can watch happen**: `config/checkout-config.json` ships
  a 50/50 rollout split. Change the percentages (or flip `enabled` to
  `false`), restart the `mock-rollfuse-api` container, and watch the
  dashboard's variation split move within one Prometheus scrape interval.

## How it's wired

```
loadgen ──> checkout ──> mock-rollfuse-api   (GET /v1/config, POST /v1/exposure-events)
               │
               └─ /metrics ──> prometheus ──> grafana
```

`mock-rollfuse-api` (`cmd/mock-rollfuse-api/main.go`) is a small stand-in
for the real rollfuse platform API — it serves the static Configuration in
`config/checkout-config.json` and logs (accepts) exposure events, so this
example runs completely self-contained. Point `checkout` at a real
rollfuse environment instead by setting `ROLLFUSE_API_BASE_URL` and
`ROLLFUSE_SERVICE_CREDENTIAL` and dropping `mock-rollfuse-api` from
`docker-compose.yml` — the application code doesn't change at all.

## Try it against your own rollfuse project

```bash
docker compose stop mock-rollfuse-api
ROLLFUSE_API_BASE_URL=https://api.rollfuse.com \
ROLLFUSE_SERVICE_CREDENTIAL=<your Service Credential> \
  docker compose up checkout prometheus grafana loadgen
```

## Development

```bash
go build ./...
go vet ./...
```

`checkout` reads `ROLLFUSE_API_BASE_URL` (default
`http://localhost:8090`), `ROLLFUSE_SERVICE_CREDENTIAL` (default
`demo-credential`) and `LISTEN_ADDR` (default `:8080`).
`mock-rollfuse-api` reads `CONFIG_PATH` (default
`/config/checkout-config.json`) and `LISTEN_ADDR` (default `:8090`).

## Related

- [`rollfuse/go-sdk`](https://github.com/rollfuse/go-sdk) — the SDK this
  example runs.
- [`rollfuse/js-sdk`](https://github.com/rollfuse/js-sdk) — the JS/TS SDK
  family (server, browser, React).
