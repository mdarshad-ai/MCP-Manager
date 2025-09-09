#!/usr/bin/env node
/**
 * Unified Development Script for MCP Manager
 * Orchestrates Go backend, TypeScript frontend, and Electron in development mode
 */

import { spawn, spawnSync } from 'node:child_process'
import process from 'node:process'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const ROOT = path.resolve(__dirname, '..')
process.chdir(ROOT)

// Configuration
const CONFIG = {
  daemon: {
    bin: path.join(ROOT, 'services', 'manager', 'bin', 'mcp-manager'),
    src: path.join(ROOT, 'services', 'manager'),
    health: 'http://127.0.0.1:38018/healthz',
    port: 38018
  },
  vite: {
    url: 'http://localhost:5173',
    port: 5173
  },
  electron: {
    enabled: false,
    delay: 2000 // Wait for vite before starting electron
  }
}

// Color codes for console output
const colors = {
  reset: '\x1b[0m',
  bright: '\x1b[1m',
  dim: '\x1b[2m',
  red: '\x1b[31m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  magenta: '\x1b[35m',
  cyan: '\x1b[36m',
  white: '\x1b[37m'
}

// Process management
const processes = new Map()
let shuttingDown = false

// Logging utilities
function log(component, message, color = colors.blue) {
  const timestamp = new Date().toISOString().split('T')[1].slice(0, 8)
  console.log(`${colors.dim}[${timestamp}]${colors.reset} ${color}[${component}]${colors.reset} ${message}`)
}

function logError(component, message) {
  log(component, message, colors.red)
}

function logSuccess(component, message) {
  log(component, message, colors.green)
}

function logInfo(component, message) {
  log(component, message, colors.cyan)
}

// Check if command exists
function which(cmd) {
  const result = spawnSync(process.platform === 'win32' ? 'where' : 'which', [cmd], { 
    stdio: 'ignore' 
  })
  return result.status === 0
}

// Wait for service to be healthy
async function waitForHealth(url, timeoutMs = 30000, intervalMs = 500) {
  const start = Date.now()
  log('health', `Waiting for ${url}...`, colors.yellow)
  
  while (Date.now() - start < timeoutMs) {
    try {
      const res = await fetch(url, { method: 'GET' })
      if (res.ok) {
        logSuccess('health', `Service ready at ${url}`)
        return true
      }
    } catch {}
    await new Promise((r) => setTimeout(r, intervalMs))
  }
  
  logError('health', `Service failed to become healthy at ${url}`)
  return false
}

// Start a process with enhanced logging
function startProcess(name, cmd, args, options = {}) {
  logInfo(name, `Starting ${cmd} ${args.join(' ')}`)
  
  const proc = spawn(cmd, args, {
    stdio: 'pipe',
    shell: false,
    cwd: ROOT,
    ...options
  })
  
  // Enhanced output handling with colored prefixes
  proc.stdout?.on('data', (data) => {
    const lines = data.toString().split('\n').filter(line => line.trim())
    lines.forEach(line => {
      console.log(`${colors.dim}[${name}]${colors.reset} ${line}`)
    })
  })
  
  proc.stderr?.on('data', (data) => {
    const lines = data.toString().split('\n').filter(line => line.trim())
    lines.forEach(line => {
      console.error(`${colors.red}[${name}]${colors.reset} ${line}`)
    })
  })
  
  proc.on('exit', (code, signal) => {
    if (!shuttingDown) {
      if (code === 0 || signal === 'SIGTERM') {
        logInfo(name, `Process exited (code=${code}, signal=${signal})`)
      } else {
        logError(name, `Process crashed (code=${code}, signal=${signal})`)
        // Auto-restart critical services
        if (name === 'daemon' && !shuttingDown) {
          log(name, 'Auto-restarting in 2 seconds...', colors.yellow)
          setTimeout(() => startDaemon(), 2000)
        }
      }
    }
    processes.delete(name)
  })
  
  proc.on('error', (err) => {
    logError(name, `Failed to start: ${err.message}`)
    processes.delete(name)
  })
  
  processes.set(name, proc)
  return proc
}

// Start Go daemon
function startDaemon() {
  // Check if binary exists
  if (fs.existsSync(CONFIG.daemon.bin)) {
    logInfo('daemon', 'Starting from binary')
    return startProcess('daemon', CONFIG.daemon.bin, [])
  }
  
  // Check if Go is installed
  if (!which('go')) {
    logError('daemon', 'Go not found. Install Go or build the daemon first: npm run build:manager')
    process.exit(1)
  }
  
  // Start with go run
  logInfo('daemon', 'Starting with go run')
  return startProcess('daemon', 'go', ['run', './cmd/manager'], {
    cwd: CONFIG.daemon.src
  })
}

// Start Vite development server
function startVite() {
  return startProcess('vite', 'npm', ['-w', 'apps/desktop', 'run', 'dev'])
}

// Start Electron
function startElectron() {
  return startProcess('electron', 'npx', ['electron', 'apps/desktop'], {
    env: {
      ...process.env,
      NODE_ENV: 'development'
    }
  })
}

// Graceful shutdown
async function shutdown(signal = 'SIGTERM') {
  if (shuttingDown) return
  shuttingDown = true
  
  console.log('\n')
  log('system', `Shutting down (${signal})...`, colors.yellow)
  
  // Stop processes in reverse order
  const stopOrder = ['electron', 'vite', 'daemon']
  
  for (const name of stopOrder) {
    const proc = processes.get(name)
    if (proc) {
      logInfo('system', `Stopping ${name}...`)
      try {
        proc.kill('SIGTERM')
        // Give process time to cleanup
        await new Promise(r => setTimeout(r, 500))
      } catch (err) {
        logError('system', `Error stopping ${name}: ${err.message}`)
      }
    }
  }
  
  // Force kill remaining processes
  for (const [name, proc] of processes) {
    try {
      proc.kill('SIGKILL')
    } catch {}
  }
  
  logSuccess('system', 'Shutdown complete')
  process.exit(0)
}

// Parse command line arguments
function parseArgs() {
  const args = process.argv.slice(2)
  const options = {
    electron: false,
    daemonOnly: false,
    viteOnly: false,
    skipHealth: false
  }
  
  for (const arg of args) {
    switch (arg) {
      case '--electron':
        options.electron = true
        break
      case '--daemon-only':
        options.daemonOnly = true
        break
      case '--vite-only':
        options.viteOnly = true
        break
      case '--skip-health':
        options.skipHealth = true
        break
      case '--help':
        console.log(`
MCP Manager Unified Development Script

Usage: node scripts/dev-unified.mjs [options]

Options:
  --electron      Also start Electron (desktop app mode)
  --daemon-only   Start only the Go daemon
  --vite-only     Start only the Vite dev server
  --skip-health   Skip health checks
  --help          Show this help message

The script will:
1. Start the Go daemon (backend API)
2. Wait for daemon to be healthy
3. Start Vite dev server (frontend)
4. Optionally start Electron

All processes are managed together and will shutdown gracefully on Ctrl+C.
`)
        process.exit(0)
    }
  }
  
  return options
}

// Main orchestration
async function main() {
  const options = parseArgs()
  
  console.log(`${colors.bright}${colors.cyan}
╔══════════════════════════════════════╗
║    MCP Manager Development Mode      ║
╚══════════════════════════════════════╝${colors.reset}`)
  
  // Setup signal handlers
  process.on('SIGINT', () => shutdown('SIGINT'))
  process.on('SIGTERM', () => shutdown('SIGTERM'))
  
  try {
    // Start daemon
    if (!options.viteOnly) {
      startDaemon()
      
      // Wait for daemon health
      if (!options.skipHealth) {
        const daemonHealthy = await waitForHealth(CONFIG.daemon.health, 30000, 750)
        if (!daemonHealthy) {
          logError('system', 'Daemon failed to start. Check logs above.')
          await shutdown()
          return
        }
      }
    }
    
    // Start Vite
    if (!options.daemonOnly) {
      startVite()
      
      // Wait for Vite
      if (!options.skipHealth && options.electron) {
        const viteHealthy = await waitForHealth(CONFIG.vite.url, 30000, 750)
        if (!viteHealthy) {
          logError('system', 'Vite failed to start. Check logs above.')
          await shutdown()
          return
        }
      }
    }
    
    // Start Electron if requested
    if (options.electron && !options.daemonOnly) {
      // Give Vite a moment to stabilize
      await new Promise(r => setTimeout(r, CONFIG.electron.delay))
      startElectron()
    }
    
    // Show status
    console.log(`${colors.bright}${colors.green}
╔══════════════════════════════════════╗
║     All services started!            ║
╠══════════════════════════════════════╣
║  Backend:  http://127.0.0.1:38018   ║
║  Frontend: http://localhost:5173    ║
╚══════════════════════════════════════╝${colors.reset}

Press Ctrl+C to stop all services
`)
    
    // Keep process alive
    process.stdin.resume()
    
  } catch (error) {
    logError('system', error.message)
    await shutdown()
  }
}

// Run if executed directly
if (import.meta.url === `file://${process.argv[1]}`) {
  main()
}