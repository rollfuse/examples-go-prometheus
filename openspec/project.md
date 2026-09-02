# Project Context

## What This Repo Is

An example repository under the `rollfuse` GitHub organization: a
feature-flagged demo checkout service showing
[`@rollfuse/go-sdk`](https://github.com/rollfuse/go-sdk) instrumented with
Prometheus, runnable end-to-end via `docker compose up` with no rollfuse
account required (a bundled mock stands in for the platform API). It is
not part of the rollfuse platform itself.

## Repository Shape

```text
/
├── cmd/
│   ├── checkout/            The demo service: evaluates a flag per request,
│   │                        records Prometheus metrics, simulates two
│   │                        checkout implementations.
│   └── mock-rollfuse-api/   Self-contained stand-in for the real platform
│                            API (GET /v1/config, POST /v1/exposure-events).
├── config/
│   └── checkout-config.json Static Configuration the mock API serves.
├── prometheus/
│   └── prometheus.yml       Scrape config.
├── grafana/
│   └── provisioning/, dashboards/   Auto-provisioned datasource + dashboard.
└── docker-compose.yml       The whole stack, one command.
```

## Working On This Repo

- Keep it runnable with a single `docker compose up` — never add a step
  that requires manual setup (an account, a token, a config edit) before
  the demo works.
- `go build ./...` and `go vet ./...` must stay clean; there is no test
  suite beyond that and the CI compose smoke test
  (`.github/workflows/ci.yml`) — this is a demo, not a library, so
  correctness is verified by "does the documented flow actually work,"
  not unit tests.
- A change to `cmd/checkout`'s metrics or `grafana/dashboards/*.json`
  should keep the two in sync — a metric the dashboard doesn't chart, or
  a panel querying a metric that no longer exists, defeats the point.
- OpenSpec here is for planning nontrivial changes to the demo itself,
  not for specifying rollfuse platform behavior (that lives in
  `rollfuse/rollfuse`).
