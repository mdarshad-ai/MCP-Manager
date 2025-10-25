import React from "react";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../components/ui/tabs";
import { Button } from "../components/ui/button";
import { Badge } from "../components/ui/badge";
import { Alert, AlertDescription } from "../components/ui/alert";
import { Trash2, Edit, Plus, AlertTriangle, Loader2, CheckSquare } from "lucide-react";
import { useToast } from "../hooks/use-toast";
import {
  listInstalledServers,
  deleteInstalledServer,
  listMarketplaceItems,
  addMarketplaceItem,
  updateMarketplaceItem,
  deleteMarketplaceItem,
  resolveAttentionItem
} from "../api";

interface InstalledServer {
  slug: string;
  name: string;
  status: 'running' | 'stopped' | 'error';
  path: string;
}

interface MarketplaceItem {
  slug: string;
  name: string;
  description: string;
  category: string;
  tags: string[];
  version: string;
  author: string;
  repository: string;
  license: string;
  createdAt: string;
  updatedAt: string;
  install?: { type: string; uri: string };
  remote?: { apiEndpoint: string; provider: string; authType?: string };
  configExample?: string;
  needsAttention?: boolean;
  attentionReason?: string;
}

export function AdminPanel() {
  const [installedServers, setInstalledServers] = React.useState<InstalledServer[]>([]);
  const [marketplaceItems, setMarketplaceItems] = React.useState<MarketplaceItem[]>([]);
  const [loading, setLoading] = React.useState(false);
  const [busyServer, setBusyServer] = React.useState<string | null>(null);
  const [showAddItemDialog, setShowAddItemDialog] = React.useState(false);
  const [editingItem, setEditingItem] = React.useState<MarketplaceItem | null>(null);
  const [selectedItems, setSelectedItems] = React.useState<string[]>([]);
  const { toast } = useToast();

  React.useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    try {
      console.log('Loading admin data...');
      const [servers, items] = await Promise.all([
        listInstalledServers(),
        listMarketplaceItems()
      ]);
      console.log('Admin data loaded:', { servers, items });
      setInstalledServers(servers || []);
      setMarketplaceItems(items || []);
    } catch (error) {
      console.error('Error loading admin panel data:', error);
      toast({ title: "Error loading data", description: `Failed to load admin panel data: ${error}`, variant: "destructive" });
    } finally {
      setLoading(false);
    }
  };

  const handleDeleteServer = async (slug: string) => {
    if (!confirm(`Are you sure you want to delete the installed server "${slug}"? This action cannot be undone.`)) {
      return;
    }

    setBusyServer(slug);
    try {
      await deleteInstalledServer(slug);
      setInstalledServers(prev => prev.filter(s => s.slug !== slug));
      toast({
        title: "Server deleted",
        description: `Successfully deleted server "${slug}"`
      });
    } catch (error: any) {
      toast({
        title: "Delete failed",
        description: error?.message || "Failed to delete server",
        variant: "destructive"
      });
    } finally {
      setBusyServer(null);
    }
  };

  const handleDeleteMarketplaceItem = async (slug: string) => {
    if (!confirm(`Are you sure you want to remove "${slug}" from the marketplace?`)) {
      return;
    }

    try {
      await deleteMarketplaceItem(slug);
      setMarketplaceItems(prev => prev.filter(i => i.slug !== slug));
      toast({
        title: "Item removed",
        description: `Removed "${slug}" from marketplace`
      });
    } catch (error: any) {
      toast({
        title: "Remove failed",
        description: error?.message || "Failed to remove marketplace item",
        variant: "destructive"
      });
    }
  };

  const handleAddMarketplaceItem = async (item: Omit<MarketplaceItem, 'createdAt' | 'updatedAt'>) => {
    try {
      const newItem = await addMarketplaceItem(item);
      setMarketplaceItems(prev => [...prev, newItem]);
      setShowAddItemDialog(false);
      toast({
        title: "Item added",
        description: `Added "${item.name}" to marketplace`
      });
    } catch (error: any) {
      toast({
        title: "Add failed",
        description: error?.message || "Failed to add marketplace item",
        variant: "destructive"
      });
    }
  };

  const handleEditMarketplaceItem = async (slug: string, updates: Partial<MarketplaceItem>) => {
    try {
      const updatedItem = await updateMarketplaceItem(slug, updates);
      setMarketplaceItems(prev => prev.map(i => i.slug === slug ? updatedItem : i));
      setEditingItem(null);
      toast({
        title: "Item updated",
        description: `Updated "${slug}" in marketplace`
      });
    } catch (error: any) {
      toast({
        title: "Update failed",
        description: error?.message || "Failed to update marketplace item",
        variant: "destructive"
      });
    }
  };

  const handleResolveAttention = async (slug: string) => {
    try {
      await resolveAttentionItem(slug);
      setMarketplaceItems(prev => prev.map(item => 
        item.slug === slug 
          ? { ...item, needsAttention: false, attentionReason: undefined }
          : item
      ));
      setSelectedItems(prev => prev.filter(item => item !== slug));
      toast({
        title: "Item Resolved",
        description: `Marked "${slug}" as fixed`
      });
    } catch (error: any) {
      toast({
        title: "Resolve failed",
        description: error?.message || "Failed to resolve attention item",
        variant: "destructive"
      });
    }
  };

  const handleBulkResolve = async () => {
    if (selectedItems.length === 0) return;
    
    try {
      await Promise.all(selectedItems.map(slug => resolveAttentionItem(slug)));
      setMarketplaceItems(prev => prev.map(item => 
        selectedItems.includes(item.slug)
          ? { ...item, needsAttention: false, attentionReason: undefined }
          : item
      ));
      setSelectedItems([]);
      toast({
        title: "Bulk Resolution Complete",
        description: `Resolved ${selectedItems.length} items successfully`
      });
    } catch (error: any) {
      toast({
        title: "Bulk resolve failed",
        description: error?.message || "Failed to resolve some items",
        variant: "destructive"
      });
    }
  };

  const handleSelectItem = (slug: string) => {
    setSelectedItems(prev => 
      prev.includes(slug) 
        ? prev.filter(item => item !== slug)
        : [...prev, slug]
    );
  };

  const itemsNeedingAttention = marketplaceItems.filter(item => item.needsAttention);

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'running':
        return <Badge variant="default" className="bg-green-500">Running</Badge>;
      case 'stopped':
        return <Badge variant="secondary">Stopped</Badge>;
      case 'error':
        return <Badge variant="destructive">Error</Badge>;
      default:
        return <Badge variant="outline">Unknown</Badge>;
    }
  };

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Admin Panel</h1>
          <p className="text-muted-foreground">Manage installed servers and marketplace listings</p>
        </div>
        <Button onClick={loadData} disabled={loading}>
          {loading ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : null}
          Refresh
        </Button>
      </div>

      <Alert>
        <AlertTriangle className="h-4 w-4" />
        <AlertDescription>
          Admin actions are permanent and cannot be undone. Use with caution.
        </AlertDescription>
      </Alert>

      <Tabs defaultValue="servers" className="space-y-4">
        <TabsList>
          <TabsTrigger value="servers">Installed Servers</TabsTrigger>
          <TabsTrigger value="needs-attention" className="flex items-center gap-2">
            Needs Attention
            {itemsNeedingAttention.length > 0 && (
              <Badge variant="destructive" className="text-xs">
                {itemsNeedingAttention.length}
              </Badge>
            )}
          </TabsTrigger>
          <TabsTrigger value="marketplace">Marketplace Management</TabsTrigger>
        </TabsList>

        <TabsContent value="servers" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Installed MCP Servers</CardTitle>
            </CardHeader>
            <CardContent>
              {installedServers.length === 0 ? (
                <p className="text-muted-foreground">No servers installed</p>
              ) : (
                <div className="space-y-2">
                  {installedServers.map((server) => (
                    <div key={server.slug} className="flex items-center justify-between p-3 border rounded">
                      <div className="flex items-center space-x-3">
                        <div>
                          <p className="font-medium">{server.name}</p>
                          <p className="text-sm text-muted-foreground">{server.slug}</p>
                        </div>
                        {getStatusBadge(server.status)}
                      </div>
                      <Button
                        variant="destructive"
                        size="sm"
                        onClick={() => handleDeleteServer(server.slug)}
                        disabled={busyServer === server.slug}
                      >
                        {busyServer === server.slug ? (
                          <Loader2 className="h-4 w-4 animate-spin" />
                        ) : (
                          <Trash2 className="h-4 w-4" />
                        )}
                      </Button>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="marketplace" className="space-y-4">
            <div className="flex justify-between items-center">
             <h3 className="text-lg font-medium">Marketplace Items</h3>
             <Button size="sm" onClick={() => setShowAddItemDialog(true)}>
               <Plus className="h-4 w-4 mr-2" />
               Add Item
             </Button>
           </div>

          <Card>
            <CardContent className="pt-6">
              {marketplaceItems.length === 0 ? (
                <p className="text-muted-foreground">No marketplace items</p>
              ) : (
                <div className="space-y-2">
                  {marketplaceItems.map((item) => (
                     <div key={item.slug} className="flex items-center justify-between p-3 border rounded">
                       <div className="flex-1">
                          <div className="flex items-center space-x-2 mb-2">
                            <p className="font-medium">{item.name}</p>
                            <Badge variant="outline">{item.category}</Badge>
                            <Badge variant="secondary">v{item.version}</Badge>
                            {item.needsAttention && (
                              <Badge variant="destructive" className="text-xs">
                                <AlertTriangle className="h-3 w-3 mr-1" />
                                Needs Attention
                              </Badge>
                            )}
                          </div>
                          <p className="text-sm text-muted-foreground mb-2">{item.description}</p>
                          {item.needsAttention && item.attentionReason && (
                            <Alert className="mb-2">
                              <AlertTriangle className="h-4 w-4" />
                              <AlertDescription className="text-xs">
                                {item.attentionReason}
                              </AlertDescription>
                            </Alert>
                          )}
                         <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
                           <span>Author: {item.author}</span>
                           <span>•</span>
                           <span>License: {item.license}</span>
                           <span>•</span>
                           <a href={item.repository} target="_blank" rel="noopener noreferrer" className="text-blue-600 hover:underline">
                             Repository
                           </a>
                         </div>
                         {item.tags && item.tags.length > 0 && (
                           <div className="flex flex-wrap gap-1 mt-2">
                             {item.tags.map((tag, index) => (
                               <Badge key={index} variant="outline" className="text-xs">
                                 {tag}
                               </Badge>
                             ))}
                           </div>
                         )}
                       </div>
                       <div className="flex space-x-2">
                         <Button
                           variant="outline"
                           size="sm"
                           onClick={() => setEditingItem(item)}
                         >
                           <Edit className="h-4 w-4" />
                         </Button>
                         <Button
                           variant="destructive"
                           size="sm"
                           onClick={() => handleDeleteMarketplaceItem(item.slug)}
                         >
                           <Trash2 className="h-4 w-4" />
                         </Button>
                       </div>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="needs-attention" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <AlertTriangle className="h-5 w-5 text-orange-500" />
                Items Needing Attention
                <Badge variant="destructive">
                  {itemsNeedingAttention.length}
                </Badge>
              </CardTitle>
            </CardHeader>
            <CardContent>
              {itemsNeedingAttention.length === 0 ? (
                <div className="text-center py-8">
                  <AlertTriangle className="h-12 w-12 text-green-500 mx-auto mb-4" />
                  <h3 className="text-lg font-semibold text-green-700 mb-2">All Items Are Healthy</h3>
                  <p className="text-muted-foreground">No marketplace items currently need attention.</p>
                </div>
              ) : (
                <div className="space-y-4">
                  {/* Bulk Actions */}
                  {selectedItems.length > 0 && (
                    <div className="flex items-center justify-between p-4 bg-blue-50 rounded-lg border border-blue-200">
                      <span className="text-blue-800">
                        {selectedItems.length} item(s) selected
                      </span>
                      <div className="flex gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => setSelectedItems([])}
                        >
                          Clear Selection
                        </Button>
                        <Button
                          size="sm"
                          onClick={handleBulkResolve}
                          className="bg-green-600 hover:bg-green-700"
                        >
                          Mark All as Fixed ({selectedItems.length})
                        </Button>
                      </div>
                    </div>
                  )}

                  {/* Items Needing Attention */}
                  <div className="space-y-4">
                    {itemsNeedingAttention.map((item) => (
                      <div key={item.slug} className="border-l-4 border-orange-500 bg-orange-50 p-4 rounded-lg">
                        <div className="flex items-start justify-between">
                          <div className="flex-1">
                            <div className="flex items-center gap-2 mb-2">
                              <input
                                type="checkbox"
                                checked={selectedItems.includes(item.slug)}
                                onChange={() => handleSelectItem(item.slug)}
                                className="h-4 w-4"
                              />
                              <AlertTriangle className="h-5 w-5 text-orange-500" />
                              <h3 className="font-semibold text-orange-800">{item.name}</h3>
                              <Badge variant="outline">{item.category}</Badge>
                              <Badge variant="secondary">v{item.version}</Badge>
                            </div>
                            <p className="text-sm text-muted-foreground mb-2">{item.description}</p>
                            <Alert className="mb-3 border-orange-200 bg-orange-100">
                              <AlertTriangle className="h-4 w-4" />
                              <AlertDescription className="text-orange-800">
                                <strong>Issue:</strong> {item.attentionReason}
                              </AlertDescription>
                            </Alert>
                            <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
                              <span>Author: {item.author}</span>
                              <span>•</span>
                              <span>License: {item.license}</span>
                              <span>•</span>
                              <a href={item.repository} target="_blank" rel="noopener noreferrer" className="text-blue-600 hover:underline">
                                Repository
                              </a>
                            </div>
                          </div>
                          <div className="flex items-center gap-2">
                            <Button
                              size="sm"
                              onClick={() => handleResolveAttention(item.slug)}
                              className="bg-green-600 hover:bg-green-700"
                            >
                              Mark as Fixed
                            </Button>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      {/* Add Marketplace Item Dialog */}
      {showAddItemDialog && (
        <MarketplaceItemDialog
          onSave={handleAddMarketplaceItem}
          onCancel={() => setShowAddItemDialog(false)}
        />
      )}

      {/* Edit Marketplace Item Dialog */}
      {editingItem && (
        <MarketplaceItemDialog
          item={editingItem}
          onSave={(updates) => handleEditMarketplaceItem(editingItem.slug, updates)}
          onCancel={() => setEditingItem(null)}
        />
      )}
    </div>
  );
}

// Marketplace Item Dialog Component
function MarketplaceItemDialog({
  item,
  onSave,
  onCancel
}: {
  item?: MarketplaceItem;
  onSave: (item: Omit<MarketplaceItem, 'createdAt' | 'updatedAt'>) => void;
  onCancel: () => void;
}) {
  const [formData, setFormData] = React.useState({
    slug: item?.slug || '',
    name: item?.name || '',
    description: item?.description || '',
    category: item?.category || '',
    tags: item?.tags?.join(', ') || '',
    version: item?.version || '',
    author: item?.author || '',
    repository: item?.repository || '',
    license: item?.license || '',
    installType: item?.install?.type || '',
    installUri: item?.install?.uri || '',
    remoteApiEndpoint: item?.remote?.apiEndpoint || '',
    remoteProvider: item?.remote?.provider || '',
    remoteAuthType: item?.remote?.authType || '',
    configExample: item?.configExample || ''
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const itemData: any = {
      slug: formData.slug,
      name: formData.name,
      description: formData.description,
      category: formData.category,
      tags: formData.tags ? formData.tags.split(',').map(t => t.trim()).filter(t => t) : undefined,
      version: formData.version,
      author: formData.author,
      repository: formData.repository,
      license: formData.license,
      configExample: formData.configExample
    };

    // Add install if provided
    if (formData.installType && formData.installUri) {
      itemData.install = {
        type: formData.installType,
        uri: formData.installUri
      };
    }

    // Add remote if provided
    if (formData.remoteApiEndpoint && formData.remoteProvider) {
      itemData.remote = {
        apiEndpoint: formData.remoteApiEndpoint,
        provider: formData.remoteProvider,
        authType: formData.remoteAuthType || undefined
      };
    }

    onSave(itemData);
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center p-4 z-50">
      <div className="bg-white text-black rounded-lg p-6 w-full max-w-2xl max-h-[90vh] overflow-y-auto">
        <h2 className="text-xl font-semibold mb-4 text-black">
          {item ? 'Edit Marketplace Item' : 'Add Marketplace Item'}
        </h2>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1 text-black">Slug *</label>
              <input
                type="text"
                value={formData.slug}
                onChange={(e) => setFormData(prev => ({ ...prev, slug: e.target.value }))}
                className="w-full p-2 border rounded text-black bg-white"
                required
                disabled={!!item} // Can't change slug when editing
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1 text-black">Name *</label>
              <input
                type="text"
                value={formData.name}
                onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
                className="w-full p-2 border rounded text-black bg-white"
                required
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1 text-black">Category *</label>
              <input
                type="text"
                value={formData.category}
                onChange={(e) => setFormData(prev => ({ ...prev, category: e.target.value }))}
                className="w-full p-2 border rounded text-black bg-white"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1 text-black">Version</label>
              <input
                type="text"
                value={formData.version}
                onChange={(e) => setFormData(prev => ({ ...prev, version: e.target.value }))}
                className="w-full p-2 border rounded text-black bg-white"
                placeholder="1.0.0"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1 text-black">Description</label>
            <textarea
              value={formData.description}
              onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
              className="w-full p-2 border rounded text-black bg-white"
              rows={3}
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1 text-black">Author</label>
              <input
                type="text"
                value={formData.author}
                onChange={(e) => setFormData(prev => ({ ...prev, author: e.target.value }))}
                className="w-full p-2 border rounded text-black bg-white"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1 text-black">License</label>
              <input
                type="text"
                value={formData.license}
                onChange={(e) => setFormData(prev => ({ ...prev, license: e.target.value }))}
                className="w-full p-2 border rounded text-black bg-white"
                placeholder="MIT"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1 text-black">Repository URL</label>
            <input
              type="url"
              value={formData.repository}
              onChange={(e) => setFormData(prev => ({ ...prev, repository: e.target.value }))}
              className="w-full p-2 border rounded text-black bg-white"
              placeholder="https://github.com/user/repo"
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-1 text-black">Tags (comma-separated)</label>
            <input
              type="text"
              value={formData.tags}
              onChange={(e) => setFormData(prev => ({ ...prev, tags: e.target.value }))}
              className="w-full p-2 border rounded text-black bg-white"
              placeholder="tag1, tag2, tag3"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1 text-black">Install Type</label>
              <select
                value={formData.installType}
                onChange={(e) => setFormData(prev => ({ ...prev, installType: e.target.value }))}
                className="w-full p-2 border rounded text-black bg-white"
              >
                <option value="">Select type</option>
                <option value="npm">NPM</option>
                <option value="git">Git</option>
                <option value="pip">Pip</option>
                <option value="docker-image">Docker Image</option>
                <option value="docker-compose">Docker Compose</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1 text-black">Install URI</label>
              <input
                type="text"
                value={formData.installUri}
                onChange={(e) => setFormData(prev => ({ ...prev, installUri: e.target.value }))}
                className="w-full p-2 border rounded text-black bg-white"
                placeholder="package-name or git-url"
              />
            </div>
          </div>

          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1 text-black">Remote API Endpoint</label>
              <input
                type="url"
                value={formData.remoteApiEndpoint}
                onChange={(e) => setFormData(prev => ({ ...prev, remoteApiEndpoint: e.target.value }))}
                className="w-full p-2 border rounded text-black bg-white"
                placeholder="https://api.example.com/mcp"
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1 text-black">Remote Provider</label>
              <input
                type="text"
                value={formData.remoteProvider}
                onChange={(e) => setFormData(prev => ({ ...prev, remoteProvider: e.target.value }))}
                className="w-full p-2 border rounded text-black bg-white"
                placeholder="GitHub, Notion, etc."
              />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1 text-black">Remote Auth Type</label>
              <select
                value={formData.remoteAuthType}
                onChange={(e) => setFormData(prev => ({ ...prev, remoteAuthType: e.target.value }))}
                className="w-full p-2 border rounded text-black bg-white"
              >
                <option value="">Select auth type</option>
                <option value="api_key">API Key</option>
                <option value="oauth2">OAuth2</option>
                <option value="basic">Basic</option>
              </select>
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1 text-black">Config Example</label>
            <textarea
              value={formData.configExample}
              onChange={(e) => setFormData(prev => ({ ...prev, configExample: e.target.value }))}
              className="w-full p-2 border rounded text-black bg-white font-mono text-sm"
              rows={4}
              placeholder='{"mcpServers": {"server-name": {"command": "npx", "args": ["package"]}}}'
            />
          </div>

          <div className="flex justify-end space-x-2 pt-4">
            <Button type="button" variant="outline" onClick={onCancel}>
              Cancel
            </Button>
            <Button type="submit">
              {item ? 'Update' : 'Add'} Item
            </Button>
          </div>
        </form>
      </div>
    </div>
  );
}