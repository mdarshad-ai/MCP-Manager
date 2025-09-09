# MCP Manager — UI Spec (v0)

Design system: shadcn/ui (New York), neutral palette, admin layout with collapsible sidebar (expanded by default, no persistence).

## Sidebar
- Items: Dashboard, Install, Servers, Clients, Logs, Settings.
- Servers: dynamic subitems grouped by status (Ready > Degraded > Down) with green/orange/red dots.

## Dashboard
- Columns: Status • Name • CPU • RAM • Uptime • Restarts • Last Ping • Actions (Logs, Details, Restart, Stop).
- Sorting: Status (Ready > Degraded > Down), then Name A–Z.
- Filters: chips [Running][Degraded][Stopped] + text search.
- Refresh: auto every 60s + manual Refresh button.
- Row actions: Logs, Details, Restart (confirm), Stop (confirm with 10s grace then force).

## Install
- Sources: git, npm, pip, Docker (image + compose when Docker detected).
- Fields: Source Type, URI/Path, Slug (auto-suggest, editable), Runtime (prefilled), Package Manager (contextual).
- Validation before Install: reachability, slug uniqueness, runtime detection, disk space.
- Progress: step list + live log pane; Cancel allowed during download/build.
- Docker defaults: images → stdio (switchable to HTTP); compose → HTTP only (default port 8080).

## Server Details (single scroll page with anchors)
- Overview, Execution, Environment, Permissions, Clients, Logs sections.
- Editable by default: Autostart, Args, Env Vars (not masked; create vault entries), Permissions (FS roots, Net hosts).
- Read-only by default: Transport, Command (unlock via Advanced toggle).
- Degraded criteria: missed pings, RTT > 1000ms, ≥1 restart in last 10m (no resource thresholds).

## Clients
- Clients: Claude Desktop, Cursor (Global), Continue, Cursor CLI, Claude Code, Codex CLI, Gemini CLI.
- Detection: command on PATH or known config path.
- Apply: Writer when path known; otherwise prompt once for path then remember. Always show diff confirmation before writing.
- Badge: Detected (green) / Not Found (gray).

## Logs
- Default: Follow OFF (toggle to follow).
- Search: simple inline search box.
- Retention: hybrid — 128MB per server; global cap 1GB, trim per file then oldest across files.

## Settings
- Defaults: Autostart ON, Dark mode ON (not system), Logs cap 1GB total.
