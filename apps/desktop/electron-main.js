// @ts-check
const { app, BrowserWindow, Menu, shell, ipcMain } = require("electron");
const path = require("node:path");
const { spawn } = require("node:child_process");
const { existsSync } = require("node:fs");
const isDev = process.env.NODE_ENV === "development" || !app.isPackaged;

let daemon = null;
let mainWindow = null;

function resolveDaemonPath() {
  // In production, binary is under app resources bin/
  if (!isDev) {
    const prod = path.join(
      process.resourcesPath,
      "bin",
      process.platform === "darwin" ? "mcp-manager" : "mcp-manager.exe",
    );
    if (existsSync(prod)) return prod;
  }
  // In dev, use repository binary if built
  const dev = path.join(__dirname, "..", "..", "services", "manager", "bin", "mcp-manager");
  if (existsSync(dev)) return dev;
  // Try from project root
  const root = path.join(__dirname, "..", "..", "..", "..", "services", "manager", "bin", "mcp-manager");
  if (existsSync(root)) return root;
  return null;
}

function startDaemon() {
  const bin = resolveDaemonPath();
  if (!bin) {
    console.warn("Manager daemon binary not found. Build it with: npm run build:manager");
    // In dev mode, assume daemon is started separately
    if (isDev) {
      console.log("In development mode - assuming daemon is running separately");
      return;
    }
    return;
  }

  // Don't start daemon in development mode since it's already running
  if (isDev) {
    console.log("In development mode - daemon already running separately");
    return;
  }
  console.log("Starting daemon from:", bin);
  daemon = spawn(bin, [], {
    stdio: ["ignore", "pipe", "pipe"],
    env: { ...process.env, PORT: "7099" },
  });

  daemon.stdout?.on("data", (data) => {
    console.log(`[daemon]: ${data}`);
  });

  daemon.stderr?.on("data", (data) => {
    console.error(`[daemon-err]: ${data}`);
  });

  daemon.on("exit", (code) => {
    console.log("daemon exited", code);
    daemon = null;
  });
}

function stopDaemon() {
  if (daemon && !daemon.killed) {
    try {
      daemon.kill("SIGTERM");
    } catch {}
    daemon = null;
  }
}

async function createWindow() {
  mainWindow = new BrowserWindow({
    width: 1200,
    height: 800,
    minWidth: 900,
    minHeight: 600,
    backgroundColor: "#0b0b0b",
    webPreferences: {
      nodeIntegration: false,
      contextIsolation: true,
      preload: path.join(__dirname, "preload.js"),
    },
    title: "MCP Manager",
    titleBarStyle: process.platform === "darwin" ? "hiddenInset" : "default",
    show: false,
  });

  // Create application menu
  createMenu();

  // Load the app
  if (isDev) {
    // In dev, load from Vite dev server
    await mainWindow.loadURL("http://localhost:8099");
    mainWindow.webContents.openDevTools();
  } else {
    // In production, load from built files
    const indexPath = path.join(__dirname, "dist", "index.html");
    await mainWindow.loadFile(indexPath);
  }

  // Show window when ready
  mainWindow.once("ready-to-show", () => {
    mainWindow.show();
  });

  // Handle external links
  mainWindow.webContents.setWindowOpenHandler(({ url }) => {
    shell.openExternal(url);
    return { action: "deny" };
  });

  mainWindow.on("closed", () => {
    mainWindow = null;
  });
}

function createMenu() {
  const template = [
    ...(process.platform === "darwin"
      ? [
          {
            label: app.getName(),
            submenu: [
              { role: "about" },
              { type: "separator" },
              { role: "services", submenu: [] },
              { type: "separator" },
              { role: "hide" },
              { role: "hideOthers" },
              { role: "unhide" },
              { type: "separator" },
              { role: "quit" },
            ],
          },
        ]
      : []),
    {
      label: "File",
      submenu: [
        {
          label: "Settings",
          accelerator: process.platform === "darwin" ? "Cmd+," : "Ctrl+,",
          click: () => {
            mainWindow?.webContents.send("navigate", "/settings");
          },
        },
        { type: "separator" },
        process.platform === "darwin" ? { role: "close" } : { role: "quit" },
      ],
    },
    {
      label: "Edit",
      submenu: [
        { role: "undo" },
        { role: "redo" },
        { type: "separator" },
        { role: "cut" },
        { role: "copy" },
        { role: "paste" },
        { role: "selectAll" },
      ],
    },
    {
      label: "View",
      submenu: [
        { role: "reload" },
        { role: "forceReload" },
        { role: "toggleDevTools" },
        { type: "separator" },
        { role: "resetZoom" },
        { role: "zoomIn" },
        { role: "zoomOut" },
        { type: "separator" },
        { role: "togglefullscreen" },
      ],
    },
    {
      label: "Window",
      submenu: [
        { role: "minimize" },
        { role: "close" },
        ...(process.platform === "darwin" ? [{ type: "separator" }, { role: "front" }] : []),
      ],
    },
    {
      label: "Help",
      submenu: [
        {
          label: "Documentation",
          click: () => shell.openExternal("https://github.com/your-org/mcp-manager"),
        },
        {
          label: "Report Issue",
          click: () => shell.openExternal("https://github.com/your-org/mcp-manager/issues"),
        },
      ],
    },
  ];

  const menu = Menu.buildFromTemplate(template);
  Menu.setApplicationMenu(menu);
}

// IPC handlers
ipcMain.handle("app:version", () => app.getVersion());
ipcMain.handle("app:platform", () => process.platform);

app.on("ready", () => {
  startDaemon();
  createWindow();
});

app.on("window-all-closed", () => {
  if (process.platform !== "darwin") app.quit();
});

app.on("before-quit", () => {
  stopDaemon();
});

app.on("activate", () => {
  if (BrowserWindow.getAllWindows().length === 0) createWindow();
});
