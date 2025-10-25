export type ServerRow = {
  name: string;
  slug: string;
  status: "ready" | "degraded" | "down";
};

const BASE = "http://127.0.0.1:8080";

export async function fetchServers(): Promise<ServerRow[]> {
  const r = await fetch(`${BASE}/v1/servers`);
  if (!r.ok) throw new Error(`servers ${r.status}`);
  return r.json();
}

export async function serverAction(slug: string, action: "start" | "stop" | "restart"): Promise<void> {
  const r = await fetch(`${BASE}/v1/servers/${encodeURIComponent(slug)}/actions`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ action }),
  });
  if (!r.ok) throw new Error(`action ${r.status}`);
}



export async function installCancel(id: string): Promise<void> {
  const r = await fetch(`${BASE}/v1/install/cancel?id=${encodeURIComponent(id)}`, { method: "POST" });
  if (!r.ok) throw new Error("install cancel failed");
}

export type ClientDetection = { name: string; detected: boolean; path?: string };

export async function clientsDetect(): Promise<ClientDetection[]> {
  const r = await fetch(`${BASE}/v1/clients/detect`);
  if (!r.ok) throw new Error("clients detect failed");
  return r.json();
}

export async function clientsApply(client: string, config: any, path?: string): Promise<void> {
  const r = await fetch(`${BASE}/v1/clients/apply`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ client, config, path }),
  });
  if (!r.ok) throw new Error("clients apply failed");
}

export async function clientPreview(client: string): Promise<any> {
  const r = await fetch(`${BASE}/v1/clients/preview?client=${encodeURIComponent(client)}`);
  if (!r.ok) throw new Error("preview failed");
  return r.json();
}

export async function clientCurrent(client: string): Promise<any> {
  const r = await fetch(`${BASE}/v1/clients/current?client=${encodeURIComponent(client)}`);
  if (!r.ok) return {};
  try {
    return await r.json();
  } catch {
    return {};
  }
}

export async function fetchLogTail(slug: string, tail: number = 200): Promise<string> {
  const r = await fetch(`${BASE}/v1/logs/${encodeURIComponent(slug)}?tail=${tail}`);
  if (!r.ok) throw new Error("log tail failed");
  return r.text();
}

export async function getAutostart(): Promise<{ enabled: boolean; path: string; platform: string }> {
  const r = await fetch(`${BASE}/v1/settings/autostart`);
  if (!r.ok) throw new Error("autostart get failed");
  return r.json();
}

export async function setAutostart(enabled: boolean): Promise<{ enabled: boolean; path: string; platform: string }> {
  const r = await fetch(`${BASE}/v1/settings/autostart`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      enabled,
      platform: "darwin", // macOS
      method: "launchagent", // Use LaunchAgent for macOS
    }),
  });
  if (!r.ok) throw new Error("autostart set failed");
  return r.json();
}

// macOS specific autostart management
export async function setupMacOSAutostart(appPath: string, enabled: boolean): Promise<void> {
  const r = await fetch(`${BASE}/v1/system/macos/autostart`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      enabled,
      appPath,
      launchAgentPath: `${process.env.HOME}/Library/LaunchAgents/com.mcp-manager.plist`,
    }),
  });
  if (!r.ok) throw new Error("macOS autostart setup failed");
}

// Enhanced API endpoints

// Health monitoring
export type ProcessHealth = {
  name: string;
  status: "healthy" | "unhealthy" | "unknown";
  lastCheck: string;
  errorCount: number;
  uptime?: number;
  responseTime?: number;
  details?: Record<string, any>;
};

export async function fetchHealthSummary(): Promise<Record<string, ProcessHealth>> {
  const r = await fetch(`${BASE}/v1/health`);
  if (!r.ok) throw new Error("health summary failed");
  return r.json();
}

export async function fetchHealthDetail(slug: string): Promise<ProcessHealth> {
  const r = await fetch(`${BASE}/v1/health/${encodeURIComponent(slug)}`);
  if (!r.ok) throw new Error("health detail failed");
  return r.json();
}

// Server statistics and process info
export type ServerStats = {
  supervisor?: {
    totalProcesses: number;
    runningProcesses: number;
    uptimeSeconds: number;
  };
  health?: Record<string, ProcessHealth>;
  logs?: Record<string, any>;
};

export async function fetchServerStats(): Promise<ServerStats> {
  const r = await fetch(`${BASE}/v1/stats`);
  if (!r.ok) throw new Error("stats failed");
  return r.json();
}

export async function fetchServerInfo(slug: string): Promise<Record<string, any>> {
  const r = await fetch(`${BASE}/v1/servers/${encodeURIComponent(slug)}/info`);
  if (!r.ok) throw new Error("server info failed");
  return r.json();
}

export async function updateServerEnvVars(slug: string, envVars: Record<string, string>): Promise<void> {
  const r = await fetch(`${BASE}/v1/servers/${encodeURIComponent(slug)}/env`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ envVars }),
  });
  if (!r.ok) throw new Error("update env vars failed");
}

// Log streaming with Server-Sent Events
export type LogEntry = {
  timestamp: string;
  level: string;
  message: string;
  source?: string;
};

export function createLogStreamEventSource(slug: string, fromLine?: number): EventSource {
  const url = new URL(`${BASE}/v1/logs/stream/${encodeURIComponent(slug)}`);
  if (fromLine !== undefined) {
    url.searchParams.set("fromLine", fromLine.toString());
  }
  return new EventSource(url.toString());
}

// Enhanced settings
export type AppSettings = {
  theme: "light" | "dark" | "auto";
  autostart: boolean;
  logsCap: number;
  performance: {
    refreshInterval: number;
    maxLogLines: number;
  };
  storage: {
    used: number;
    available: number;
  };
};

export async function fetchSettings(): Promise<AppSettings> {
  const r = await fetch(`${BASE}/v1/settings`);
  if (!r.ok) throw new Error("settings fetch failed");
  return r.json();
}

export async function updateSettings(settings: Partial<AppSettings>): Promise<AppSettings> {
  const r = await fetch(`${BASE}/v1/settings`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(settings),
  });
  if (!r.ok) throw new Error("settings update failed");
  return r.json();
}

export async function resetSettings(): Promise<AppSettings> {
  const r = await fetch(`${BASE}/v1/settings/reset`, { method: "POST" });
  if (!r.ok) throw new Error("settings reset failed");
  return r.json();
}

export async function clearStorage(type: "logs" | "cache" | "all"): Promise<{ freed: number }> {
  const r = await fetch(`${BASE}/v1/storage/clear`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ type }),
  });
  if (!r.ok) throw new Error("storage clear failed");
  return r.json();
}

// Advanced installation
export type InstallProgress = {
  id: string;
  stage: "validating" | "downloading" | "installing" | "configuring" | "finalizing";
  progress: number;
  message: string;
  logs: string[];
  done: boolean;
  success?: boolean;
  error?: string;
};

export async function fetchInstallHistory(): Promise<
  Array<{
    id: string;
    type: string;
    uri: string;
    slug: string;
    timestamp: string;
    status: "completed" | "failed" | "cancelled";
    duration: number;
  }>
> {
  const r = await fetch(`${BASE}/v1/install/list`);
  if (!r.ok) throw new Error("install history failed");
  return r.json();
}


// Client configuration management
export async function openConfigFile(client: string): Promise<void> {
  // Web-compatible approach: open config in a new endpoint
  const r = await fetch(`${BASE}/v1/clients/${encodeURIComponent(client)}/open`, {
    method: "POST",
  });
  if (!r.ok) {
    // Fallback: Try to get the path and display it
    const paths = await fetch(`${BASE}/v1/clients/paths`)
      .then((r) => r.json())
      .catch(() => ({}));
    const path = paths[client];
    if (path) {
      // For macOS, use the open command via API
      const openR = await fetch(`${BASE}/v1/system/open`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ path, app: "default" }),
      });
      if (!openR.ok) {
        throw new Error(`Config file location: ${path}\nPlease open manually.`);
      }
    } else {
      throw new Error("Config file path not found");
    }
  }
}

export type ConfigDiff = {
  added: Record<string, any>;
  modified: Record<string, any>;
  removed: string[];
};

export async function getConfigDiff(client: string): Promise<ConfigDiff> {
  const [current, preview] = await Promise.all([clientCurrent(client), clientPreview(client)]);

  const diff: ConfigDiff = { added: {}, modified: {}, removed: [] };

  // Simple diff implementation - could be enhanced
  const currentKeys = Object.keys(current);
  const previewKeys = Object.keys(preview);

  for (const key of previewKeys) {
    if (!currentKeys.includes(key)) {
      diff.added[key] = preview[key];
    } else if (JSON.stringify(current[key]) !== JSON.stringify(preview[key])) {
      diff.modified[key] = preview[key];
    }
  }

  for (const key of currentKeys) {
    if (!previewKeys.includes(key)) {
      diff.removed.push(key);
    }
  }

  return diff;
}

// Enhanced error handling with retry mechanism
class APIError extends Error {
  constructor(
    message: string,
    public status?: number,
    public endpoint?: string,
  ) {
    super(message);
    this.name = "APIError";
  }
}

export async function fetchWithRetry(url: string, options?: RequestInit, retries: number = 3): Promise<Response> {
  let lastError: Error | null = null;

  for (let i = 0; i < retries; i++) {
    try {
      const response = await fetch(url, options);
      if (!response.ok) {
        throw new APIError(`HTTP ${response.status}`, response.status, url);
      }
      return response;
    } catch (error) {
      lastError = error as Error;
      if (i < retries - 1) {
        await new Promise((resolve) => setTimeout(resolve, 1000 * 2 ** i)); // Exponential backoff
      }
    }
  }

  throw lastError || new Error("Fetch failed after retries");
}

// External MCP Server Management API

export type ExternalServerProvider = {
  id: string;
  name: string;
  description: string;
  icon?: string;
  configFields: {
    key: string;
    label: string;
    type: "text" | "password" | "number" | "url" | "select";
    required: boolean;
    placeholder?: string;
    options?: { value: string; label: string }[];
    validation?: {
      pattern?: string;
      minLength?: number;
      maxLength?: number;
    };
  }[];
};

export type ExternalServerConfig = {
  slug: string;
  name: string;
  provider: string;
  displayName?: string;
  config: Record<string, any>;
  status: {
    state: "active" | "inactive" | "error" | "connecting" | "syncing";
    message?: string;
    lastChecked?: string;
    responseTime?: number;
  };
  lastSync?: string;
  autoStart?: boolean;
  apiEndpoint: string;
  authType: "api_key" | "oauth2" | "basic";
  createdAt?: string;
  updatedAt?: string;
};

export async function fetchExternalServers(): Promise<ExternalServerConfig[]> {
  const r = await fetch(`${BASE}/v1/external/servers`);
  if (!r.ok) throw new Error(`external servers fetch ${r.status}`);
  const data = await r.json();
  return data || [];
}

export async function createExternalServer(
  server: Omit<ExternalServerConfig, "id" | "status" | "createdAt" | "updatedAt">,
): Promise<ExternalServerConfig> {
  const r = await fetch(`${BASE}/v1/external/servers`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(server),
  });
  if (!r.ok) throw new Error(`external server create ${r.status}`);
  return r.json();
}

export async function updateExternalServer(
  id: string,
  server: Partial<ExternalServerConfig>,
): Promise<ExternalServerConfig> {
  const r = await fetch(`${BASE}/v1/external/servers/${encodeURIComponent(id)}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(server),
  });
  if (!r.ok) throw new Error(`external server update ${r.status}`);
  return r.json();
}

export async function deleteExternalServer(id: string): Promise<void> {
  const r = await fetch(`${BASE}/v1/external/servers/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
  if (!r.ok) throw new Error(`external server delete ${r.status}`);
}

export async function testExternalConnection(slug: string): Promise<{
  success: boolean;
  message?: string;
  responseTime?: number;
}> {
  const r = await fetch(`${BASE}/v1/external/servers/${encodeURIComponent(slug)}/test`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
  });
  if (!r.ok) throw new Error(`external server test ${r.status}`);
  return r.json();
}

export async function getProviderTemplates(): Promise<ExternalServerProvider[]> {
  const r = await fetch(`${BASE}/v1/external/providers`);
  if (!r.ok) throw new Error(`provider templates fetch ${r.status}`);
  const data = await r.json();
  return data || [];
}

// Credentials API helpers
export async function credsStore(provider: string, credentials: Record<string, string>) {
  const r = await fetch(`${BASE}/v1/credentials`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ provider, credentials }),
  });
  if (!r.ok) throw new Error(`credentials store ${r.status}`);
  return r.json();
}

export async function credsGetRequirements(provider: string) {
  const r = await fetch(`${BASE}/v1/credentials/${encodeURIComponent(provider)}`);
  if (!r.ok) throw new Error(`credentials requirements ${r.status}`);
  return r.json();
}

export async function credsUpdate(provider: string, credentials: Record<string, string>) {
  const r = await fetch(`${BASE}/v1/credentials/${encodeURIComponent(provider)}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ credentials }),
  });
  if (!r.ok) throw new Error(`credentials update ${r.status}`);
  return r.json();
}

export async function credsDelete(provider: string) {
  const r = await fetch(`${BASE}/v1/credentials/${encodeURIComponent(provider)}`, { method: "DELETE" });
  if (!r.ok) throw new Error(`credentials delete ${r.status}`);
  return r.json();
}

export async function credsValidate(provider: string, credentials: Record<string, string>) {
  const r = await fetch(`${BASE}/v1/credentials/validate`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ provider, credentials }),
  });
  if (!r.ok) throw new Error(`credentials validate ${r.status}`);
  return r.json();
}

// Advanced Installation API (New System)
export type AdvancedInstallRequest = {
  type: "git" | "npm" | "pip" | "docker-image" | "docker-compose";
  uri: string;
  slug: string;
  options?: Record<string, any>;
};

export type AdvancedInstallResponse = {
  jobId: string;
  status: string;
  message?: string;
};

export type AdvancedInstallStatus = {
  jobId: string;
  status: "pending" | "running" | "completed" | "failed" | "cancelled";
  stage: string;
  progress: number;
  message: string;
  logs?: string[];
  result?: {
    success: boolean;
    message?: string;
  };
};

export async function installStartAdvanced(request: AdvancedInstallRequest): Promise<AdvancedInstallResponse> {
  console.log('installStartAdvanced request:', request);
  console.log('Request URL:', `${BASE}/v1/install/start`);
  
  const r = await fetch(`${BASE}/v1/install/start`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(request),
  });
  
  console.log('Response status:', r.status);
  console.log('Response ok:', r.ok);
  
  if (!r.ok) {
    const errorText = await r.text();
    console.error('Response error text:', errorText);
    throw new Error(`advanced install start failed: ${r.status} - ${errorText}`);
  }
  
  const response = await r.json();
  console.log('Response data:', response);
  return response;
}

export async function installLogsAdvanced(jobId: string): Promise<AdvancedInstallStatus> {
  const r = await fetch(`${BASE}/v1/install/logs?id=${encodeURIComponent(jobId)}`);
  if (!r.ok) throw new Error(`advanced install logs failed: ${r.status}`);
  return r.json();
}

export async function finalizeInstallationAdvanced(jobId: string): Promise<void> {
  const r = await fetch(`${BASE}/v1/install/finalize?id=${encodeURIComponent(jobId)}`, {
    method: "POST",
  });
  if (!r.ok) throw new Error(`advanced install finalize failed: ${r.status}`);
}

// Admin Panel API Functions

export type InstalledServer = {
  slug: string;
  name: string;
  status: 'running' | 'stopped' | 'error';
  path: string;
  installDate?: string;
  version?: string;
};

export type MarketplaceItem = {
  slug: string;
  name: string;
  category: string;
  description: string;
  repoUrl?: string;
  docsUrl?: string;
  install?: { type: string; uri: string };
  remote?: { apiEndpoint: string; provider: string; authType?: string };
  configExample?: string;
  tags?: string[];
  createdAt?: string;
  updatedAt?: string;
};

export async function listInstalledServers(): Promise<InstalledServer[]> {
  const r = await fetch(`${BASE}/v1/admin/servers`);
  if (!r.ok) throw new Error(`list installed servers failed: ${r.status}`);
  return r.json();
}

export async function deleteInstalledServer(slug: string): Promise<void> {
  const r = await fetch(`${BASE}/v1/admin/servers/${encodeURIComponent(slug)}`, {
    method: "DELETE",
  });
  if (!r.ok) throw new Error(`delete installed server failed: ${r.status}`);
}

export async function listMarketplaceItems(): Promise<MarketplaceItem[]> {
  const r = await fetch(`${BASE}/v1/admin/marketplace`);
  if (!r.ok) throw new Error(`list marketplace items failed: ${r.status}`);
  return r.json();
}

export async function addMarketplaceItem(item: Omit<MarketplaceItem, 'createdAt' | 'updatedAt'>): Promise<MarketplaceItem> {
  const r = await fetch(`${BASE}/v1/admin/marketplace`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(item),
  });
  if (!r.ok) throw new Error(`add marketplace item failed: ${r.status}`);
  return r.json();
}

export async function updateMarketplaceItem(slug: string, item: Partial<MarketplaceItem>): Promise<MarketplaceItem> {
  const r = await fetch(`${BASE}/v1/admin/marketplace/${encodeURIComponent(slug)}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(item),
  });
  if (!r.ok) throw new Error(`update marketplace item failed: ${r.status}`);
  return r.json();
}

export async function deleteMarketplaceItem(slug: string): Promise<void> {
  const r = await fetch(`${BASE}/v1/admin/marketplace/${encodeURIComponent(slug)}`, {
    method: "DELETE",
  });
  if (!r.ok) throw new Error(`delete marketplace item failed: ${r.status}`);
}

export async function resolveAttentionItem(slug: string): Promise<{resolved: boolean; message: string; timestamp: string}> {
  const r = await fetch(`${BASE}/v1/admin/marketplace/${encodeURIComponent(slug)}/resolve-attention`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" }
  });
  if (!r.ok) throw new Error(`resolve attention failed: ${r.status}`);
  return r.json();
}
