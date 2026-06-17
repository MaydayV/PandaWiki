# AGENTS.md

This file is for coding agents working in `PandaWiki`.
Prefer small, targeted changes that match the repository's existing patterns.
Communicate with the user in Chinese unless they explicitly ask otherwise.

## Source Of Truth

- There is no existing `AGENTS.md` in this repository.
- No Cursor rules were found in `.cursor/rules/` or `.cursorrules`.
- No Copilot instructions were found in `.github/copilot-instructions.md`.
- Root `CLAUDE.md` exists and should be treated as repository guidance.
- Key note from `CLAUDE.md`: respect backend layer boundaries and use Chinese for communication.

## Repository Layout

- `backend/`: Go backend services, APIs, workers, migrations, docs.
- `web/`: pnpm workspace for frontend apps.
- `web/admin/`: React + Vite admin console.
- `web/app/`: Next.js user-facing web app.
- `sdk/`: SDK-related code.

## Architecture Expectations

- Backend follows a layered structure.
- Put business logic in `backend/usecase/`.
- Keep HTTP-specific logic in `backend/handler/`.
- Keep persistence and external data access in `backend/repo/`.
- Put shared request/response and domain models in `backend/domain/` and `backend/api/`.
- Do not move business rules into handlers just to finish a feature quickly.

## Tooling Versions And Basics

- Go version: `1.24.3` from `backend/go.mod`.
- Frontend package manager: `pnpm` only.
- Admin app build tool: Vite.
- User app build tool: Next.js.
- Backend formatting/linting is driven by `gofmt`, `goimports`, and `golangci-lint`.
- Frontend formatting is primarily Prettier.

## Install Commands

Run these from the repository root unless noted otherwise.

### Backend

- Install Go dependencies: `cd backend && go mod tidy`
- Build all backend packages: `cd backend && go build ./...`

### Frontend

- Install web dependencies: `cd web && pnpm install`
- Start both frontend apps: `cd web && pnpm dev`

## Build Commands

### Backend

- Build all packages: `cd backend && go build ./...`
- Build one package: `cd backend && go build ./usecase`
- Generate swagger/wire code: `cd backend && make generate`
- Generate pro swagger/wire code: `cd backend && make generate_pro`

### Frontend

- Build all web apps: `cd web && pnpm build`
- Build admin only: `cd web/admin && pnpm build`
- Build app only: `cd web/app && pnpm build`
- Analyze admin build: `cd web/admin && pnpm build:analyze`

## Test Commands

### Backend

- Run all tests: `cd backend && go test ./...`
- Run one package: `cd backend && go test ./usecase`
- Run one test by name in one package: `cd backend && go test ./usecase -run TestName`
- Run one test with verbose output: `cd backend && go test ./usecase -run TestName -v`
- Run one package with race detector when needed: `cd backend && go test -race ./usecase`

### Frontend

- `web/admin` currently has no dedicated test script in `package.json`.
- `web/app` currently has no dedicated test script in `package.json`.
- For frontend verification, use lint, typecheck, and build as the practical test substitute.
- Admin typecheck via build pipeline: `cd web/admin && pnpm build`
- App lint: `cd web/app && pnpm lint`

## Lint And Format Commands

### Backend

- Lint backend: `cd backend && golangci-lint run`
- Full backend lint target: `cd backend && make lint`
- Format one file/package: `cd backend && gofmt -w path/to/file.go && goimports -w path/to/file.go`
- `cd backend && make lint` is required before commit for any backend code change.

Notes:

- `make lint` also runs generation steps and `go mod tidy`; it is heavier than `golangci-lint run`.
- `make lint` already covers backend formatting via configured formatters, so agents should treat it as the required backend verification step before commit.
- `backend/.golangci.toml` enables standard linters and formatters.

### Frontend

- Root formatting: `cd web && pnpm exec prettier --write .`
- Admin lint one file or folder: `cd web/admin && pnpm exec eslint src/path/to/file.tsx`
- App format: `cd web/app && pnpm format`
- App format check: `cd web/app && pnpm format:check`
- App lint: `cd web/app && pnpm lint`

## Code Style: Go

- Use `gofmt`/`goimports`; do not hand-format imports.
- Keep imports grouped by standard library, third-party, then local modules.
- Use tabs and standard Go formatting.
- Exported names use Go PascalCase; unexported names use camelCase.
- Receiver names should be short and consistent, usually `u`, `h`, `r`.
- Keep functions focused; prefer extracting domain logic into `usecase` over large handlers.
- Return wrapped errors with context using `fmt.Errorf("...: %w", err)`.
- Use `errors.Is` for sentinel checks.
- Log internal details, but return user-facing messages from handlers.
- Follow existing pattern: log with `u.logger.Error(...)` or `h.NewResponseWithError(...)` rather than panicking.
- Prefer explicit structs over untyped maps unless responding with a tiny ad hoc payload.
- Respect context propagation. Use `c.Request().Context()` in handlers.
- For streaming/SSE paths, preserve existing event shapes and flushing behavior.

## Code Style: TypeScript / React

- Follow Prettier formatting and the existing single-quote style.
- Use path aliases like `@/request/...` and `@/store` in admin code.
- Prefer explicit types for props and request payloads.
- Avoid `any`; lint only warns today, but new code should use concrete types where practical.
- Use `type` for simple object aliases and event payloads; use `interface` for component props when already idiomatic nearby.
- Components are function components with hooks.
- Keep state local unless it is shared across routes or features.
- Clean up subscriptions/AbortControllers/SSE clients in `useEffect` cleanup or modal close handlers.
- Reuse generated request types from `@/request/types` when available.
- Do not edit generated swagger client files unless absolutely necessary; prefer wrappers like `nodeStream.ts` for special protocols.
- Match existing UI libraries: MUI plus `@ctzhian/ui`.

## Naming Conventions

- Backend request structs end with `Req`, response structs with `Resp`.
- Usecase methods are verbs or verb phrases, e.g. `SummaryNode`, `MoveNode`.
- Frontend modal and page components use PascalCase filenames and component names.
- Request helpers in admin use verb-based names from generated swagger files or descriptive wrapper names.

## Error Handling Expectations

- Never swallow backend errors silently.
- If an error is user-actionable, return a clear message from the handler.
- If an error is for diagnostics only, log the wrapped error and return a generic message.
- In frontend code, surface actionable failures with `message.error(...)`.
- For async UI actions, always stop loading state in both success and error paths.
- For streaming flows, handle both transport errors and protocol-level `error` events.

## Generated Code And API Clients

- `web/admin/src/request/*` contains generated swagger client code.
- Generated files use disabled linting and should generally not be hand-edited.
- If an endpoint needs non-JSON behavior, create a thin wrapper alongside generated code instead of forcing the generated client.
- Backend swagger generation uses `swag`; if handler docs change significantly, regenerate docs with `cd backend && make generate`.
- If swagger files change, run `cd web && pnpm api` before committing frontend code so generated request clients stay in sync.

## Practical Guidance For Agents

- Read the nearby code before changing patterns.
- Prefer the smallest correct change.
- Preserve existing protocol contracts unless the task explicitly changes them.
- If the user asks for a review, only produce a review report; do not modify code as part of the review task.
- If changing an HTTP contract, search for every caller first.
- When touching backend request flows, verify both handler behavior and frontend request wrappers.
- When touching frontend request code, check interceptors and special transports like SSE.
- Do not introduce a new abstraction unless the same logic clearly repeats.

## Verification Checklist

- Backend change: run `go test` on the affected package at minimum.
- Any backend code change requires `cd backend && make lint` before commit.
- API contract change: search for all callers and update them together.
- If swagger files were modified, run `cd web && pnpm api` before committing any related frontend changes.
- Frontend admin change: run targeted `eslint` and, if feasible, `pnpm build` in `web/admin`.
- Frontend app change: run `pnpm lint` or `pnpm build` in `web/app` as appropriate.
- If full builds fail because of pre-existing unrelated issues, say so explicitly.
