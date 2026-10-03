<p align="center">
  <img src="web/public/icon.svg" alt="Oktopus" width="96" height="96">
</p>

# Oktopus

**[Русский](README.md)**

**Documentation:** [Russian](docs/ru/index.html) · [English](docs/en/index.html)

Oktopus controls outbound internet traffic for an organization: a single HTTP/HTTPS exit point, clear rules for who may reach which destinations, an audit trail, and a web console for operators. In MITM mode the proxy can **inspect outbound HTTP requests** against custom rules after ACL allows the connection.

It fits corporate access filtering, compliance logging, LDAP integration, familiar Squid-style policy (`acl`, `http_access`), and customizable block pages for end users.

## Capabilities


| Area                  | What the system provides                                                                                                                                      |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Access control**    | Squid-style policy: source, destination, domains/SNI, ports, URLs, LDAP groups, named lists; `http_access` ordering, delay pools, upstream TLS verification |
| **Proxy modes**       | **Tunnel** (CONNECT without decryption) and **MITM** (HTTPS termination with a root CA on clients); plain HTTP, WebSocket over MITM                        |
| **Inspection**        | After ACL allow — ordered Lua rules on outbound requests (headers, body, allow/deny); MITM only                                                                |
| **Authentication**    | Client login to the proxy: local accounts or LDAP; ACL may use directory groups                                                                               |
| **Control panel**     | Start/stop proxy, ACL and inspection editors, API users, access log, reports, rule explain / inspect                                                          |
| **Logging & reports** | Persisted allow/deny decisions with context (host, user, matched rule, inspection) for investigations                                                          |
| **Policy validation** | CI runs ACL integration tests and bypass scenarios; the evaluate API returns allow/deny for request fields without sending traffic through the proxy          |


## Throughput

Measured on **localhost** (load generator and proxy on the same host), **6 cores ~4.3 GHz**, policy with on the order of **~1.5M** SNI checks and **outbound inspection** enabled (MITM). Tool: `go run ./cmd load`.


| Metric                                                                                  | Value                               |
| --------------------------------------------------------------------------------------- | ----------------------------------- |
| Peak RPS, **allow** (passes ACL, inspection, and full handling)                         | up to **~14,000 requests per second** |
| Peak RPS, **deny** (blocked at ACL, no full relay to origin)                            | up to **~45,000 requests per second** |
| Average **ACL + inspect** time per request                                              | **< 1 ms**                          |
| Proxy process RAM in this configuration                                                 | **~600 MB**                         |


## Quick start (Docker)

Git and [Docker Compose](https://docs.docker.com/compose/) required. Step-by-step: [deployment docs](docs/en/quickstart.html).

```bash
git clone https://github.com/Camyil-89/oktopus.git
cd oktopus
```

**Linux / macOS:** `./scripts/stack-up.sh` — writes `.env`, builds and starts the stack, prints UI URL and admin password.

**Windows:** `scripts\stack-up.bat` (uses `docker-compose.desktop.yml`).

After login, confirm **RUNNING** and configure clients with the listen address from [proxy settings](docs/en/manage/proxy.html).

## Development

1. `docker compose -f docker-compose.dev.yml up -d`
2. Copy `.env.example` to `.env`
3. `go run ./cmd serve` (API default `http://127.0.0.1:8000`)
4. In `web/`: `npm install`, `npm run dev`; set `NEXT_PUBLIC_API_URL=http://localhost:8000` in `web/.env.local`

Details: [developer intro](docs/en/index.html). Page map: [docs/DOC_MAP.md](docs/DOC_MAP.md). Build docs: `node docs/build.mjs`.

## CLI

```text
go run ./cmd serve          # API + proxy, migrations on startup
go run ./cmd load [flags]   # load test: GET through HTTP proxy
```

```bash
go run ./cmd load -url=https://example.com/ -workers=32 -duration=30s -proxy=127.0.0.1:8080
```

## Technology

Go (proxy and API), Next.js (console), PostgreSQL (settings and policy), ClickHouse (access log); OpenLDAP only in the dev stack for directory integration tests.

## Repository layout


| Path                | Contents                      |
| ------------------- | ----------------------------- |
| `cmd/`              | `serve`, `load`               |
| `internal/proxy/`   | Proxy core, ACL, HTTPS, inspection |
| `web/`              | Management UI                 |
| `docs/`             | Documentation ru/en           |
| `attack-and-tests/` | Integration checks            |
| `scripts/`          | `stack-up.*`, CI              |
