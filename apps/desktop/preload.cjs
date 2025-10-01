const { contextBridge, ipcRenderer } = require("electron");

// Expose protected methods that allow the renderer process to use
// the ipcRenderer without exposing the entire object
contextBridge.exposeInMainWorld("electronAPI", {
  // App information
  getVersion: () => ipcRenderer.invoke("app:version"),
  getPlatform: () => ipcRenderer.invoke("app:platform"),

  // Navigation
  onNavigate: (callback) => {
    ipcRenderer.on("navigate", (event, path) => callback(path));
  },

  // External links
  openExternal: (url) => shell.openExternal(url),

  // Daemon management
  restartDaemon: () => ipcRenderer.invoke("daemon:restart"),

  // File system (if needed in future)
  selectDirectory: () => ipcRenderer.invoke("dialog:openDirectory"),
  selectFile: () => ipcRenderer.invoke("dialog:openFile"),
});
