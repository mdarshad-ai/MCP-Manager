import React from "react";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { Filter, Search, Plus, RefreshCw, ShoppingBag } from "lucide-react";
import {
  fetchExternalServers,
  deleteExternalServer,
  type ExternalServerConfig,
} from "../api";
import { ExternalServerCard } from "../components/ExternalServerCard";
import { ExternalServerForm } from "../components/ExternalServerForm";

export function ExternalServers() {
  const [servers, setServers] = React.useState<ExternalServerConfig[]>([]);
  const [search, setSearch] = React.useState("");
  const [statusFilter, setStatusFilter] = React.useState<string>("all");
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [showAddDialog, setShowAddDialog] = React.useState(false);
  const [editingServer, setEditingServer] = React.useState<ExternalServerConfig | null>(null);
  const [refreshing, setRefreshing] = React.useState(false);

  const loadServers = React.useCallback(async () => {
    try {
      setError(null);
      const data = await fetchExternalServers();
      setServers(data);
    } catch (err) {
      console.error("Failed to fetch remote servers:", err);
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }, []);

  React.useEffect(() => {
    loadServers();
    // Refresh every 30 seconds
    const interval = setInterval(loadServers, 30000);
    return () => clearInterval(interval);
  }, [loadServers]);

  const handleRefresh = async () => {
    setRefreshing(true);
    await loadServers();
    setRefreshing(false);
  };

  const handleDelete = async (id: string) => {
    if (!confirm("Are you sure you want to delete this remote server?")) return;

    try {
      await deleteExternalServer(id);
      setServers(prev => prev.filter(s => s.id !== id));
    } catch (err) {
      console.error("Failed to delete server:", err);
      alert(`Failed to delete server: ${(err as Error).message}`);
    }
  };

  const handleEdit = (server: ExternalServerConfig) => {
    setEditingServer(server);
  };

  const handleFormSuccess = (server: ExternalServerConfig) => {
    if (editingServer) {
      // Update existing server
      setServers(prev => prev.map(s => s.id === server.id ? server : s));
      setEditingServer(null);
    } else {
      // Add new server
      setServers(prev => [...prev, server]);
      setShowAddDialog(false);
    }
  };

  const filteredServers = React.useMemo(() => {
    return servers.filter((server) => {
      const serverStatus = server.status?.state || 'inactive';
      const statusMatch = statusFilter === "all" || 
        (statusFilter === "connected" && serverStatus === "active") ||
        (statusFilter === "disconnected" && serverStatus === "inactive") ||
        (statusFilter === "error" && serverStatus === "error");
      const searchMatch = 
        !search ||
        server.name.toLowerCase().includes(search.toLowerCase()) ||
        (server.displayName || server.provider).toLowerCase().includes(search.toLowerCase());
      return statusMatch && searchMatch;
    });
  }, [servers, search, statusFilter]);

  const getStatusCounts = React.useMemo(() => {
    return servers.reduce(
      (counts, server) => {
        const status = server.status?.state || 'inactive';
        const mappedStatus = status === 'active' ? 'connected' : 
                            status === 'inactive' ? 'disconnected' : status;
        counts[mappedStatus] = (counts[mappedStatus] || 0) + 1;
        return counts;
      },
      {} as Record<string, number>
    );
  }, [servers]);

  if (loading) {
    return (
      <div className="p-6 flex items-center justify-center">
        <div className="flex items-center gap-2">
          <RefreshCw className="h-4 w-4 animate-spin" />
          Loading remote servers...
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-6">
      {error && (
        <Alert variant="destructive">
          <AlertDescription>Failed to load remote servers: {error}</AlertDescription>
        </Alert>
      )}

      {/* Statistics Overview */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Total Servers</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{servers.length}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Connected</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-600">{getStatusCounts.connected || 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Disconnected</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-orange-600">{getStatusCounts.disconnected || 0}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">Errors</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-red-600">{getStatusCounts.error || 0}</div>
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle>Remote MCP Servers</CardTitle>
            <div className="flex gap-3 items-center">
              <Select value={statusFilter} onValueChange={setStatusFilter}>
                <SelectTrigger className="w-44">
                  <div className="flex items-center gap-2">
                    <Filter className="h-4 w-4" />
                    <SelectValue placeholder="Filter by status" />
                  </div>
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All statuses</SelectItem>
                  <SelectItem value="connected">Connected</SelectItem>
                  <SelectItem value="connecting">Connecting</SelectItem>
                  <SelectItem value="disconnected">Disconnected</SelectItem>
                  <SelectItem value="error">Error</SelectItem>
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
              <a href="#/market" className="text-sm inline-flex items-center gap-2 text-muted-foreground hover:text-foreground">
                <ShoppingBag className="h-4 w-4" /> Marketplace
              </a>
              <Button variant="outline" onClick={handleRefresh} disabled={refreshing}>
                <RefreshCw className={`h-4 w-4 mr-2 ${refreshing ? 'animate-spin' : ''}`} />
                Refresh
              </Button>
              <Dialog open={showAddDialog} onOpenChange={setShowAddDialog}>
                <DialogTrigger asChild>
                  <Button>
                    <Plus className="h-4 w-4 mr-2" />
                    Add Server
                  </Button>
                </DialogTrigger>
                <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
                  <DialogHeader>
                    <DialogTitle>Add Remote MCP Server</DialogTitle>
                    <DialogDescription>
                      Configure a connection to a remote MCP server.
                    </DialogDescription>
                  </DialogHeader>
                  <ExternalServerForm
                    onSuccess={handleFormSuccess}
                    onCancel={() => setShowAddDialog(false)}
                  />
                </DialogContent>
              </Dialog>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {filteredServers.length === 0 ? (
            <div className="text-center py-12 text-muted-foreground">
              {search || statusFilter !== "all" ? (
                <div>
                  <p className="text-lg mb-2">No servers match your filters</p>
                  <p className="text-sm">Try adjusting your search or filter criteria</p>
                </div>
              ) : (
                <div>
                  <p className="text-lg mb-2">No remote servers configured</p>
                  <p className="text-sm mb-4">Get started by adding your first remote MCP server</p>
                  <Button onClick={() => setShowAddDialog(true)}>
                    <Plus className="h-4 w-4 mr-2" />
                    Add Your First Server
                  </Button>
                </div>
              )}
            </div>
          ) : (
            <div className="grid gap-4">
              {filteredServers.map((server) => (
                <ExternalServerCard
                  key={server.slug}
                  server={server}
                  onEdit={handleEdit}
                  onDelete={handleDelete}
                />
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Edit Dialog */}
      {editingServer && (
        <Dialog open={!!editingServer} onOpenChange={(open) => !open && setEditingServer(null)}>
          <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
            <DialogHeader>
              <DialogTitle>Edit Remote MCP Server</DialogTitle>
              <DialogDescription>
                Update the configuration for "{editingServer.name}".
              </DialogDescription>
            </DialogHeader>
            <ExternalServerForm
              server={editingServer}
              onSuccess={handleFormSuccess}
              onCancel={() => setEditingServer(null)}
            />
          </DialogContent>
        </Dialog>
      )}
    </div>
  );
}
