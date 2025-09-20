package autostart

import (
    "fmt"
    "os"
    "path/filepath"
    "os/exec"
)

// PlistPath returns the user LaunchAgents plist path.
func PlistPath() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil { return "", err }
    return filepath.Join(home, "Library", "LaunchAgents", "com.mcp.manager.plist"), nil
}

// PlistContents returns a minimal launchd plist for the manager daemon.
func PlistContents(execPath string) string {
    return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key><string>com.mcp.manager</string>
  <key>ProgramArguments</key>
  <array>
    <string>%s</string>
  </array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>%s</string>
  <key>StandardErrorPath</key><string>%s</string>
</dict>
</plist>`, execPath, logPath("stdout"), logPath("stderr"))
}

func logPath(name string) string {
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".mcp", "logs", "manager-"+name+".log")
}

// Install writes the plist and attempts to load it via launchctl.
func Install(execPath string) error {
    p, err := PlistPath(); if err != nil { return err }
    if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil { return err }
    if err := os.WriteFile(p, []byte(PlistContents(execPath)), 0o644); err != nil { return err }
    _ = exec.Command("launchctl", "unload", p).Run()
    _ = exec.Command("launchctl", "load", "-w", p).Run()
    return nil
}

// Remove unloads and deletes the plist.
func Remove() error {
    p, err := PlistPath(); if err != nil { return err }
    _ = exec.Command("launchctl", "unload", p).Run()
    _ = os.Remove(p)
    return nil
}
