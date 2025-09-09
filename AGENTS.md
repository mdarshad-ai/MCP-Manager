# Repository Guidelines

## Project Structure & Module Organization
- `apps/desktop`: Electron + React UI (Chromium shell, routes: `/dashboard`, `/install`, `/server/:slug`, `/clients`, `/settings`, `/logs/:slug`).
- `services/manager`: Manager Daemon (installers, supervisor, health, client writers). Start with Node for v0; Go/Rust optional later.
- `packages/shared`: shared types and utilities (registry schema, IPC contracts).
- `scripts/`: bootstrap, packaging, login-daemon helpers; `docs/`: ADRs/design.
- Runtime data lives outside the repo at `~/.mcp/` (registry, installs, logs, cache, secrets) per the requirements.

## Build, Test, and Development Commands
- `npm ci` or `pnpm i`: install dependencies.
- `npm run dev:desktop`: start Electron + Vite/React with hot reload.
- `npm run dev:manager`: run daemon in watch mode.
- `npm run build`: package desktop app and compile daemon.
- `npm test`: unit tests; `npm run e2e`: Playwright UI tests.
- Optional: `make setup | make test | make lint | make package` as wrappers.
- If using Go/Rust for daemon: `go test ./...` or `cargo test` respectively.

## Coding Style & Naming Conventions
- Language: TypeScript (UI + Node daemon). If Go/Rust is used, follow community defaults.
- Indentation: 2 spaces (TS/JS/JSON/YAML); 4 (Go/Rust).
- Naming: camelCase (functions/vars), PascalCase (classes/components), kebab-case (dirs/packages), lower-case package names in Go; Rust crates kebab-case.
- Formatting/Linting: Biome (TS/JS) preferred; `gofmt`/`golangci-lint` (Go); `rustfmt`/`clippy` (Rust).

## Testing Guidelines
- UI: Vitest + React Testing Library; E2E: Playwright (happy-path flows: install, autostart toggle, health view).
- Daemon (Node): Vitest; Go: `testing`; Rust: built-in tests.
- Naming: `*.test.ts(x)`, `*_test.go`, Rust `mod tests {}`; mirror `apps/` and `services/` structure under `tests/` where applicable.
- Target ≥80% coverage; include error paths and restart policies.

## Commit & Pull Request Guidelines
- Conventional Commits (`feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `test:`).
- Keep PRs focused; include: summary, linked issues (`Closes #123`), screenshots (UI) or log excerpts (daemon), and updated tests/docs.
- Note client config impact (Claude/Cursor) and any security implications.

## Security & Configuration Tips
- Never commit secrets; provide `.env.example`. Store real secrets in OS keychain-backed vault.
- Default loopback-only for HTTP servers; warn on unsigned/untrusted sources.
- Write only under `~/.mcp/`; avoid modifying global system paths. Audit deps periodically (`npm audit`, `pnpm audit`, or `cargo audit`/`govulncheck`).
