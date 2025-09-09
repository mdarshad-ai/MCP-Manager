#!/usr/bin/env node
/**
 * Unified Build Script for MCP Manager
 * Builds both Go backend and TypeScript frontend as a single cohesive application
 */

import { spawn, spawnSync } from 'node:child_process'
import { existsSync, mkdirSync, rmSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import process from 'node:process'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const ROOT = path.resolve(__dirname, '..')
const BUILD_DIR = path.join(ROOT, 'build')
const DIST_DIR = path.join(ROOT, 'dist')

// Color codes for console output
const colors = {
  reset: '\x1b[0m',
  bright: '\x1b[1m',
  green: '\x1b[32m',
  yellow: '\x1b[33m',
  blue: '\x1b[34m',
  red: '\x1b[31m',
  cyan: '\x1b[36m'
}

function log(message, color = colors.blue) {
  console.log(`${color}[build]${colors.reset} ${message}`)
}

function logError(message) {
  console.error(`${colors.red}[build error]${colors.reset} ${message}`)
}

function logSuccess(message) {
  console.log(`${colors.green}[build success]${colors.reset} ${message}`)
}

// Execute command with proper error handling
async function exec(command, args = [], options = {}) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, {
      stdio: 'inherit',
      shell: false,
      cwd: ROOT,
      ...options
    })
    
    child.on('exit', (code) => {
      if (code === 0) {
        resolve()
      } else {
        reject(new Error(`${command} exited with code ${code}`))
      }
    })
    
    child.on('error', (err) => {
      reject(err)
    })
  })
}

// Check if a command exists
function commandExists(cmd) {
  const result = spawnSync(
    process.platform === 'win32' ? 'where' : 'which',
    [cmd],
    { stdio: 'ignore' }
  )
  return result.status === 0
}

// Clean build directories
function clean() {
  log('Cleaning previous build artifacts...', colors.yellow)
  
  // Clean build directory
  if (existsSync(BUILD_DIR)) {
    rmSync(BUILD_DIR, { recursive: true, force: true })
  }
  
  // Clean Go binary
  const goBinDir = path.join(ROOT, 'services', 'manager', 'bin')
  if (existsSync(goBinDir)) {
    rmSync(goBinDir, { recursive: true, force: true })
  }
  
  // Clean frontend dist
  const frontendDist = path.join(ROOT, 'apps', 'desktop', 'dist')
  if (existsSync(frontendDist)) {
    rmSync(frontendDist, { recursive: true, force: true })
  }
  
  log('Clean completed')
}

// Build Go backend
async function buildBackend() {
  log('Building Go backend...', colors.cyan)
  
  if (!commandExists('go')) {
    throw new Error('Go is not installed. Please install Go: https://golang.org/dl/')
  }
  
  const binDir = path.join(ROOT, 'services', 'manager', 'bin')
  mkdirSync(binDir, { recursive: true })
  
  const outputName = process.platform === 'win32' ? 'mcp-manager.exe' : 'mcp-manager'
  const outputPath = path.join(binDir, outputName)
  
  // Build with optimizations
  const env = {
    ...process.env,
    CGO_ENABLED: '0', // Disable CGO for static binary
    GOOS: process.platform === 'win32' ? 'windows' : process.platform,
    GOARCH: process.arch === 'x64' ? 'amd64' : process.arch
  }
  
  await exec('go', [
    'build',
    '-ldflags', '-s -w', // Strip debug info for smaller binary
    '-o', outputPath,
    './cmd/manager'
  ], {
    cwd: path.join(ROOT, 'services', 'manager'),
    env
  })
  
  logSuccess(`Go backend built: ${outputPath}`)
}

// Build TypeScript frontend
async function buildFrontend() {
  log('Building TypeScript frontend...', colors.cyan)
  
  // Install dependencies if needed
  const nodeModules = path.join(ROOT, 'node_modules')
  if (!existsSync(nodeModules)) {
    log('Installing dependencies...')
    await exec('npm', ['install'])
  }
  
  // Build frontend
  await exec('npm', ['run', 'build'], {
    cwd: path.join(ROOT, 'apps', 'desktop')
  })
  
  logSuccess('Frontend built successfully')
}

// Build Electron app
async function buildElectron(platform) {
  log(`Building Electron app for ${platform}...`, colors.cyan)
  
  const platformFlag = platform ? `--${platform}` : ''
  const args = ['run', platformFlag ? `dist:${platform}` : 'dist'].filter(Boolean)
  
  await exec('npm', args, {
    cwd: path.join(ROOT, 'apps', 'desktop')
  })
  
  logSuccess(`Electron app built for ${platform || 'current platform'}`)
}

// Copy build artifacts to unified output
async function collectArtifacts() {
  log('Collecting build artifacts...', colors.yellow)
  
  mkdirSync(BUILD_DIR, { recursive: true })
  
  // Copy Go binary
  const goBinary = process.platform === 'win32' ? 'mcp-manager.exe' : 'mcp-manager'
  const goBinPath = path.join(ROOT, 'services', 'manager', 'bin', goBinary)
  
  if (existsSync(goBinPath)) {
    await exec('cp', [goBinPath, BUILD_DIR])
  }
  
  // Frontend artifacts are handled by electron-builder
  
  logSuccess('Build artifacts collected')
}

// Main build orchestration
async function build(options = {}) {
  const startTime = Date.now()
  
  console.log(`${colors.bright}${colors.green}
╔══════════════════════════════════════╗
║     MCP Manager Unified Build        ║
╚══════════════════════════════════════╝${colors.reset}`)
  
  try {
    // Parse build options
    const {
      clean: shouldClean = true,
      backend = true,
      frontend = true,
      electron = false,
      platform = null,
      production = false
    } = options
    
    // Set environment
    if (production) {
      process.env.NODE_ENV = 'production'
    }
    
    // Step 1: Clean
    if (shouldClean) {
      clean()
    }
    
    // Step 2: Build backend
    if (backend) {
      await buildBackend()
    }
    
    // Step 3: Build frontend
    if (frontend) {
      await buildFrontend()
    }
    
    // Step 4: Build Electron (if requested)
    if (electron) {
      await buildElectron(platform)
    }
    
    // Step 5: Collect artifacts
    await collectArtifacts()
    
    const elapsed = ((Date.now() - startTime) / 1000).toFixed(2)
    console.log(`${colors.bright}${colors.green}
╔══════════════════════════════════════╗
║   Build completed in ${elapsed}s         ║
╚══════════════════════════════════════╝${colors.reset}`)
    
  } catch (error) {
    logError(error.message)
    process.exit(1)
  }
}

// Parse command line arguments
function parseArgs() {
  const args = process.argv.slice(2)
  const options = {}
  
  for (let i = 0; i < args.length; i++) {
    switch (args[i]) {
      case '--no-clean':
        options.clean = false
        break
      case '--backend-only':
        options.frontend = false
        options.electron = false
        break
      case '--frontend-only':
        options.backend = false
        break
      case '--electron':
        options.electron = true
        break
      case '--mac':
        options.electron = true
        options.platform = 'mac'
        break
      case '--win':
        options.electron = true
        options.platform = 'win'
        break
      case '--linux':
        options.electron = true
        options.platform = 'linux'
        break
      case '--production':
        options.production = true
        break
      case '--help':
        console.log(`
MCP Manager Unified Build Script

Usage: node scripts/build.mjs [options]

Options:
  --no-clean        Skip cleaning previous build artifacts
  --backend-only    Build only the Go backend
  --frontend-only   Build only the TypeScript frontend
  --electron        Build Electron app for current platform
  --mac            Build Electron app for macOS
  --win            Build Electron app for Windows
  --linux          Build Electron app for Linux
  --production     Build in production mode
  --help           Show this help message

Examples:
  npm run build                # Build everything
  npm run build:backend        # Build only backend
  npm run build:frontend       # Build only frontend
  npm run build:electron       # Build with Electron packaging
`)
        process.exit(0)
    }
  }
  
  return options
}

// Run the build
if (import.meta.url === `file://${process.argv[1]}`) {
  const options = parseArgs()
  build(options)
}