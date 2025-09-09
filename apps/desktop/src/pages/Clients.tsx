import {
  AlertCircle,
  Brain,
  Bot,
  Check,
  ChevronDown,
  ChevronRight,
  Cloud,
  Code2,
  Database,
  Download,
  FileText,
  FolderOpen,
  GitBranch,
  Github,
  Globe,
  HardDrive,
  MessageSquare,
  MousePointer,
  RefreshCw,
  Save,
  Search,
  Settings,
  Upload,
  Users,
  X,
  Zap,
} from "lucide-react";
import React from "react";
import {
  type ClientDetection,
  clientCurrent,
  clientsApply,
  clientsDetect,
  fetchServers,
} from "../api";
import { Alert, AlertDescription } from "../components/ui/alert";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Checkbox } from "../components/ui/checkbox";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "../components/ui/collapsible";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "../components/ui/dialog";
import { Input } from "../components/ui/input";
import { ScrollArea } from "../components/ui/scroll-area";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../components/ui/select";
import { Separator } from "../components/ui/separator";
import { ToggleGroup, ToggleGroupItem } from "../components/ui/toggle-group";
import { useToast } from "../hooks/use-toast";

// Tool categories with icons
const toolCategories = {
  "AI & Reasoning": {
    icon: Brain,
    tools: ["mcp-think-tool", "sequentialthinking", "structured-thinking"],
    description: "AI-powered reasoning and analysis tools",
  },
  "Development": {
    icon: Code2,
    tools: ["filesystem", "github", "git", "postgres"],
    description: "Development and version control tools",
  },
  "Automation": {
    icon: Bot,
    tools: ["browser-automation", "playwright", "fetch", "cursor-command-center"],
    description: "Automation and testing tools",
  },
  "Collaboration": {
    icon: Users,
    tools: ["slack", "notion"],
    description: "Team collaboration and communication tools",
  },
  "Storage": {
    icon: HardDrive,
    tools: ["memory-server"],
    description: "Data persistence and storage tools",
  },
  "Services": {
    icon: Cloud,
    tools: ["render"],
    description: "Cloud services and deployment tools",
  },
};

// Tool metadata
const toolMetadata: Record<string, { icon: React.ElementType; description: string }> = {
  "mcp-think-tool": { icon: Brain, description: "Advanced AI reasoning and analysis" },
  "sequentialthinking": { icon: RefreshCw, description: "Step-by-step problem solving" },
  "structured-thinking": { icon: Zap, description: "Structured approach to complex tasks" },
  "filesystem": { icon: FolderOpen, description: "File system access and operations" },
  "github": { icon: Github, description: "GitHub repository management" },
  "git": { icon: GitBranch, description: "Git version control operations" },
  "postgres": { icon: Database, description: "PostgreSQL database access" },
  "browser-automation": { icon: Bot, description: "Control and automate browsers" },
  "playwright": { icon: MousePointer, description: "Web testing and automation" },
  "fetch": { icon: Globe, description: "Make HTTP requests and fetch data" },
  "cursor-command-center": { icon: MousePointer, description: "Cursor IDE command center" },
  "slack": { icon: MessageSquare, description: "Slack messaging integration" },
  "notion": { icon: FileText, description: "Notion workspace integration" },
  "memory-server": { icon: HardDrive, description: "Persistent memory storage" },
  "render": { icon: Cloud, description: "Render deployment service" },
};

type ViewMode = "matrix" | "list";
type ToolAssignment = Record<string, Record<string, boolean>>;

interface ConfigDialogState {
  client: string;
  current: any;
  preview: any;
}

interface ConfigureDialogState {
  client: string;
  tools: Record<string, boolean>;
}

// Helper function to merge configurations
function mergeConfigurations(current: any, preview: any): any {
  const merged = JSON.parse(JSON.stringify(current || {}));
  
  const usesServersArray = Array.isArray(current?.servers);
  const usesMcpServersObject = current?.mcpServers && typeof current.mcpServers === 'object';
  
  if (preview?.servers && Array.isArray(preview.servers)) {
    if (usesServersArray) {
      if (!merged.servers) {
        merged.servers = [];
      }
      
      const existingServerNames = new Set(merged.servers.map((s: any) => s.name));
      
      for (const server of preview.servers) {
        if (!existingServerNames.has(server.name)) {
          merged.servers.push(server);
        }
      }
    } else if (usesMcpServersObject) {
      if (!merged.mcpServers) {
        merged.mcpServers = {};
      }
      
      for (const server of preview.servers) {
        if (!merged.mcpServers[server.name]) {
          merged.mcpServers[server.name] = {
            command: server.command,
            args: server.args || [],
            ...(server.env && { env: server.env }),
            ...(server.transport && { transport: server.transport })
          };
        }
      }
    } else {
      merged.servers = preview.servers;
    }
  }
  
  for (const key in preview) {
    if (key !== 'servers' && preview[key] !== undefined) {
      merged[key] = preview[key];
    }
  }
  
  return merged;
}

// Component to display configuration with highlighting
function ConfigurationDisplay({ current, preview }: { current: any; preview: any }) {
  const merged = mergeConfigurations(current, preview);
  
  const renderConfigWithHighlight = () => {
    const mergedStr = JSON.stringify(merged, null, 2);
    
    let existingServerNames: string[] = [];
    if (current?.servers && Array.isArray(current.servers)) {
      existingServerNames = current.servers.map((s: any) => s.name);
    } else if (current?.mcpServers && typeof current.mcpServers === 'object') {
      existingServerNames = Object.keys(current.mcpServers);
    }
    
    const newServerNames = preview?.servers 
      ? preview.servers.map((s: any) => s.name).filter((name: string) => !existingServerNames.includes(name))
      : [];
    
    const lines = mergedStr.split('\n');
    let inNewServer = false;
    let currentKey = '';
    let braceDepth = 0;
    let inServersSection = false;
    let inMcpServersSection = false;
    
    return lines.map((line, idx) => {
      const openBraces = (line.match(/{/g) || []).length;
      const closeBraces = (line.match(/}/g) || []).length;
      braceDepth += openBraces - closeBraces;
      
      if (line.includes('"servers":')) {
        inServersSection = true;
        inMcpServersSection = false;
      } else if (line.includes('"mcpServers":')) {
        inMcpServersSection = true;
        inServersSection = false;
      }
      
      if (inServersSection && line.includes('"name":')) {
        const nameMatch = line.match(/"name":\s*"([^"]+)"/);
        if (nameMatch) {
          const serverName = nameMatch[1];
          inNewServer = newServerNames.includes(serverName);
        }
      }
      
      if (inMcpServersSection) {
        const keyMatch = line.match(/^\s{4}"([^"]+)":\s*{/);
        if (keyMatch) {
          currentKey = keyMatch[1];
          inNewServer = newServerNames.includes(currentKey);
        }
      }
      
      if (inNewServer && braceDepth === (inServersSection ? 1 : 2) && line.trim().startsWith('}')) {
        const wasInNewServer = inNewServer;
        inNewServer = false;
        return (
          <div key={idx} className={wasInNewServer ? "bg-green-100 dark:bg-green-900/20" : ""}>
            {line}
          </div>
        );
      }
      
      return (
        <div key={idx} className={inNewServer ? "bg-green-100 dark:bg-green-900/20" : ""}>
          {line}
        </div>
      );
    });
  };
  
  return <>{renderConfigWithHighlight()}</>;
}

export function Clients() {
  const { toast } = useToast();
  const [viewMode, setViewMode] = React.useState<ViewMode>("matrix");
  const [clients, setClients] = React.useState<ClientDetection[]>([]);
  const [availableTools, setAvailableTools] = React.useState<string[]>([]);
  const [assignments, setAssignments] = React.useState<ToolAssignment>({});
  const [searchTerm, setSearchTerm] = React.useState("");
  const [selectedCategory, setSelectedCategory] = React.useState("all");
  const [loading, setLoading] = React.useState(false);
  const [refreshing, setRefreshing] = React.useState(false);
  const [saveStatus, setSaveStatus] = React.useState<Record<string, "saved" | "pending" | "error">>({});
  const [expandedCategories, setExpandedCategories] = React.useState<Set<string>>(
    new Set(Object.keys(toolCategories))
  );
  const [expandedClients, setExpandedClients] = React.useState<Set<string>>(new Set());
  const [configDialog, setConfigDialog] = React.useState<ConfigDialogState | null>(null);
  const [configureDialog, setConfigureDialog] = React.useState<ConfigureDialogState | null>(null);

  // Load initial data
  React.useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setRefreshing(true);
    try {
      // Detect clients
      const detectedClients = await clientsDetect();
      setClients(detectedClients);

      // Get unique tools from all sources
      const toolSet = new Set<string>();
      
      // Add tools from running servers
      try {
        const servers = await fetchServers();
        servers.forEach((s: any) => toolSet.add(s.slug));
      } catch (err) {
        console.error('Failed to fetch servers:', err);
      }

      // Add tools from client configurations
      for (const client of detectedClients.filter(c => c.detected)) {
        try {
          const config = await clientCurrent(client.name);
          if (config?.servers && Array.isArray(config.servers)) {
            config.servers.forEach((s: any) => s.name && toolSet.add(s.name));
          } else if (config?.mcpServers && typeof config.mcpServers === 'object') {
            Object.keys(config.mcpServers).forEach(name => toolSet.add(name));
          }
        } catch (err) {
          console.error(`Failed to load config for ${client.name}:`, err);
        }
      }

      // Add known tools from metadata
      Object.keys(toolMetadata).forEach(tool => toolSet.add(tool));

      const tools = Array.from(toolSet).sort();
      setAvailableTools(tools);

      // Load current assignments
      const newAssignments: ToolAssignment = {};
      for (const client of detectedClients) {
        newAssignments[client.name] = {};
        
        if (client.detected) {
          try {
            const config = await clientCurrent(client.name);
            
            if (config?.servers && Array.isArray(config.servers)) {
              for (const tool of tools) {
                newAssignments[client.name][tool] = config.servers.some((s: any) => s.name === tool);
              }
            } else if (config?.mcpServers && typeof config.mcpServers === 'object') {
              for (const tool of tools) {
                newAssignments[client.name][tool] = !!config.mcpServers[tool];
              }
            } else {
              for (const tool of tools) {
                newAssignments[client.name][tool] = false;
              }
            }
          } catch (err) {
            for (const tool of tools) {
              newAssignments[client.name][tool] = false;
            }
          }
        } else {
          for (const tool of tools) {
            newAssignments[client.name][tool] = false;
          }
        }
      }
      setAssignments(newAssignments);
    } catch (err) {
      console.error('Failed to load data:', err);
      toast({
        title: "Error",
        description: "Failed to load client configurations",
        variant: "destructive",
      });
    } finally {
      setRefreshing(false);
    }
  };

  const handleCheckboxChange = (client: string, tool: string, checked: boolean) => {
    setAssignments(prev => ({
      ...prev,
      [client]: {
        ...prev[client],
        [tool]: checked,
      },
    }));
    setSaveStatus(prev => ({
      ...prev,
      [client]: "pending",
    }));
  };

  const handleSelectColumn = (client: string, selectAll: boolean) => {
    const visibleTools = getFilteredTools();
    setAssignments(prev => ({
      ...prev,
      [client]: {
        ...prev[client],
        ...Object.fromEntries(visibleTools.map(tool => [tool, selectAll])),
      },
    }));
    setSaveStatus(prev => ({
      ...prev,
      [client]: "pending",
    }));
  };

  const toggleCategory = (category: string) => {
    setExpandedCategories(prev => {
      const next = new Set(prev);
      if (next.has(category)) {
        next.delete(category);
      } else {
        next.add(category);
      }
      return next;
    });
  };

  const toggleClient = (client: string) => {
    setExpandedClients(prev => {
      const next = new Set(prev);
      if (next.has(client)) {
        next.delete(client);
      } else {
        next.add(client);
      }
      return next;
    });
  };

  const getFilteredTools = () => {
    let tools = availableTools;
    
    if (selectedCategory !== "all") {
      const category = toolCategories[selectedCategory as keyof typeof toolCategories];
      if (category) {
        tools = tools.filter(tool => category.tools.includes(tool));
      }
    }
    
    if (searchTerm) {
      tools = tools.filter(tool =>
        tool.toLowerCase().includes(searchTerm.toLowerCase()) ||
        toolMetadata[tool]?.description.toLowerCase().includes(searchTerm.toLowerCase())
      );
    }
    
    return tools;
  };

  const getToolsByCategory = () => {
    const filtered = getFilteredTools();
    const categorized: Record<string, string[]> = {};
    const uncategorized: string[] = [];

    for (const tool of filtered) {
      let found = false;
      for (const [category, data] of Object.entries(toolCategories)) {
        if (data.tools.includes(tool)) {
          if (!categorized[category]) {
            categorized[category] = [];
          }
          categorized[category].push(tool);
          found = true;
          break;
        }
      }
      if (!found) {
        uncategorized.push(tool);
      }
    }

    if (uncategorized.length > 0) {
      categorized["Other"] = uncategorized;
    }

    return categorized;
  };

  const openConfigureDialog = (clientName: string) => {
    setConfigureDialog({
      client: clientName,
      tools: assignments[clientName] || {},
    });
  };

  const openAdvancedDialog = async (clientName: string) => {
    setLoading(true);
    try {
      const current = await clientCurrent(clientName);
      const preview = {
        servers: availableTools
          .filter(tool => assignments[clientName]?.[tool])
          .map(tool => ({
            name: tool,
            command: `/path/to/${tool}`,
            args: [],
            transport: "stdio",
          })),
      };
      
      setConfigDialog({
        client: clientName,
        current,
        preview,
      });
    } catch (err) {
      toast({
        title: "Error",
        description: `Failed to load configuration: ${(err as Error).message}`,
        variant: "destructive",
      });
    } finally {
      setLoading(false);
    }
  };

  const applyConfigureDialog = () => {
    if (!configureDialog) return;
    
    setAssignments(prev => ({
      ...prev,
      [configureDialog.client]: configureDialog.tools,
    }));
    setSaveStatus(prev => ({
      ...prev,
      [configureDialog.client]: "pending",
    }));
    setConfigureDialog(null);
  };

  const applyAdvancedDialog = async () => {
    if (!configDialog) return;
    
    setLoading(true);
    try {
      const mergedConfig = mergeConfigurations(configDialog.current, configDialog.preview);
      await clientsApply(configDialog.client, mergedConfig);
      setSaveStatus(prev => ({
        ...prev,
        [configDialog.client]: "saved",
      }));
      toast({
        title: "Configuration Applied",
        description: `Successfully updated configuration for ${configDialog.client}`,
      });
      setConfigDialog(null);
      await loadData();
    } catch (err) {
      toast({
        title: "Error",
        description: `Failed to apply configuration: ${(err as Error).message}`,
        variant: "destructive",
      });
    } finally {
      setLoading(false);
    }
  };

  const saveClientConfiguration = async (clientName: string) => {
    setLoading(true);
    try {
      const client = clients.find(c => c.name === clientName);
      if (!client) return;

      const currentConfig = await clientCurrent(clientName);
      let newConfig: any;

      if (currentConfig?.servers && Array.isArray(currentConfig.servers)) {
        newConfig = {
          servers: availableTools
            .filter(tool => assignments[clientName]?.[tool])
            .map(tool => ({
              name: tool,
              command: `/path/to/${tool}`,
              args: [],
              transport: "stdio",
            })),
        };
      } else if (currentConfig?.mcpServers || clientName.includes("Cursor")) {
        newConfig = {
          mcpServers: Object.fromEntries(
            availableTools
              .filter(tool => assignments[clientName]?.[tool])
              .map(tool => [
                tool,
                {
                  command: `/path/to/${tool}`,
                  args: [],
                },
              ])
          ),
        };
      } else {
        newConfig = {
          servers: availableTools
            .filter(tool => assignments[clientName]?.[tool])
            .map(tool => ({
              name: tool,
              command: `/path/to/${tool}`,
              args: [],
              transport: "stdio",
            })),
        };
      }

      await clientsApply(clientName, newConfig);
      setSaveStatus(prev => ({
        ...prev,
        [clientName]: "saved",
      }));
      toast({
        title: "Configuration Saved",
        description: `Successfully saved configuration for ${clientName}`,
      });
    } catch (err) {
      setSaveStatus(prev => ({
        ...prev,
        [clientName]: "error",
      }));
      toast({
        title: "Save Failed",
        description: `Failed to save configuration for ${clientName}`,
        variant: "destructive",
      });
    } finally {
      setLoading(false);
    }
  };

  const saveAllConfigurations = async () => {
    const pendingClients = Object.entries(saveStatus)
      .filter(([_, status]) => status === "pending")
      .map(([client]) => client);

    for (const client of pendingClients) {
      await saveClientConfiguration(client);
    }
  };

  const exportConfiguration = () => {
    const exportData = {
      version: "1.0",
      timestamp: new Date().toISOString(),
      assignments,
    };
    const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `mcp-assignments-${Date.now()}.json`;
    a.click();
    URL.revokeObjectURL(url);
    toast({
      title: "Exported",
      description: "Configuration exported successfully",
    });
  };

  const importConfiguration = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = (e) => {
      try {
        const data = JSON.parse(e.target?.result as string);
        if (data.assignments) {
          setAssignments(data.assignments);
          const newSaveStatus: Record<string, "pending"> = {};
          for (const client of clients) {
            newSaveStatus[client.name] = "pending";
          }
          setSaveStatus(newSaveStatus);
          toast({
            title: "Imported",
            description: "Configuration imported successfully",
          });
        }
      } catch (err) {
        toast({
          title: "Import Failed",
          description: "Invalid configuration file",
          variant: "destructive",
        });
      }
    };
    reader.readAsText(file);
  };

  const hasUnsavedChanges = Object.values(saveStatus).some(status => status === "pending");
  const categorizedTools = getToolsByCategory();

  return (
    <div className="p-6 space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Client Configuration</h1>
          <p className="text-muted-foreground">Manage MCP tool assignments across all clients</p>
        </div>
        <div className="flex gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={exportConfiguration}
          >
            <Download className="h-4 w-4 mr-2" />
            Export
          </Button>
          <label>
            <Button variant="outline" size="sm" asChild>
              <span>
                <Upload className="h-4 w-4 mr-2" />
                Import
                <input
                  type="file"
                  accept=".json"
                  className="hidden"
                  onChange={importConfiguration}
                />
              </span>
            </Button>
          </label>
          <Button
            variant="outline"
            size="sm"
            onClick={loadData}
            disabled={refreshing}
          >
            <RefreshCw className={`h-4 w-4 mr-2 ${refreshing ? "animate-spin" : ""}`} />
            Refresh
          </Button>
        </div>
      </div>

      {/* Controls */}
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-4">
              <ToggleGroup type="single" value={viewMode} onValueChange={(v) => v && setViewMode(v as ViewMode)}>
                <ToggleGroupItem value="matrix" aria-label="Matrix view">
                  Matrix
                </ToggleGroupItem>
                <ToggleGroupItem value="list" aria-label="List view">
                  List
                </ToggleGroupItem>
              </ToggleGroup>
              
              <Select value={selectedCategory} onValueChange={setSelectedCategory}>
                <SelectTrigger className="w-48">
                  <SelectValue placeholder="Filter by category" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Categories</SelectItem>
                  {Object.keys(toolCategories).map(category => {
                    const CategoryIcon = toolCategories[category as keyof typeof toolCategories].icon;
                    return (
                      <SelectItem key={category} value={category}>
                        <div className="flex items-center gap-2">
                          <CategoryIcon className="h-4 w-4" />
                          {category}
                        </div>
                      </SelectItem>
                    );
                  })}
                </SelectContent>
              </Select>
              
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-muted-foreground h-4 w-4" />
                <Input
                  placeholder="Search tools..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="pl-9 w-64"
                />
              </div>
            </div>
            
            {hasUnsavedChanges && (
              <Button
                onClick={saveAllConfigurations}
                disabled={loading}
                className="animate-pulse"
              >
                <Save className="h-4 w-4 mr-2" />
                Save All Changes
              </Button>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Main Content */}
      {viewMode === "matrix" ? (
        <Card>
          <CardContent className="p-0">
            <div className="border rounded-lg overflow-x-auto">
              <table className="w-full">
                <thead className="bg-muted/50 sticky top-0">
                  <tr>
                    <th className="text-left p-3 font-medium min-w-[200px]">Tool Name</th>
                    {clients.map(client => (
                      <th key={client.name} className="text-center p-3 font-medium min-w-[120px]">
                        <div className="space-y-1">
                          <div className="flex items-center justify-center gap-2">
                            <span className="text-sm">{client.name.replace(" (Global)", "")}</span>
                            {!client.detected && (
                              <Badge variant="outline" className="text-xs">
                                Not Found
                              </Badge>
                            )}
                            {saveStatus[client.name] === "saved" && (
                              <Check className="h-3 w-3 text-green-500" />
                            )}
                            {saveStatus[client.name] === "error" && (
                              <X className="h-3 w-3 text-red-500" />
                            )}
                          </div>
                          {client.detected && (
                            <div className="flex gap-1 justify-center">
                              <Button
                                variant="ghost"
                                size="sm"
                                className="h-5 text-xs px-1"
                                onClick={() => handleSelectColumn(client.name, true)}
                              >
                                All
                              </Button>
                              <Button
                                variant="ghost"
                                size="sm"
                                className="h-5 text-xs px-1"
                                onClick={() => handleSelectColumn(client.name, false)}
                              >
                                None
                              </Button>
                              <Button
                                variant="ghost"
                                size="sm"
                                className="h-5 text-xs px-1"
                                onClick={() => openAdvancedDialog(client.name)}
                              >
                                Advanced
                              </Button>
                            </div>
                          )}
                        </div>
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {Object.entries(categorizedTools).map(([category, tools]) => (
                    <React.Fragment key={category}>
                      <tr className="bg-muted/30">
                        <td colSpan={clients.length + 1} className="p-2">
                          <button
                            onClick={() => toggleCategory(category)}
                            className="flex items-center gap-2 w-full text-left hover:bg-muted/50 rounded px-2 py-1"
                          >
                            {expandedCategories.has(category) ? (
                              <ChevronDown className="h-4 w-4" />
                            ) : (
                              <ChevronRight className="h-4 w-4" />
                            )}
                            <span className="font-medium flex items-center gap-2">
                              {category === "Other" ? (
                                <FolderOpen className="h-4 w-4" />
                              ) : (
                                React.createElement(toolCategories[category as keyof typeof toolCategories]?.icon || FolderOpen, { className: "h-4 w-4" })
                              )}
                              {category}
                            </span>
                            <span className="text-sm text-muted-foreground">({tools.length})</span>
                          </button>
                        </td>
                      </tr>
                      {expandedCategories.has(category) && tools.map((tool, idx) => (
                        <tr key={tool} className={idx % 2 === 0 ? "bg-background" : "bg-muted/10"}>
                          <td className="p-3">
                            <div className="flex items-center gap-2">
                              {React.createElement(toolMetadata[tool]?.icon || FolderOpen, { className: "h-5 w-5" })}
                              <div>
                                <div className="font-mono text-sm">{tool}</div>
                                <div className="text-xs text-muted-foreground">{toolMetadata[tool]?.description}</div>
                              </div>
                            </div>
                          </td>
                          {clients.map(client => (
                            <td key={`${client.name}-${tool}`} className="text-center p-3">
                              <Checkbox
                                checked={assignments[client.name]?.[tool] || false}
                                onCheckedChange={(checked) =>
                                  handleCheckboxChange(client.name, tool, checked as boolean)
                                }
                                disabled={!client.detected}
                              />
                            </td>
                          ))}
                        </tr>
                      ))}
                    </React.Fragment>
                  ))}
                </tbody>
              </table>
            </div>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          {clients.map(client => {
            const isExpanded = expandedClients.has(client.name);
            const activeTools = availableTools.filter(tool => assignments[client.name]?.[tool]);
            const categorizedActiveTools: Record<string, string[]> = {};
            
            for (const tool of activeTools) {
              for (const [category, data] of Object.entries(toolCategories)) {
                if (data.tools.includes(tool)) {
                  if (!categorizedActiveTools[category]) {
                    categorizedActiveTools[category] = [];
                  }
                  categorizedActiveTools[category].push(tool);
                  break;
                }
              }
            }

            return (
              <Card key={client.name}>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      <button
                        onClick={() => toggleClient(client.name)}
                        className="hover:bg-muted rounded p-1"
                      >
                        {isExpanded ? (
                          <ChevronDown className="h-4 w-4" />
                        ) : (
                          <ChevronRight className="h-4 w-4" />
                        )}
                      </button>
                      <div>
                        <CardTitle className="text-lg flex items-center gap-2">
                          {client.name}
                          {client.detected ? (
                            <Badge variant="default" className="text-xs">Detected</Badge>
                          ) : (
                            <Badge variant="outline" className="text-xs">Not Found</Badge>
                          )}
                          {saveStatus[client.name] === "pending" && (
                            <Badge variant="outline" className="text-xs animate-pulse">Unsaved</Badge>
                          )}
                        </CardTitle>
                        {client.path && (
                          <CardDescription className="text-xs mt-1">
                            Config: {client.path}
                          </CardDescription>
                        )}
                      </div>
                    </div>
                    <div className="flex gap-2">
                      {client.detected && (
                        <>
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => openConfigureDialog(client.name)}
                          >
                            <Settings className="h-4 w-4 mr-2" />
                            Configure
                          </Button>
                          <Button
                            size="sm"
                            variant="outline"
                            onClick={() => openAdvancedDialog(client.name)}
                          >
                            Advanced
                          </Button>
                          {saveStatus[client.name] === "pending" && (
                            <Button
                              size="sm"
                              onClick={() => saveClientConfiguration(client.name)}
                              disabled={loading}
                            >
                              <Save className="h-4 w-4 mr-2" />
                              Save
                            </Button>
                          )}
                        </>
                      )}
                    </div>
                  </div>
                </CardHeader>
                {isExpanded && (
                  <CardContent>
                    {Object.entries(categorizedActiveTools).length > 0 ? (
                      <div className="space-y-2">
                        {Object.entries(categorizedActiveTools).map(([category, tools]) => (
                          <div key={category} className="flex items-start gap-3">
                            <span className="text-sm font-medium text-muted-foreground whitespace-nowrap flex items-center gap-2">
                              {React.createElement(toolCategories[category as keyof typeof toolCategories]?.icon || FolderOpen, { className: "h-4 w-4" })}
                              {category}:
                            </span>
                            <div className="flex flex-wrap gap-2">
                              {tools.map(tool => (
                                <Badge key={tool} variant="secondary" className="text-xs flex items-center gap-1">
                                  {React.createElement(toolMetadata[tool]?.icon || FolderOpen, { className: "h-3 w-3" })}
                                  {tool}
                                </Badge>
                              ))}
                            </div>
                          </div>
                        ))}
                      </div>
                    ) : (
                      <p className="text-sm text-muted-foreground">No tools configured</p>
                    )}
                    <Separator className="my-3" />
                    <p className="text-sm text-muted-foreground">
                      Total: {activeTools.length} tool{activeTools.length !== 1 ? 's' : ''} active
                    </p>
                  </CardContent>
                )}
              </Card>
            );
          })}
        </div>
      )}

      {/* Configure Dialog */}
      <Dialog open={!!configureDialog} onOpenChange={(open) => !open && setConfigureDialog(null)}>
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>Configure Tools: {configureDialog?.client}</DialogTitle>
            <DialogDescription>
              Select the tools you want to enable for this client
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-muted-foreground h-4 w-4" />
              <Input
                placeholder="Search tools..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                className="pl-9"
              />
            </div>
            <ScrollArea className="h-[400px] pr-4">
              <div className="space-y-4">
                {Object.entries(categorizedTools).map(([category, tools]) => (
                  <Collapsible key={category} defaultOpen>
                    <CollapsibleTrigger className="flex items-center gap-2 w-full text-left hover:bg-muted rounded px-2 py-1">
                      <ChevronDown className="h-4 w-4" />
                      <span className="font-medium flex items-center gap-2">
                        {category === "Other" ? (
                          <FolderOpen className="h-4 w-4" />
                        ) : (
                          React.createElement(toolCategories[category as keyof typeof toolCategories]?.icon || FolderOpen, { className: "h-4 w-4" })
                        )}
                        {category}
                      </span>
                      <span className="text-sm text-muted-foreground">({tools.length})</span>
                    </CollapsibleTrigger>
                    <CollapsibleContent className="mt-2 space-y-2 pl-6">
                      {tools.map(tool => (
                        <label
                          key={tool}
                          className="flex items-center gap-3 p-2 hover:bg-muted rounded cursor-pointer"
                        >
                          <Checkbox
                            checked={configureDialog?.tools[tool] || false}
                            onCheckedChange={(checked) => {
                              if (configureDialog) {
                                setConfigureDialog({
                                  ...configureDialog,
                                  tools: {
                                    ...configureDialog.tools,
                                    [tool]: checked as boolean,
                                  },
                                });
                              }
                            }}
                          />
                          {React.createElement(toolMetadata[tool]?.icon || FolderOpen, { className: "h-5 w-5" })}
                          <div className="flex-1">
                            <div className="font-mono text-sm">{tool}</div>
                            <div className="text-xs text-muted-foreground">{toolMetadata[tool]?.description}</div>
                          </div>
                        </label>
                      ))}
                    </CollapsibleContent>
                  </Collapsible>
                ))}
              </div>
            </ScrollArea>
            <div className="flex items-center justify-between pt-4">
              <p className="text-sm text-muted-foreground">
                Selected: {Object.values(configureDialog?.tools || {}).filter(Boolean).length} tools
              </p>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  onClick={() => {
                    if (configureDialog) {
                      openAdvancedDialog(configureDialog.client);
                      setConfigureDialog(null);
                    }
                  }}
                >
                  View Advanced Config
                </Button>
                <Button variant="outline" onClick={() => setConfigureDialog(null)}>
                  Cancel
                </Button>
                <Button onClick={applyConfigureDialog}>
                  Apply
                </Button>
              </div>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* Advanced Configuration Dialog */}
      <Dialog open={!!configDialog} onOpenChange={(open) => !open && setConfigDialog(null)}>
        <DialogContent className="max-w-6xl w-[90vw] h-[80vh] flex flex-col">
          <DialogHeader>
            <DialogTitle>Advanced Configuration: {configDialog?.client}</DialogTitle>
            <DialogDescription>
              Review the configuration changes. New servers are highlighted in green.
            </DialogDescription>
          </DialogHeader>

          <div className="flex-1 overflow-hidden">
            <div className="h-full flex gap-4">
              <div className="flex-1 flex flex-col">
                <div className="text-sm font-medium mb-2">Current Configuration</div>
                <ScrollArea className="flex-1 border rounded-lg">
                  <pre className="p-3 text-xs bg-muted font-mono">
                    {JSON.stringify(configDialog?.current || {}, null, 2)}
                  </pre>
                </ScrollArea>
              </div>
              <div className="flex-1 flex flex-col">
                <div className="text-sm font-medium mb-2">Merged Configuration (Preview)</div>
                <ScrollArea className="flex-1 border rounded-lg">
                  <div className="p-3 text-xs font-mono">
                    {configDialog?.preview && (
                      <ConfigurationDisplay 
                        current={configDialog.current} 
                        preview={configDialog.preview}
                      />
                    )}
                  </div>
                </ScrollArea>
                <div className="mt-4 flex justify-end gap-2">
                  <Button variant="outline" onClick={() => setConfigDialog(null)}>
                    Cancel
                  </Button>
                  <Button onClick={applyAdvancedDialog} disabled={loading}>
                    {loading ? "Applying..." : "Apply Configuration"}
                  </Button>
                </div>
              </div>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}