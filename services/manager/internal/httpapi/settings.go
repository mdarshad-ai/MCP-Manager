package httpapi

import (
    "encoding/json"
    "net/http"
    "os"

    "mcp/manager/internal/autostart"
    "mcp/manager/internal/settings"
)

func (s *Server) handleAutostartGet(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet { w.WriteHeader(http.StatusMethodNotAllowed); return }
    p, err := autostart.PlistPath()
    if err != nil { w.WriteHeader(http.StatusInternalServerError); return }
    enabled := fileExists(p)
    writeJSON(w, map[string]any{"enabled": enabled, "path": p})
}

func (s *Server) handleAutostartSet(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost { w.WriteHeader(http.StatusMethodNotAllowed); return }
    var body struct{ Enabled bool `json:"enabled"` }
    if err := json.NewDecoder(r.Body).Decode(&body); err != nil { w.WriteHeader(http.StatusBadRequest); return }
    if body.Enabled {
        exe, err := os.Executable(); if err != nil { w.WriteHeader(http.StatusInternalServerError); return }
        if err := autostart.Install(exe); err != nil { w.WriteHeader(http.StatusInternalServerError); return }
    } else {
        if err := autostart.Remove(); err != nil { w.WriteHeader(http.StatusInternalServerError); return }
    }
    s.handleAutostartGet(w, r)
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

// handleSettingsGet returns the current application settings
func (s *Server) handleSettingsGet(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }

    settings, err := settings.GetCached()
    if err != nil {
        http.Error(w, "failed to load settings", http.StatusInternalServerError)
        return
    }

    writeJSON(w, settings)
}

// handleSettingsUpdate updates application settings
func (s *Server) handleSettingsUpdate(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPut && r.Method != http.MethodPost {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }

    var newSettings settings.Settings
    if err := json.NewDecoder(r.Body).Decode(&newSettings); err != nil {
        http.Error(w, "invalid JSON", http.StatusBadRequest)
        return
    }

    if err := settings.UpdateCached(&newSettings); err != nil {
        http.Error(w, "failed to save settings", http.StatusInternalServerError)
        return
    }

    writeJSON(w, newSettings)
}

// handleSettingsPartial updates specific settings sections
func (s *Server) handleSettingsPartial(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPatch {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }

    currentSettings, err := settings.GetCached()
    if err != nil {
        http.Error(w, "failed to load current settings", http.StatusInternalServerError)
        return
    }

    // Parse partial update
    var patch map[string]json.RawMessage
    if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
        http.Error(w, "invalid JSON", http.StatusBadRequest)
        return
    }

    // Apply patches to current settings
    updated := *currentSettings // copy current settings

    for key, value := range patch {
        switch key {
        case "autostart":
            var autostartSettings settings.AutostartSettings
            if err := json.Unmarshal(value, &autostartSettings); err != nil {
                http.Error(w, "invalid autostart settings", http.StatusBadRequest)
                return
            }
            updated.Autostart = autostartSettings

        case "theme":
            var themeSettings settings.ThemeSettings
            if err := json.Unmarshal(value, &themeSettings); err != nil {
                http.Error(w, "invalid theme settings", http.StatusBadRequest)
                return
            }
            updated.Theme = themeSettings

        case "logs":
            var logSettings settings.LogSettings
            if err := json.Unmarshal(value, &logSettings); err != nil {
                http.Error(w, "invalid log settings", http.StatusBadRequest)
                return
            }
            updated.Logs = logSettings

        case "manager":
            var managerSettings settings.ManagerSettings
            if err := json.Unmarshal(value, &managerSettings); err != nil {
                http.Error(w, "invalid manager settings", http.StatusBadRequest)
                return
            }
            updated.Manager = managerSettings

        default:
            http.Error(w, "unknown settings section: "+key, http.StatusBadRequest)
            return
        }
    }

    // Save updated settings
    if err := settings.UpdateCached(&updated); err != nil {
        http.Error(w, "failed to save settings", http.StatusInternalServerError)
        return
    }

    writeJSON(w, updated)
}

// handleSettingsReset resets settings to defaults
func (s *Server) handleSettingsReset(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        w.WriteHeader(http.StatusMethodNotAllowed)
        return
    }

    defaultSettings := settings.NewDefault()
    if err := settings.UpdateCached(defaultSettings); err != nil {
        http.Error(w, "failed to reset settings", http.StatusInternalServerError)
        return
    }

    writeJSON(w, defaultSettings)
}
