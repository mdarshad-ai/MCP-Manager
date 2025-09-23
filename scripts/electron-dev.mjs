#!/usr/bin/env node
import { spawn } from "node:child_process";
import { existsSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT = path.resolve(__dirname, "..");

// Wait for Vite to be ready
async function waitForVite(url = "http://localhost:8099", timeout = 30000) {
  const start = Date.now();
  while (Date.now() - start < timeout) {
    try {
      const res = await fetch(url);
      if (res.ok) return true;
    } catch {}
    await new Promise((r) => setTimeout(r, 500));
  }
  return false;
}

async function main() {
  console.log("[electron-dev] Waiting for Vite dev server...");

  const viteReady = await waitForVite();
  if (!viteReady) {
    console.error('[electron-dev] Vite dev server not ready. Make sure to run "npm run dev" first.');
    process.exit(1);
  }

  console.log("[electron-dev] Starting Electron...");

  // Start Electron
  const electron = spawn("npx", ["electron", "apps/desktop"], {
    stdio: "inherit",
    shell: true,
    cwd: ROOT,
    env: {
      ...process.env,
      NODE_ENV: "development",
    },
  });

  electron.on("exit", (code) => {
    console.log(`[electron-dev] Electron exited with code ${code}`);
    process.exit(code || 0);
  });
}

main().catch(console.error);
