#!/usr/bin/env node
import { spawn, spawnSync } from 'node:child_process'
import process from 'node:process'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const procs = []
const __dirname = path.dirname(fileURLToPath(import.meta.url))
const ROOT = path.resolve(__dirname, '..')
process.chdir(ROOT)
const DAEMON_BIN = path.join(ROOT, 'services', 'manager', 'bin', 'mcp-manager')
const HEALTH_URL = 'http://127.0.0.1:38018/healthz'

function run(cmd, args, options = {}) {
  const p = spawn(cmd, args, { stdio: 'inherit', shell: false, cwd: ROOT, ...options })
  procs.push(p)
  p.on('exit', (code, signal) => {
    console.log(`[dev] ${cmd} exited code=${code} signal=${signal}`)
  })
  p.on('error', (err) => {
    console.error(`[dev] failed to start ${cmd}:`, err.message)
  })
  return p
}

function which(cmd) {
  const r = spawnSync(process.platform === 'win32' ? 'where' : 'which', [cmd], { stdio: 'ignore' })
  return r.status === 0
}

async function waitForHealth(url, timeoutMs = 15000, intervalMs = 500) {
  const start = Date.now()
  while (Date.now() - start < timeoutMs) {
    try {
      const res = await fetch(url, { method: 'GET' })
      if (res.ok) return true
    } catch {}
    await new Promise((r) => setTimeout(r, intervalMs))
  }
  return false
}

function startDaemon() {
  if (fs.existsSync(DAEMON_BIN)) {
    console.log('[dev] starting daemon binary')
    return run(DAEMON_BIN, [])
  }
  if (!which('go')) {
    console.error('[dev] Go not found. Install Go (`brew install go`) or build the daemon first: `npm run build:manager`')
    process.exit(1)
  }
  console.log('[dev] starting daemon via `go run` in services/manager')
  return spawn('go', ['run', './cmd/manager'], { stdio: 'inherit', shell: false, cwd: path.join(ROOT, 'services', 'manager') })
}

function startUI() {
  console.log('[dev] starting Vite dev server (apps/desktop)')
  return run('npm', ['-w', 'apps/desktop', 'run', 'dev'])
}

function shutdown() {
  console.log('[dev] shutting down...')
  for (const p of procs) {
    try { p.kill('SIGTERM') } catch {}
  }
}

process.on('SIGINT', shutdown)
process.on('SIGTERM', shutdown)

// Orchestrate: start daemon, wait for health, then start UI
const daemon = startDaemon()
;
(async () => {
  const ok = await waitForHealth(HEALTH_URL, 20000, 750)
  if (!ok) {
    console.error('[dev] daemon did not become healthy at', HEALTH_URL)
    console.error('[dev] check that Go is installed and no port conflict exists')
    shutdown()
    process.exit(1)
  }
  const ui = startUI()
  ui.on('exit', (code) => {
    shutdown()
    process.exit(code ?? 0)
  })
})()
