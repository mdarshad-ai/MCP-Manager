import React from "react";
import { useTheme } from "@/components/theme-provider";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Progress } from "@/components/ui/progress";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { type AppSettings, fetchSettings, resetSettings, updateSettings, clearStorage } from "../api";
import { Trash2 } from "lucide-react";

export function Settings() {
  const { theme, setTheme } = useTheme();
  const [settings, setSettings] = React.useState<AppSettings | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [saving, setSaving] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [unsavedChanges, setUnsavedChanges] = React.useState(false);

  const loadSettings = React.useCallback(async () => {
    setLoading(true);
    try {
      const data = await fetchSettings();
      setSettings(data);
      setError(null);
    } catch (err) {
      setError(`Failed to load settings: ${(err as Error).message}`);
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    loadSettings();
  }, [loadSettings]);

  const updateSetting = <K extends keyof AppSettings>(key: K, value: AppSettings[K]) => {
    if (settings) {
      setSettings({ ...settings, [key]: value });
      setUnsavedChanges(true);
    }
  };

  const updateNestedSetting = <K extends keyof AppSettings, NK extends keyof AppSettings[K]>(
    parentKey: K,
    nestedKey: NK,
    value: AppSettings[K][NK],
  ) => {
    if (settings && typeof settings[parentKey] === "object" && settings[parentKey] !== null) {
      const newParent = { ...(settings[parentKey] as any), [nestedKey]: value };
      setSettings({ ...settings, [parentKey]: newParent });
      setUnsavedChanges(true);
    }
  };

  const saveSettings = async () => {
    if (!settings) return;
    setSaving(true);
    try {
      const updated = await updateSettings(settings);
      setSettings(updated);
      setUnsavedChanges(false);
      setError(null);
    } catch (err) {
      setError(`Failed to save settings: ${(err as Error).message}`);
    } finally {
      setSaving(false);
    }
  };

  const resetToDefaults = async () => {
    if (!confirm("Reset all settings to defaults? This cannot be undone.")) return;
    setSaving(true);
    try {
      const defaultSettings = await resetSettings();
      setSettings(defaultSettings);
      setUnsavedChanges(false);
      setError(null);
    } catch (err) {
      setError(`Failed to reset settings: ${(err as Error).message}`);
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="p-6">
        <Card>
          <CardHeader>
            <CardTitle>Settings</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-muted-foreground">Loading settings...</div>
          </CardContent>
        </Card>
      </div>
    );
  }

  if (!settings) {
    return (
      <div className="p-6">
        <Card>
          <CardHeader>
            <CardTitle>Settings</CardTitle>
          </CardHeader>
          <CardContent>
            <Alert variant="destructive">
              <AlertDescription>Failed to load settings</AlertDescription>
            </Alert>
            <Button className="mt-4" onClick={loadSettings}>
              Retry
            </Button>
          </CardContent>
        </Card>
      </div>
    );
  }

  const formatBytes = (bytes: number): string => {
    const units = ["B", "KB", "MB", "GB"];
    let size = bytes;
    let unitIndex = 0;

    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex++;
    }

    return `${size.toFixed(1)} ${units[unitIndex]}`;
  };

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Settings</h1>
          <p className="text-muted-foreground">Configure application preferences</p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={resetToDefaults} disabled={saving}>
            Reset to Defaults
          </Button>
          <Button
            onClick={saveSettings}
            disabled={saving || !unsavedChanges}
            variant={unsavedChanges ? "default" : "outline"}
          >
            {saving ? "Saving..." : "Save Changes"}
          </Button>
        </div>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {unsavedChanges && (
        <Alert>
          <AlertDescription>You have unsaved changes. Click "Save Changes" to apply them.</AlertDescription>
        </Alert>
      )}

      <div className="space-y-8">
        {/* General Settings */}
        <Card>
          <CardHeader>
            <CardTitle>General</CardTitle>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="flex items-center justify-between">
              <div className="space-y-0.5">
                <Label className="text-base">Autostart Manager</Label>
                <p className="text-sm text-muted-foreground">Start MCP Manager automatically at login</p>
              </div>
              <Switch checked={settings?.autostart || false} onCheckedChange={(checked) => updateSetting("autostart", checked)} />
            </div>

            <Separator />

            <div className="space-y-2">
              <Label>Theme</Label>
              <Select value={theme} onValueChange={setTheme}>
                <SelectTrigger className="w-48">
                  <SelectValue placeholder="Select theme" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="light">Light</SelectItem>
                  <SelectItem value="dark">Dark</SelectItem>
                  <SelectItem value="system">System</SelectItem>
                </SelectContent>
              </Select>
              <p className="text-sm text-muted-foreground">Choose your preferred color scheme</p>
            </div>
          </CardContent>
        </Card>

        {/* Performance Settings */}
        <Card>
          <CardHeader>
            <CardTitle>Performance</CardTitle>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="space-y-2">
              <Label htmlFor="refresh-interval">Refresh Interval (seconds)</Label>
              <Input
                id="refresh-interval"
                type="number"
                value={settings?.performance?.refreshInterval ? settings.performance.refreshInterval / 1000 : 5}
                onChange={(e) => updateNestedSetting("performance", "refreshInterval", Number(e.target.value) * 1000)}
                min="1"
                max="60"
                className="w-32"
              />
              <p className="text-sm text-muted-foreground">How often to refresh server status and metrics</p>
            </div>

            <div className="space-y-2">
              <Label>Maximum Log Lines</Label>
              <Select
                value={settings?.performance?.maxLogLines?.toString() || "1000"}
                onValueChange={(value) => updateNestedSetting("performance", "maxLogLines", Number(value))}
              >
                <SelectTrigger className="w-48">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="100">100 lines</SelectItem>
                  <SelectItem value="500">500 lines</SelectItem>
                  <SelectItem value="1000">1,000 lines</SelectItem>
                  <SelectItem value="5000">5,000 lines</SelectItem>
                  <SelectItem value="10000">10,000 lines</SelectItem>
                </SelectContent>
              </Select>
              <p className="text-sm text-muted-foreground">Maximum number of log lines to keep in memory per server</p>
            </div>
          </CardContent>
        </Card>

        {/* Storage Settings */}
        <Card>
          <CardHeader>
            <CardTitle>Storage</CardTitle>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="space-y-2">
              <Label htmlFor="logs-cap">Logs Storage Limit (MB)</Label>
              <Input
                id="logs-cap"
                type="number"
                value={settings?.logsCap || 100}
                onChange={(e) => updateSetting("logsCap", Number(e.target.value))}
                min="10"
                max="10000"
                className="w-32"
              />
              <p className="text-sm text-muted-foreground">Maximum disk space to use for storing logs</p>
            </div>

            {settings.storage && (
              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm">Storage Usage</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-3">
                    <div className="flex justify-between text-sm">
                      <span>Used:</span>
                      <span>{formatBytes(settings?.storage?.used || 0)}</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span>Available:</span>
                      <span>{formatBytes(settings?.storage?.available || 0)}</span>
                    </div>
                    <Progress
                      value={Math.min(
                        100,
                        settings?.storage ? (settings.storage.used / (settings.storage.used + settings.storage.available)) * 100 : 0,
                      )}
                      className="w-full"
                    />
                  </div>
                  <div className="mt-4 flex gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={async () => {
                        try {
                          const result = await clearStorage("logs");
                          alert(`Cleared ${formatBytes(result.freed)} of log files`);
                          await loadSettings();
                        } catch (err) {
                          alert(`Failed to clear logs: ${(err as Error).message}`);
                        }
                      }}
                    >
                      <Trash2 className="h-4 w-4 mr-2" />
                      Clear Logs
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={async () => {
                        try {
                          const result = await clearStorage("cache");
                          alert(`Cleared ${formatBytes(result.freed)} of cache`);
                          await loadSettings();
                        } catch (err) {
                          alert(`Failed to clear cache: ${(err as Error).message}`);
                        }
                      }}
                    >
                      <Trash2 className="h-4 w-4 mr-2" />
                      Clear Cache
                    </Button>
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={async () => {
                        if (confirm("Are you sure you want to clear all storage? This action cannot be undone.")) {
                          try {
                            const result = await clearStorage("all");
                            alert(`Cleared ${formatBytes(result.freed)} of storage`);
                            await loadSettings();
                          } catch (err) {
                            alert(`Failed to clear storage: ${(err as Error).message}`);
                          }
                        }
                      }}
                    >
                      <Trash2 className="h-4 w-4 mr-2" />
                      Clear All
                    </Button>
                  </div>
                </CardContent>
              </Card>
            )}
          </CardContent>
        </Card>

        {/* Advanced Settings */}
        <Card>
          <CardHeader>
            <CardTitle>Advanced</CardTitle>
          </CardHeader>
          <CardContent>
            <Alert variant="destructive">
              <AlertDescription>
                <div className="space-y-4">
                  <div>
                    <h4 className="font-medium">Danger Zone</h4>
                    <p className="text-sm mt-1">These actions cannot be undone. Use with caution.</p>
                  </div>
                  <div className="flex gap-2">
                    <Button variant="destructive" size="sm" onClick={resetToDefaults} disabled={saving}>
                      Reset All Settings
                    </Button>
                    <Button
                      variant="destructive"
                      size="sm"
                      onClick={() => {
                        if (confirm("Clear all logs? This cannot be undone.")) {
                          // Implementation would go here
                          alert("Log clearing not yet implemented");
                        }
                      }}
                    >
                      Clear All Logs
                    </Button>
                  </div>
                </div>
              </AlertDescription>
            </Alert>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
