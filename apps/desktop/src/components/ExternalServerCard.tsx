import React from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { 
  Circle, 
  MoreHorizontal, 
  Edit, 
  Trash2, 
  TestTube,
  Clock,
  Activity,
  AlertCircle,
  CheckCircle,
  XCircle,
  Loader2
} from "lucide-react";
import { testExternalConnection, type ExternalServerConfig } from "../api";

type ExternalServerCardProps = {
  server: ExternalServerConfig;
  onEdit: (server: ExternalServerConfig) => void;
  onDelete: (id: string) => void;
};

function StatusBadge({ status }: { status: ExternalServerConfig["status"] }) {
  // Handle case where status might be an object
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
          icon: Circle, 
          label: "Disconnected",
          className: "bg-gray-100 text-gray-800 border-gray-200"
        };
      case "error":
        return { 
          variant: "destructive" as const, 
          icon: XCircle, 
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

function HealthStatusBadge({ healthStatus }: { healthStatus?: ExternalServerConfig["healthStatus"] }) {
  if (!healthStatus) {
    return <span className="text-xs text-muted-foreground">No health data</span>;
  }

  const getConfig = (status: string) => {
    switch (status) {
      case "healthy":
        return { 
          icon: CheckCircle, 
          label: "Healthy",
          className: "text-green-600"
        };
      case "unhealthy":
        return { 
          icon: XCircle, 
          label: "Unhealthy",
          className: "text-red-600"
        };
      default:
        return { 
          icon: AlertCircle, 
          label: "Unknown",
          className: "text-gray-600"
        };
    }
  };

  const config = getConfig(healthStatus.status);
  const Icon = config.icon;

  return (
    <div className="flex items-center gap-1">
      <Icon className={`h-3 w-3 ${config.className}`} />
      <span className={`text-xs ${config.className}`}>{config.label}</span>
      {healthStatus.responseTime && (
        <span className="text-xs text-muted-foreground">({healthStatus.responseTime}ms)</span>
      )}
    </div>
  );
}

export function ExternalServerCard({ server, onEdit, onDelete }: ExternalServerCardProps) {
  const [testing, setTesting] = React.useState(false);
  const [testResult, setTestResult] = React.useState<{
    success: boolean;
    message?: string;
    responseTime?: number;
  } | null>(null);

  const handleTest = async () => {
    setTesting(true);
    setTestResult(null);
    
    try {
      const result = await testExternalConnection(server.slug);
      setTestResult(result);
      setTimeout(() => setTestResult(null), 5000); // Clear after 5 seconds
    } catch (err) {
      console.error("Test connection failed:", err);
      setTestResult({
        success: false,
        message: (err as Error).message,
      });
      setTimeout(() => setTestResult(null), 5000);
    } finally {
      setTesting(false);
    }
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

  return (
    <Card className="hover:shadow-md transition-shadow">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="space-y-1">
            <CardTitle className="text-lg">{server.name}</CardTitle>
            <div className="flex items-center gap-3 text-sm text-muted-foreground">
              <span className="flex items-center gap-1">
                <Circle className="h-3 w-3" />
                {server.displayName || server.provider}
              </span>
              {server.lastSync && (
                <span className="flex items-center gap-1">
                  <Clock className="h-3 w-3" />
                  Last sync: {formatDate(server.lastSync)}
                </span>
              )}
            </div>
          </div>
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" className="h-8 w-8 p-0">
                <span className="sr-only">Open menu</span>
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuItem onClick={handleTest} disabled={testing}>
                {testing ? (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <TestTube className="mr-2 h-4 w-4" />
                )}
                Test Connection
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => onEdit(server)}>
                <Edit className="mr-2 h-4 w-4" />
                Edit
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem 
                onClick={() => onDelete(server.slug)}
                className="text-destructive focus:text-destructive"
              >
                <Trash2 className="mr-2 h-4 w-4" />
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="text-sm text-muted-foreground">Status:</span>
            <StatusBadge status={server.status} />
          </div>
          <div className="flex items-center gap-2">
            <Activity className="h-4 w-4 text-muted-foreground" />
            {server.status?.responseTime && (
              <span className="text-xs text-muted-foreground">
                {server.status.responseTime}ms
              </span>
            )}
          </div>
        </div>

        {/* Test Result */}
        {testResult && (
          <div className={`p-2 rounded-md text-xs ${
            testResult.success 
              ? "bg-green-50 text-green-700 border border-green-200" 
              : "bg-red-50 text-red-700 border border-red-200"
          }`}>
            <div className="flex items-center gap-1">
              {testResult.success ? (
                <CheckCircle className="h-3 w-3" />
              ) : (
                <XCircle className="h-3 w-3" />
              )}
              <span className="font-medium">
                {testResult.success ? "Connection successful" : "Connection failed"}
              </span>
              {testResult.responseTime && (
                <span className="text-muted-foreground">({testResult.responseTime}ms)</span>
              )}
            </div>
            {testResult.message && (
              <p className="mt-1 text-xs">{testResult.message}</p>
            )}
          </div>
        )}

        {/* Error Message */}
        {server.status?.state === "error" && server.status?.message && (
          <div className="p-2 rounded-md bg-red-50 text-red-700 border border-red-200 text-xs">
            <div className="flex items-start gap-1">
              <AlertCircle className="h-3 w-3 mt-0.5 shrink-0" />
              <span>{server.status.message}</span>
            </div>
          </div>
        )}

        {server.status?.lastChecked && (
          <div className="flex items-center justify-between text-xs text-muted-foreground pt-2 border-t">
            <span>Last checked: {formatDate(server.status.lastChecked)}</span>
            {server.status.responseTime && (
              <span>Response: {server.status.responseTime}ms</span>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  );
}