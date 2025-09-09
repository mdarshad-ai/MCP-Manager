import React from "react";
import { 
  Globe, 
  TestTube, 
  Settings, 
  CheckCircle, 
  XCircle, 
  AlertCircle, 
  Clock,
  Activity,
  Loader2,
  Edit,
  Trash2,
  Plus
} from "lucide-react";
import { Alert, AlertDescription } from "./ui/alert";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "./ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "./ui/tabs";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "./ui/dialog";
import {
  fetchExternalServers,
  testExternalConnection,
  deleteExternalServer,
  type ExternalServerConfig,
} from "../api";
import { ExternalServerForm } from "./ExternalServerForm";

type ExternalServerDetailsProps = {
  slug: string;
};

function StatusBadge({ status }: { status: ExternalServerConfig["status"] }) {
  const statusString = status?.state || 'inactive';
  const getConfig = (status: string) => {
    switch (status) {
      case "active":
        return { 
          variant: "default" as const, 
          icon: CheckCircle, 
          label: "Connected",
          className: "bg-green-100 text-green-800 border-green-200"
        };
      case "connecting":
      case "syncing":
        return { 
          variant: "secondary" as const, 
          icon: Loader2, 
          label: "Connecting",
          className: "bg-blue-100 text-blue-800 border-blue-200"
        };
      case "inactive":
        return { 
          variant: "outline" as const, 
          icon: XCircle, 
          label: "Disconnected",
          className: "bg-gray-100 text-gray-800 border-gray-200"
        };
      case "error":
        return { 
          variant: "destructive" as const, 
          icon: AlertCircle, 
          label: "Error",
          className: "bg-red-100 text-red-800 border-red-200"
        };
      default:
        return { 
          variant: "outline" as const, 
          icon: AlertCircle, 
          label: status,
          className: "bg-gray-100 text-gray-800 border-gray-200"
        };
    }
  };

  const config = getConfig(statusString);
  const Icon = config.icon;

  return (
    <Badge variant={config.variant} className={`text-xs ${config.className}`}>
      <Icon className={`h-3 w-3 mr-1 ${statusString === "connecting" ? "animate-spin" : ""}`} />
      {config.label}
    </Badge>
  );
}

export function ExternalServerDetails({ slug }: ExternalServerDetailsProps) {
  const [server, setServer] = React.useState<ExternalServerConfig | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [testing, setTesting] = React.useState(false);
  const [testResult, setTestResult] = React.useState<{
    success: boolean;
    message?: string;
    responseTime?: number;
  } | null>(null);
  const [showEditDialog, setShowEditDialog] = React.useState(false);
  const [deleting, setDeleting] = React.useState(false);

  const loadServerDetails = React.useCallback(async () => {
    try {
      const servers = await fetchExternalServers();
      const foundServer = servers.find(s => s.slug === slug);
      if (foundServer) {
        setServer(foundServer);
      } else {
        setError("Remote server not found");
      }
    } catch (err) {
      console.error("Failed to load server details:", err);
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, [slug]);

  React.useEffect(() => {
    loadServerDetails();
    // Refresh every 30 seconds
    const interval = setInterval(loadServerDetails, 30000);
    return () => clearInterval(interval);
  }, [loadServerDetails]);

  const handleTest = async () => {
    if (!server) return;
    
    setTesting(true);
    setTestResult(null);
    
    try {
      const result = await testExternalConnection(server.slug);
      setTestResult(result);
      // Refresh server status after test
      loadServerDetails();
    } catch (err) {
      console.error("Test connection failed:", err);
      setTestResult({
        success: false,
        message: (err as Error).message,
      });
    } finally {
      setTesting(false);
    }
  };

  const handleDelete = async () => {
    if (!server || !confirm(`Are you sure you want to delete "${server.name}"?`)) return;

    setDeleting(true);
    try {
      await deleteExternalServer(server.slug);
      // Navigate back to dashboard
      window.location.hash = "#/dashboard";
    } catch (err) {
      console.error("Failed to delete server:", err);
      alert(`Failed to delete server: ${(err as Error).message}`);
    } finally {
      setDeleting(false);
    }
  };

  const handleFormSuccess = (updatedServer: ExternalServerConfig) => {
    setServer(updatedServer);
    setShowEditDialog(false);
    loadServerDetails(); // Refresh to get latest status
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMins / 60);
    const diffDays = Math.floor(diffHours / 24);

    if (diffMins < 1) return "Just now";
    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    if (diffDays === 1) return "Yesterday";
    if (diffDays < 7) return `${diffDays}d ago`;
    
    return date.toLocaleDateString();
  };

  if (loading) {
    return (
      <div className="p-6 flex items-center justify-center">
        <div className="flex items-center gap-2">
          <Loader2 className="h-4 w-4 animate-spin" />
          <span>Loading remote server details...</span>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6">
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>
            <div className="font-medium">Error loading remote server</div>
            <div className="mt-1">{error}</div>
          </AlertDescription>
        </Alert>
      </div>
    );
  }

  if (!server) {
    return (
      <div className="p-6">
        <Alert>
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>Remote server not found</AlertDescription>
        </Alert>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <div className="flex items-center gap-3 mb-2">
            <Globe className="h-6 w-6 text-muted-foreground" />
            <h1 className="text-2xl font-semibold tracking-tight">{server.name}</h1>
          </div>
          <div className="flex items-center gap-3 text-sm text-muted-foreground">
            <span>{server.displayName || server.provider}</span>
            <Badge variant="outline">{server.provider}</Badge>
            <StatusBadge status={server.status} />
            {server.status?.responseTime && (
              <span className="flex items-center gap-1">
                <Activity className="h-3 w-3" />
                {server.status.responseTime}ms
              </span>
            )}
          </div>
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={handleTest}
            disabled={testing}
          >
            {testing ? (
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
            ) : (
              <TestTube className="h-4 w-4 mr-2" />
            )}
            Test Connection
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setShowEditDialog(true)}
          >
            <Edit className="h-4 w-4 mr-2" />
            Edit
          </Button>
          <Button
            variant="destructive"
            size="sm"
            onClick={handleDelete}
            disabled={deleting}
          >
            {deleting ? (
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
            ) : (
              <Trash2 className="h-4 w-4 mr-2" />
            )}
            Delete
          </Button>
        </div>
      </div>

      {/* Test Result */}
      {testResult && (
        <Alert variant={testResult.success ? "default" : "destructive"}>
          {testResult.success ? (
            <CheckCircle className="h-4 w-4" />
          ) : (
            <XCircle className="h-4 w-4" />
          )}
          <AlertDescription>
            <div className="flex items-center justify-between">
              <div>
                <span className="font-medium">
                  {testResult.success ? "Connection successful!" : "Connection failed"}
                </span>
                {testResult.responseTime && (
                  <span className="ml-2 text-xs">({testResult.responseTime}ms)</span>
                )}
              </div>
            </div>
            {testResult.message && (
              <p className="mt-1 text-sm">{testResult.message}</p>
            )}
          </AlertDescription>
        </Alert>
      )}

      <Tabs defaultValue="overview" className="space-y-4">
        <TabsList className="grid w-full grid-cols-3">
          <TabsTrigger value="overview">
            <Globe className="h-4 w-4 mr-2" />
            Overview
          </TabsTrigger>
          <TabsTrigger value="configuration">
            <Settings className="h-4 w-4 mr-2" />
            Configuration
          </TabsTrigger>
          <TabsTrigger value="health">
            <Activity className="h-4 w-4 mr-2" />
            Health
          </TabsTrigger>
        </TabsList>

        <TabsContent value="overview">
          <div className="grid gap-6">
            <Card>
              <CardHeader>
                <CardTitle>Server Information</CardTitle>
                <CardDescription>Basic details about this remote MCP server</CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">Provider</label>
                    <p className="text-sm">{server.provider}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">Auth Type</label>
                    <p className="text-sm capitalize">{server.authType?.replace('_', ' ')}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">API Endpoint</label>
                    <p className="text-sm font-mono">{server.apiEndpoint}</p>
                  </div>
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">Auto Start</label>
                    <p className="text-sm">{server.autoStart ? "Enabled" : "Disabled"}</p>
                  </div>
                </div>
                
                {server.createdAt && (
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">Created</label>
                    <p className="text-sm">{formatDate(server.createdAt)}</p>
                  </div>
                )}
                
                {server.lastSync && (
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">Last Sync</label>
                    <p className="text-sm flex items-center gap-1">
                      <Clock className="h-3 w-3" />
                      {formatDate(server.lastSync)}
                    </p>
                  </div>
                )}
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="configuration">
          <Card>
            <CardHeader>
              <CardTitle>Configuration Settings</CardTitle>
              <CardDescription>Current configuration for this remote server</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {server.config && Object.keys(server.config).length > 0 ? (
                <div className="space-y-3">
                  {Object.entries(server.config).map(([key, value]) => (
                    <div key={key} className="grid grid-cols-3 gap-4 items-center">
                      <label className="text-sm font-medium text-muted-foreground">{key}</label>
                      <div className="col-span-2">
                        <code className="text-xs bg-muted px-2 py-1 rounded">{String(value)}</code>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">No configuration settings available</p>
              )}
              
              <div className="pt-4 border-t">
                <Button 
                  variant="outline" 
                  onClick={() => setShowEditDialog(true)}
                >
                  <Edit className="h-4 w-4 mr-2" />
                  Edit Configuration
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="health">
          <Card>
            <CardHeader>
              <CardTitle>Health Status</CardTitle>
              <CardDescription>Connection health and monitoring information</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="text-sm font-medium text-muted-foreground">Current Status</label>
                  <div className="mt-1">
                    <StatusBadge status={server.status} />
                  </div>
                </div>
                {server.status?.responseTime && (
                  <div>
                    <label className="text-sm font-medium text-muted-foreground">Response Time</label>
                    <p className="text-sm">{server.status.responseTime}ms</p>
                  </div>
                )}
              </div>
              
              {server.status?.lastChecked && (
                <div>
                  <label className="text-sm font-medium text-muted-foreground">Last Health Check</label>
                  <p className="text-sm">{formatDate(server.status.lastChecked)}</p>
                </div>
              )}
              
              {server.status?.message && (
                <div>
                  <label className="text-sm font-medium text-muted-foreground">Status Message</label>
                  <p className="text-sm">{server.status.message}</p>
                </div>
              )}
              
              <div className="pt-4 border-t">
                <Button 
                  variant="outline" 
                  onClick={handleTest}
                  disabled={testing}
                >
                  {testing ? (
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  ) : (
                    <TestTube className="h-4 w-4 mr-2" />
                  )}
                  Run Health Check
                </Button>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Edit Dialog */}
      {showEditDialog && (
        <Dialog open={showEditDialog} onOpenChange={setShowEditDialog}>
          <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
            <DialogHeader>
              <DialogTitle>Edit Remote MCP Server</DialogTitle>
              <DialogDescription>
                Update the configuration for "{server.name}".
              </DialogDescription>
            </DialogHeader>
            <ExternalServerForm
              server={server}
              onSuccess={handleFormSuccess}
              onCancel={() => setShowEditDialog(false)}
            />
          </DialogContent>
        </Dialog>
      )}
    </div>
  );
}