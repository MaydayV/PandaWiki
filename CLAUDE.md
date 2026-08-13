# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

PandaWiki is an AI-powered open-source knowledge base system. This is the "Fly Version" (乘风版) fork by MaydayV, with enhanced features over the original by Chaitin. Licensed under AGPL-3.0.

## Repository Layout

- `backend/` — Go backend (API server with embedded worker, migrations). Entry points: `backend/cmd/api/`, `backend/cmd/migrate/`
- `web/` — pnpm workspace root for frontend apps
- `web/admin/` — React 19 + Vite admin console (Redux Toolkit, MUI, TipTap editor)
- `web/app/` — Next.js 16 user-facing web app (MUI, markdown-it, mermaid)
- `web/packages/` — Shared packages: `@panda-wiki/icons`, `@panda-wiki/themes`, `@panda-wiki/ui`
- `sdk/rag/` — Go client library for the RAG service
- `backend/pro` — Git submodule for private Pro edition (PandaWikiPro)
- `docs/` — Deployment docs, feature comparison, version notes

## Backend Architecture (Layered)

Strict layer boundaries — do not put business logic in handlers:

- `handler/` — HTTP handlers (Echo v4). Receives request, calls usecase, returns response.
- `usecase/` — Business logic. All domain rules belong here.
- `repo/` — Data access layer: `pg/` (PostgreSQL/GORM), `cache/` (Redis), `mq/` (NATS), `ipdb/`
- `domain/` — Domain models and shared types
- `api/` — API request/response types
- `config/` — Configuration (Viper, env-based)
- `server/http/` — HTTP server setup and route registration
- `middleware/` — HTTP middleware
- `store/pg/` — PostgreSQL store with migrations
- `pkg/` — Shared packages (bot integrations, captcha, LDAP, OAuth, rate limiting)
- `apm/` — OpenTelemetry tracing, Sentry error tracking

Receiver names: short and consistent — `u` (usecase), `h` (handler), `r` (repo).

## Build & Development Commands

### Backend (all from `backend/`)

```bash
go mod tidy                          # Install dependencies
go build ./...                       # Build all packages
go build ./usecase                   # Build one package
make generate                        # Generate swagger + Wire DI code
make generate_pro                    # Generate pro swagger + Wire DI code
make lint                            # Full lint (generation + tidy + golangci-lint) — REQUIRED before commit
make dev                             # Build Docker images locally and start with docker compose
golangci-lint run                    # Quick lint only (no generation)
gofmt -w path/to/file.go && goimports -w path/to/file.go  # Format one file
```

### Frontend (all from `web/`)

```bash
pnpm install                         # Install all workspace dependencies
pnpm dev                             # Start both admin + app dev servers
pnpm build                           # Build both admin + app
pnpm api                             # Regenerate API client from swagger

# Per-app
cd admin && pnpm build               # Build admin only (tsc -b && vite build)
cd app && pnpm build                  # Build app only (next build)
cd app && pnpm lint                   # Lint app
cd app && pnpm format                 # Format app
cd app && pnpm format:check           # Check app formatting
```

## Testing

### Backend

```bash
go test ./...                        # All tests
go test ./usecase                    # One package
go test ./usecase -run TestName -v   # One test, verbose
go test -race ./usecase              # With race detector
```

### Frontend

No dedicated test scripts. Use lint, typecheck (`tsc`), and build as verification.

## Linting & Formatting

- **Backend**: `cd backend && make lint` is required before any backend commit. Config: `backend/.golangci.toml` (standard linters, gofmt + goimports formatters).
- **Frontend admin**: `cd web/admin && pnpm exec eslint src/path/to/file.tsx`. Prettier via `cd web && pnpm exec prettier --write .`
- **Frontend app**: `cd web/app && pnpm lint` and `pnpm format`
- Husky pre-commit hooks in `web/.husky/` run Prettier on staged files.

## Code Style

### Go

- Use `gofmt`/`goimports` — never hand-format imports. Group: stdlib → third-party → local.
- Tabs for indentation, standard Go formatting.
- Exported: PascalCase. Unexported: camelCase.
- Request structs end with `Req`, response structs with `Resp`.
- Return wrapped errors: `fmt.Errorf("...: %w", err)`. Use `errors.Is` for sentinel checks.
- Log internal details; return user-facing messages from handlers.
- Use `h.NewResponseWithError(...)` or `u.logger.Error(...)` — never panic.
- Propagate context: `c.Request().Context()` in handlers.
- Keep functions focused; prefer extracting domain logic into `usecase`.

### TypeScript / React

- Prettier formatting: 2-space indent, single quotes, trailing commas, LF line endings.
- Path aliases: `@/request/...`, `@/store` in admin code.
- Prefer explicit types for props and payloads. Avoid `any`.
- Use `type` for simple aliases; `interface` for component props when idiomatic.
- Function components with hooks. Keep state local unless shared across routes.
- Clean up subscriptions/AbortControllers/SSE clients in `useEffect` cleanup.
- Reuse generated request types from `@/request/types`.
- Do not edit generated swagger client files — use wrappers like `nodeStream.ts` for special protocols.
- UI libraries: MUI + `@ctzhian/ui`.

## Generated Code

- `web/admin/src/request/*` — Generated swagger client. Has disabled linting. Do not hand-edit.
- If an endpoint needs non-JSON behavior, create a thin wrapper alongside the generated code.
- Backend swagger uses `swag`. After handler doc changes: `cd backend && make generate`.
- After swagger changes: `cd web && pnpm api` to sync frontend request clients.

## Infrastructure

The system requires 9 services (managed via Docker Compose):
- PostgreSQL 16 (pgvector), Redis 7, NATS 2.10, 阿里云 OSS (S3 API), Caddy 2.10
- RAG-lite service (Go-based RAG engine)
- API server (port 8000), Consumer worker, Caddy reverse proxy

## Verification Checklist

- Backend change: `cd backend && make lint` + `go test` on affected packages.
- API contract change: search for all callers and update them together.
- Swagger modified: `cd web && pnpm api` before committing frontend code.
- Frontend admin: targeted `eslint` + `pnpm build` in `web/admin`.
- Frontend app: `pnpm lint` or `pnpm build` in `web/app`.
- If full builds fail because of pre-existing issues, state that explicitly.
