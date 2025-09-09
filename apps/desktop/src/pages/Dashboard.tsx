import React from "react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Filter, Search, Plus, ShoppingBag } from "lucide-react";
import {
  fetchHealthSummary,
  fetchServerStats,
  fetchServers,
  fetchExternalServers,
  type ExternalServerConfig,
  type ProcessHealth,
  type ServerRow,
  type ServerStats,
  serverAction,
} from "../api";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { ExternalServerForm } from "@/components/ExternalServerForm";

type Row = {
  slug: string;
  name: string;
  status: "ready" | "degraded" | "down";
  cpu?: number;
  ramMB?: number;
  uptime?: string;
  restarts?: number;
  lastPingMs?: number;
  health?: ProcessHealth;
};

function StatusBadge({ status }: { status: Row["status"] | ProcessHealth["status"] }) {
  const getLabel = (status: string) => {
    switch (status) {
      case "ready":
      case "healthy":
        return "Ready";
      case "degraded":
      case "warning":
        return "Degraded";
      case "down":
      case "unhealthy":
        return "Down";
      case "unknown":
        return "Unknown";
      default:
        return status;
    }
  };

  const getVariant = (status: string) => {
    switch (status) {
      case "ready":
      case "healthy":
        return "default";
      case "degraded":
      case "warning":
        return "secondary";
      case "down":
      case "unhealthy":
        return "destructive";
      case "unknown":
        return "outline";
      default:
        return "outline";
    }
  };

  return (
    <Badge variant={getVariant(status) as any} className="text-xs">
      {getLabel(status)}
    </Badge>
  );
}

export function Dashboard({ rows: initialRows }: { rows?: Row[] }) {
  const [rows, setRows] = React.useState<Row[]>(initialRows ?? []);
  const [search, setSearch] = React.useState("");
  const [filters, setFilters] = React.useState<{ ready: boolean; degraded: boolean; down: boolean; unknown: boolean }>({
    ready: true,
    degraded: true,
    down: true,
    unknown: true,
  });
  const [stats, setStats] = React.useState<ServerStats | null>(null);
  const [healthData, setHealthData] = React.useState<Record<string, ProcessHealth>>({});
  const [refreshInterval, setRefreshInterval] = React.useState(5000);
  const [error, setError] = React.useState<string | null>(null);

  const loadData = React.useCallback(async () => {
    try {
      const [servers, serverStats, health] = await Promise.all([
        fetchServers(),
        fetchServerStats().catch(() => null),
        fetchHealthSummary().catch(() => ({})),
      ]);

      setStats(serverStats);
      setHealthData(health);

      const enrichedRows = servers.map((s: any) => ({
        slug: s.slug,
        name: s.name,
        status: s.status,
        cpu: s.cpu,
        ramMB: s.ramMB,
        uptime: s.uptime ? formatUptime(s.uptime) : undefined,
        restarts: s.restarts,
        lastPingMs: s.lastPingMs,
        health: health[s.slug],
      }));

      setRows(enrichedRows);
      setError(null);
    } catch (err) {
      console.error("Dashboard data fetch error:", err);
      setError((err as Error).message);
    }
  }, []);

  React.useEffect(() => {
    loadData();
    const id = setInterval(loadData, refreshInterval);
    return () => clearInterval(id);
  }, [loadData, refreshInterval]);

  const onAction = async (slug: string, action: "start" | "stop" | "restart") => {
    if (!confirm(`${action.charAt(0).toUpperCase() + action.slice(1)} this server?`)) return;

    try {
      await serverAction(slug, action);
      // Refresh data immediately after action
      setTimeout(loadData, 500);
    } catch (err) {
      console.error(`Failed to ${action} server:`, err);
      alert(`Failed to ${action} server: ${(err as Error).message}`);
    }
  };

  const filteredRows = React.useMemo(() => {
    return rows.filter((r) => {
      const statusMatch =
        filters[r.status as keyof typeof filters] || (r.health && filters[r.health.status as keyof typeof filters]);
      const searchMatch =
        !search ||
        r.name.toLowerCase().includes(search.toLowerCase()) ||
        r.slug.toLowerCase().includes(search.toLowerCase());
      return statusMatch && searchMatch;
    });
  }, [rows, filters, search]);

  return (
    <div className="p-6 space-y-6">
      {error && (
        <Alert variant="destructive">
          <AlertDescription>Error loading data: {error}</AlertDescription>
        </Alert>
      )}

      {/* Statistics Overview */}
      {stats && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">Total Processes</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats.supervisor?.totalProcesses ?? 0}</div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">Running</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold text-green-600">{stats.supervisor?.runningProcesses ?? 0}</div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">System Uptime</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {stats.supervisor?.uptimeSeconds ? formatUptime(stats.supervisor.uptimeSeconds) : "—"}
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium text-muted-foreground">Health Checks</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{Object.keys(healthData).length}</div>
            </CardContent>
          </Card>
        </div>
      )}

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Server Dashboard</CardTitle>
            <div className="flex gap-3 items-center">
              <Select 
                value={Object.entries(filters).filter(([_, v]) => v).map(([k]) => k).join(',')}
                onValueChange={(value) => {
                  const selectedFilters = value.split(',');
                  setFilters({
                    ready: selectedFilters.includes('ready'),
                    degraded: selectedFilters.includes('degraded'),
                    down: selectedFilters.includes('down'),
                    unknown: selectedFilters.includes('unknown'),
                  });
                }}
              >
                <SelectTrigger className="w-44">
                  <div className="flex items-center gap-2">
                    <Filter className="h-4 w-4" />
                    <SelectValue placeholder="Filter by status">
                      {Object.entries(filters).filter(([_, v]) => v).length === 4 
                        ? "All statuses" 
                        : Object.entries(filters).filter(([_, v]) => v).length === 0
                        ? "None selected"
                        : `${Object.entries(filters).filter(([_, v]) => v).length} selected`}
                    </SelectValue>
                  </div>
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="ready,degraded,down,unknown">All statuses</SelectItem>
                  <SelectItem value="ready">Ready only</SelectItem>
                  <SelectItem value="degraded">Degraded only</SelectItem>
                  <SelectItem value="down">Down only</SelectItem>
                  <SelectItem value="unknown">Unknown only</SelectItem>
                  <SelectItem value="ready,degraded">Active (Ready + Degraded)</SelectItem>
                  <SelectItem value="down,unknown">Inactive (Down + Unknown)</SelectItem>
                </SelectContent>
              </Select>
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  className="w-64 pl-9"
                  placeholder="Search servers..."
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                />
              </div>
              <Select value={refreshInterval.toString()} onValueChange={(value) => setRefreshInterval(Number(value))}>
                <SelectTrigger className="w-20">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="2000">2s</SelectItem>
                  <SelectItem value="5000">5s</SelectItem>
                  <SelectItem value="10000">10s</SelectItem>
                  <SelectItem value="30000">30s</SelectItem>
                </SelectContent>
              </Select>
              <a href="#/market" className="text-sm inline-flex items-center gap-2 text-muted-foreground hover:text-foreground">
                <ShoppingBag className="h-4 w-4" /> Marketplace
              </a>
              <a href="#/install" className="text-sm inline-flex items-center gap-2 text-muted-foreground hover:text-foreground">
                <Plus className="h-4 w-4" /> Install
              </a>
              <Button variant="outline" onClick={loadData}>
                Refresh
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Status</TableHead>
                <TableHead>Name</TableHead>
                <TableHead>Health</TableHead>
                <TableHead>CPU</TableHead>
                <TableHead>RAM</TableHead>
                <TableHead>Uptime</TableHead>
                <TableHead>Restarts</TableHead>
                <TableHead>Response</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredRows.map((r) => (
                <TableRow key={r.slug}>
                  <TableCell>
                    <StatusBadge status={r.status} />
                  </TableCell>
                  <TableCell className="font-medium">{r.name}</TableCell>
                  <TableCell>
                    {r.health ? (
                      <div className="flex items-center gap-2">
                        <StatusBadge status={r.health.status} />
                        {r.health.errorCount > 0 && (
                          <span className="text-xs text-destructive">({r.health.errorCount} errors)</span>
                        )}
                      </div>
                    ) : (
                      <span className="text-muted-foreground text-xs">No monitoring</span>
                    )}
                  </TableCell>
                  <TableCell>{r.cpu ?? 0}%</TableCell>
                  <TableCell>{r.ramMB ? `${r.ramMB} MB` : "0 MB"}</TableCell>
                  <TableCell>{r.uptime ?? "—"}</TableCell>
                  <TableCell>{r.restarts ?? 0}</TableCell>
                  <TableCell>
                    {r.health?.responseTime ? `${r.health.responseTime}ms` : r.lastPingMs ? `${r.lastPingMs}ms` : "—"}
                  </TableCell>
                  <TableCell>
                    <div className="flex gap-2">
                      <Button variant="link" size="sm" asChild className="h-auto p-0">
                        <a href={`#/logs/${r.slug}`}>Logs</a>
                      </Button>
                      <Button variant="link" size="sm" asChild className="h-auto p-0">
                        <a href={`#/server/${r.slug}`}>Details</a>
                      </Button>
                      <Button
                        variant="link"
                        size="sm"
                        className="h-auto p-0 text-green-600"
                        onClick={() => onAction(r.slug, "start")}
                        disabled={r.status === "ready"}
                      >
                        Start
                      </Button>
                      <Button
                        variant="link"
                        size="sm"
                        className="h-auto p-0 text-orange-600"
                        onClick={() => onAction(r.slug, "restart")}
                      >
                        Restart
                      </Button>
                      <Button
                        variant="link"
                        size="sm"
                        className="h-auto p-0 text-destructive"
                        onClick={() => onAction(r.slug, "stop")}
                        disabled={r.status === "down"}
                      >
                        Stop
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>

          {filteredRows.length === 0 && (
            <div className="text-center py-8 text-muted-foreground">
              {search ? `No servers found matching "${search}"` : "No servers match the current filters"}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}

function formatUptime(sec: number): string {
  const h = Math.floor(sec / 3600);
  const m = Math.floor((sec % 3600) / 60);
  const s = Math.floor(sec % 60);
  if (h > 0) return `${pad(h)}:${pad(m)}:${pad(s)}`;
  return `${pad(m)}:${pad(s)}`;
}

function pad(n: number): string {
  return n < 10 ? `0${n}` : String(n);
}
