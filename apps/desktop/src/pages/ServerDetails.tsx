import {
  Activity,
  AlertCircle,
  Copy,
  ExternalLink,
  Loader2,
  Play,
  Plus,
  RotateCcw,
  Settings,
  Square,
  Terminal,
  Trash2,
  Users,
} from "lucide-react";
import React from "react";
import {
  clientsApply,
  fetchHealthDetail,
  fetchLogTail,
  fetchServerInfo,
  fetchExternalServers,
  type ProcessHealth,
  serverAction,
  updateServerEnvVars,
} from "../api";
import { ExternalServerDetails } from "../components/ExternalServerDetails";
import { Alert, AlertDescription } from "../components/ui/alert";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { ScrollArea } from "../components/ui/scroll-area";
import { Separator } from "../components/ui/separator";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";

type ServerDetailsProps = {
  slug: string;
};

type ServerInfo = {
  name: string;
  slug: string;
  status: string;
  command?: string;
  args?: string[];
  transport?: string;
  env?: Record<string, string>;
  pid?: number;
  uptime?: number;
  cpu?: number;
  memory?: number;
  restarts?: number;
};

export function ServerDetails({ slug }: ServerDetailsProps) {
  const [serverType, setServerType] = React.useState<"local" | "remote" | null>(null);
  const [serverInfo, setServerInfo] = React.useState<ServerInfo | null>(null);
  const [health, setHealth] = React.useState<ProcessHealth | null>(null);
  const [envVars, setEnvVars] = React.useState<Array<{ key: string; value: string }>>([]);
  const [newEnvKey, setNewEnvKey] = React.useState("");
  const [newEnvValue, setNewEnvValue] = React.useState("");
  const [logs, setLogs] = React.useState("");
  const [logQuery, setLogQuery] = React.useState("");
  const [followLogs, setFollowLogs] = React.useState(false);
  const [busy, setBusy] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  React.useEffect(() => {
    if (!slug) return;

    // First, determine if this is a local or remote server
    const determineServerType = async () => {
      try {
        // Check remote servers first to avoid unnecessary local API calls
        const remoteServers = await fetchExternalServers();
        const foundRemote = remoteServers.find(s => s.slug === slug);
        if (foundRemote) {
          setServerType("remote");
          return;
        }
        
        // If not remote, try as local server
        const [info, healthData, logData] = await Promise.all([
          fetchServerInfo(slug),
          fetchHealthDetail(slug).catch(() => null),
          fetchLogTail(slug, 100).catch(() => ""),
        ]);
        
        setServerType("local");
        setServerInfo(info as ServerInfo);
        setHealth(healthData);
        setLogs(logData);

        if (info.env) {
          setEnvVars(Object.entries(info.env).map(([key, value]) => ({ key, value })));
        }
      } catch (err) {
        setError(`Server "${slug}" not found`);
      }
    };

    determineServerType();
  }, [slug]);

  React.useEffect(() => {
    if (!followLogs || !slug) return;

    const interval = setInterval(async () => {
      try {
        const newLogs = await fetchLogTail(slug, 100);
        setLogs(newLogs);
      } catch (err) {
        console.error("Log fetch error:", err);
      }
    }, 2000);

    return () => clearInterval(interval);
  }, [followLogs, slug]);

  const handleServerAction = async (action: "start" | "stop" | "restart") => {
    setBusy(true);
    try {
      await serverAction(slug, action);
      // Refresh server info after action
      const info = await fetchServerInfo(slug);
      setServerInfo(info as ServerInfo);
    } catch (err) {
      setError(`Failed to ${action} server: ${(err as Error).message}`);
    } finally {
      setBusy(false);
    }
  };

  const addEnvVar = () => {
    if (!newEnvKey || !newEnvValue) return;
    setEnvVars((prev) => [...prev, { key: newEnvKey, value: newEnvValue }]);
    setNewEnvKey("");
    setNewEnvValue("");
  };

  const removeEnvVar = (index: number) => {
    setEnvVars((prev) => prev.filter((_, i) => i !== index));
  };

  const copyEnvPreview = () => {
    const envText = envVars.map(({ key, value }) => `${key}=${value}`).join("\n");
    navigator.clipboard.writeText(envText);
  };

  const applyToClient = async (clientName: string) => {
    try {
      await clientsApply(clientName, { servers: { [slug]: serverInfo } });
      alert(`Successfully applied to ${clientName}`);
    } catch (err) {
      alert(`Failed to apply to ${clientName}: ${(err as Error).message}`);
    }
  };

  const filteredLogs = React.useMemo(() => {
    if (!logQuery) return logs;
    return logs
      .split("\n")
      .filter((line) => line.toLowerCase().includes(logQuery.toLowerCase()))
      .join("\n");
  }, [logs, logQuery]);

  // Handle remote servers
  if (serverType === "remote") {
    return <ExternalServerDetails slug={slug} />;
  }

  if (error) {
    return (
      <div className="p-6">
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>
            <div className="font-medium">Error loading server details</div>
            <div className="mt-1">{error}</div>
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  if (!serverType || (serverType === "local" && !serverInfo)) {
    return (
      <div className="p-6 flex items-center justify-center">
        <div className="flex items-center gap-2">
          <Loader2 className="h-4 w-4 animate-spin" />
          <span>Loading server details...</span>
        </div>
      </div>
    );
  }

  const StatusBadge = ({ status }: { status: string }) => {
    const getStatusVariant = (status: string) => {
      switch (status?.toLowerCase() || '') {
        case "ready":
        case "running":
        case "healthy":
          return "default";
        case "degraded":
        case "warning":
          return "secondary";
        case "down":
        case "stopped":
        case "error":
        case "unhealthy":
          return "destructive";
        default:
          return "outline";
      }
    };
    return <Badge variant={getStatusVariant(status) as any}>{status}</Badge>;
  };

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">{serverInfo.name}</h1>
          <div className="flex items-center gap-2 mt-2">
            <StatusBadge status={serverInfo.status} />
            {health && <StatusBadge status={health.status} />}
            {serverInfo.pid && <Badge variant="outline">PID: {serverInfo.pid}</Badge>}
            {serverInfo.uptime && <Badge variant="outline">Uptime: {formatUptime(serverInfo.uptime)}</Badge>}
          </div>
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleServerAction("start")}
            disabled={busy || serverInfo.status === "running"}
          >
            {busy ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <Play className="h-4 w-4 mr-2" />}
            Start
          </Button>
          <Button variant="outline" size="sm" onClick={() => handleServerAction("restart")} disabled={busy}>
            <RotateCcw className="h-4 w-4 mr-2" />
            Restart
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => handleServerAction("stop")}
            disabled={busy || serverInfo.status === "stopped"}
          >
            <Square className="h-4 w-4 mr-2" />
            Stop
          </Button>
        </div>
      </div>

      {/* Performance Metrics */}
      {(serverInfo.cpu !== undefined || serverInfo.memory !== undefined) && (
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-base flex items-center">
              <Activity className="h-4 w-4 mr-2" />
              Performance
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-3 gap-4">
              <Card>
                <CardContent className="p-4">
                  <div className="text-muted-foreground text-sm">CPU Usage</div>
                  <div className="text-2xl font-semibold">{serverInfo.cpu ?? 0}%</div>
                </CardContent>
              </Card>
              <Card>
                <CardContent className="p-4">
                  <div className="text-muted-foreground text-sm">Memory</div>
                  <div className="text-2xl font-semibold">{serverInfo.memory ?? 0} MB</div>
                </CardContent>
              </Card>
              <Card>
                <CardContent className="p-4">
                  <div className="text-muted-foreground text-sm">Restarts</div>
                  <div className="text-2xl font-semibold">{serverInfo.restarts ?? 0}</div>
                </CardContent>
              </Card>
            </div>
          </CardContent>
        </Card>
      )}

      <Tabs defaultValue="configuration" className="space-y-4">
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="configuration">
            <Settings className="h-4 w-4 mr-2" />
            Configuration
          </TabsTrigger>
          <TabsTrigger value="environment">
            <Terminal className="h-4 w-4 mr-2" />
            Environment
          </TabsTrigger>
          <TabsTrigger value="clients">
            <Users className="h-4 w-4 mr-2" />
            Clients
          </TabsTrigger>
          <TabsTrigger value="logs">
            <Terminal className="h-4 w-4 mr-2" />
            Logs
          </TabsTrigger>
        </TabsList>

        <TabsContent value="configuration">
          <Card>
            <CardHeader>
              <CardTitle>Server Configuration</CardTitle>
              <CardDescription>Server execution details and runtime configuration</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>Transport</Label>
                  <Input value={serverInfo.transport ?? "stdio"} readOnly className="bg-muted" />
                </div>
                <div className="space-y-2">
                  <Label>Command</Label>
                  <Input value={serverInfo.command ?? "node"} readOnly className="bg-muted" />
                </div>
              </div>
              {serverInfo.args && serverInfo.args.length > 0 && (
                <div className="space-y-2">
                  <Label>Arguments</Label>
                  <Input value={serverInfo.args.join(" ")} readOnly className="bg-muted" />
                </div>
              )}

              {health && health.details && (
                <div className="pt-4">
                  <h4 className="font-medium mb-3">Health Details</h4>
                  <Card>
                    <CardContent className="p-4">
                      <div className="space-y-2 text-sm">
                        <div className="flex items-center gap-2">
                          <span className="font-medium">Status:</span>
                          <StatusBadge status={health.status} />
                        </div>
                        <div>
                          <span className="font-medium">Last Check:</span> {new Date(health.lastCheck).toLocaleString()}
                        </div>
                        <div>
                          <span className="font-medium">Error Count:</span> {health.errorCount}
                        </div>
                        {health.responseTime && (
                          <div>
                            <span className="font-medium">Response Time:</span> {health.responseTime}ms
                          </div>
                        )}
                      </div>
                    </CardContent>
                  </Card>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="environment">
          <Card>
            <CardHeader>
              <CardTitle>Environment Variables</CardTitle>
              <CardDescription>Configure environment variables for the server</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-3">
                <div className="grid grid-cols-3 gap-3 text-sm font-medium text-muted-foreground">
                  <span>KEY</span>
                  <span>VALUE</span>
                  <span>ACTIONS</span>
                </div>
                <div className="space-y-2">
                  {envVars.map(({ key, value }, index) => (
                    <div key={index} className="grid grid-cols-3 gap-3 items-center">
                      <Input
                        value={key}
                        onChange={(e) => {
                          const newVars = [...envVars];
                          newVars[index] = { ...newVars[index], key: e.target.value };
                          setEnvVars(newVars);
                        }}
                      />
                      <Input
                        value={value}
                        type={
                          key.toLowerCase().includes("password") || key.toLowerCase().includes("secret")
                            ? "password"
                            : "text"
                        }
                        onChange={(e) => {
                          const newVars = [...envVars];
                          newVars[index] = { ...newVars[index], value: e.target.value };
                          setEnvVars(newVars);
                        }}
                      />
                      <Button variant="destructive" size="sm" onClick={() => removeEnvVar(index)}>
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  ))}
                  <div className="grid grid-cols-3 gap-3 items-center">
                    <Input placeholder="KEY" value={newEnvKey} onChange={(e) => setNewEnvKey(e.target.value)} />
                    <Input placeholder="VALUE" value={newEnvValue} onChange={(e) => setNewEnvValue(e.target.value)} />
                    <Button variant="outline" onClick={addEnvVar} disabled={!newEnvKey || !newEnvValue}>
                      <Plus className="h-4 w-4 mr-2" />
                      Add
                    </Button>
                  </div>
                </div>
              </div>
              <div className="pt-4 flex gap-2">
                <Button variant="outline" onClick={copyEnvPreview}>
                  <Copy className="h-4 w-4 mr-2" />
                  Copy .env preview
                </Button>
                <Button 
                  onClick={async () => {
                    try {
                      const envObj = envVars.reduce((acc, { key, value }) => {
                        if (key) acc[key] = value;
                        return acc;
                      }, {} as Record<string, string>);
                      
                      // Get slug from URL
                      const slug = window.location.hash.split('/')[2];
                      if (slug) {
                        await updateServerEnvVars(slug, envObj);
                        alert('Environment variables saved successfully');
                      }
                    } catch (err) {
                      alert(`Failed to save environment variables: ${(err as Error).message}`);
                    }
                  }}
                >
                  Save Environment Variables
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="clients">
          <Card>
            <CardHeader>
              <CardTitle>Client Integration</CardTitle>
              <CardDescription>Apply server configuration to different MCP clients</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex gap-3 flex-wrap">
                <Button variant="outline" onClick={() => applyToClient("Claude Desktop")}>
                  <ExternalLink className="h-4 w-4 mr-2" />
                  Apply to Claude
                </Button>
                <Button variant="outline" onClick={() => applyToClient("Cursor (Global)")}>
                  <ExternalLink className="h-4 w-4 mr-2" />
                  Apply to Cursor
                </Button>
                <Button
                  variant="outline"
                  onClick={() => {
                    const snippet = `"${slug}": {\n  "command": "${serverInfo.command}",\n  "args": ${JSON.stringify(serverInfo.args)}\n}`;
                    navigator.clipboard.writeText(snippet);
                    alert("Snippet copied to clipboard");
                  }}
                >
                  <Copy className="h-4 w-4 mr-2" />
                  Copy Continue Snippet
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="logs">
          <Card>
            <CardHeader>
              <CardTitle>Recent Logs</CardTitle>
              <CardDescription>View and search server logs in real-time</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex gap-3 items-center">
                <Input
                  placeholder="Search logs"
                  value={logQuery}
                  onChange={(e) => setLogQuery(e.target.value)}
                  className="flex-1"
                />
                <Button variant={followLogs ? "default" : "outline"} onClick={() => setFollowLogs(!followLogs)}>
                  {followLogs ? "Pause" : "Follow"}
                </Button>
                <Button variant="outline" asChild>
                  <a href={`#/logs/${slug}`}>
                    <ExternalLink className="h-4 w-4 mr-2" />
                    Full Logs
                  </a>
                </Button>
              </div>
              <ScrollArea className="h-64 w-full border rounded-lg">
                <div className="p-3 text-xs bg-black text-green-400 font-mono min-h-full whitespace-pre-wrap">
                  {filteredLogs || "No logs available"}
                </div>
              </ScrollArea>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}

function formatUptime(seconds: number): string {
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = Math.floor(seconds % 60);

  if (hours > 0) {
    return `${hours}h ${minutes}m ${secs}s`;
  } else if (minutes > 0) {
    return `${minutes}m ${secs}s`;
  } else {
    return `${secs}s`;
  }
}
