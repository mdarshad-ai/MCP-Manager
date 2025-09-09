import { Circle, Download, FileText, LayoutDashboard, Menu, Server, Settings, Users, Globe, ChevronDown, ChevronRight, ShoppingBag } from "lucide-react";
import React from "react";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { Separator } from "./ui/separator";
import { fetchExternalServers, type ExternalServerConfig } from "../api";

type LocalServerItem = {
  slug: string;
  name: string;
  status: "ready" | "degraded" | "down";
  type: "local";
};

type RemoteServerItem = {
  slug: string;
  name: string;
  status: "active" | "inactive" | "error" | "connecting";
  type: "remote";
  provider: string;
};

type ServerItem = LocalServerItem | RemoteServerItem;

function StatusDot({ status, type }: { status: string; type: "local" | "remote" }) {
  if (type === "local") {
    const colorClass = status === "ready" ? "text-green-500" : status === "degraded" ? "text-orange-500" : "text-red-500";
    return <Circle className={`h-2 w-2 ${colorClass} fill-current`} />;
  } else {
    const colorClass = status === "active" ? "text-green-500" : status === "error" ? "text-red-500" : status === "connecting" ? "text-blue-500" : "text-gray-500";
    return <Circle className={`h-2 w-2 ${colorClass} fill-current`} />;
  }
}

export function Sidebar({ 
  servers: localServers, 
  collapsed, 
  onToggleCollapse 
}: { 
  servers: Array<{slug: string; name: string; status: "ready" | "degraded" | "down"}>; 
  collapsed: boolean; 
  onToggleCollapse: () => void 
}) {
  const [currentHash, setCurrentHash] = React.useState(window.location.hash || "#/dashboard");
  const [remoteServers, setRemoteServers] = React.useState<ExternalServerConfig[]>([]);
  const [localExpanded, setLocalExpanded] = React.useState(true);
  const [remoteExpanded, setRemoteExpanded] = React.useState(true);

  React.useEffect(() => {
    const handleHashChange = () => setCurrentHash(window.location.hash || "#/dashboard");
    window.addEventListener("hashchange", handleHashChange);
    return () => window.removeEventListener("hashchange", handleHashChange);
  }, []);

  // Load remote servers
  React.useEffect(() => {
    const loadRemoteServers = async () => {
      try {
        const data = await fetchExternalServers();
        setRemoteServers(data);
      } catch (error) {
        console.error("Failed to fetch remote servers:", error);
      }
    };

    loadRemoteServers();
    // Refresh every 30 seconds
    const interval = setInterval(loadRemoteServers, 30000);
    return () => clearInterval(interval);
  }, []);

  const localServerItems: LocalServerItem[] = localServers.map(s => ({
    ...s,
    type: "local" as const
  }));

  const remoteServerItems: RemoteServerItem[] = remoteServers.map(s => ({
    slug: s.slug,
    name: s.name,
    status: s.status?.state || "inactive",
    type: "remote" as const,
    provider: s.provider
  }));

  const isActive = (path: string) => currentHash.startsWith(path);

  const NavButton = ({
    href,
    icon: Icon,
    children,
    className,
  }: {
    href: string;
    icon: any;
    children: React.ReactNode;
    className?: string;
  }) => (
    <Button
      variant={isActive(href) ? "secondary" : "ghost"}
      className={`w-full justify-start h-9 ${collapsed ? 'px-2' : 'px-3'} ${className}`}
      asChild
      title={collapsed ? children?.toString() : undefined}
    >
      <a href={href}>
        <Icon className={`h-4 w-4 ${collapsed ? '' : 'mr-2'} shrink-0`} />
        {!collapsed && <span className="truncate">{children}</span>}
      </a>
    </Button>
  );

  const ServerGroup = ({ 
    title, 
    servers, 
    expanded, 
    onToggle, 
    icon: Icon,
    type 
  }: { 
    title: string; 
    servers: ServerItem[]; 
    expanded: boolean; 
    onToggle: () => void;
    icon: any;
    type: "local" | "remote";
  }) => {
    if (collapsed) {
      // Show servers directly when collapsed
      return (
        <div className="space-y-1">
          {servers.map((s) => (
            <Button
              key={s.slug}
              variant={isActive(`#/server/${s.slug}`) ? "secondary" : "ghost"}
              className="w-full justify-center h-8 px-2"
              asChild
              title={s.name}
            >
              <a href={`#/server/${s.slug}`}>
                <StatusDot status={s.status} type={s.type} />
              </a>
            </Button>
          ))}
        </div>
      );
    }

    return (
      <div className="space-y-1">
        <Button
          variant="ghost"
          className="w-full justify-start h-8 px-3 text-sm font-medium text-muted-foreground hover:text-foreground"
          onClick={onToggle}
        >
          <Icon className="h-4 w-4 mr-2 shrink-0" />
          <span className="flex-1 text-left">{title}</span>
          {expanded ? (
            <ChevronDown className="h-3 w-3 shrink-0" />
          ) : (
            <ChevronRight className="h-3 w-3 shrink-0" />
          )}
        </Button>
        
        {expanded && (
          <div className="space-y-1 pl-6">
            {servers.map((s) => (
              <Button
                key={s.slug}
                variant={isActive(`#/server/${s.slug}`) ? "secondary" : "ghost"}
                className="w-full justify-start h-8 px-3 text-sm"
                asChild
              >
                <a href={`#/server/${s.slug}`}>
                  <StatusDot status={s.status} type={s.type} />
                  <span className="truncate ml-2">{s.name}</span>
                  {s.type === "remote" && (
                    <Badge variant="outline" className="ml-auto text-xs">
                      {(s as RemoteServerItem).provider}
                    </Badge>
                  )}
                </a>
              </Button>
            ))}
            {servers.length === 0 && (
              <div className="px-3 py-2 text-xs text-muted-foreground">
                No {type} servers
              </div>
            )}
          </div>
        )}
      </div>
    );
  };

  const totalServers = localServerItems.length + remoteServerItems.length;
  const activeServers = localServerItems.filter(s => s.status === "ready").length + 
                       remoteServerItems.filter(s => s.status === "active").length;

  return (
    <aside className={`${collapsed ? 'w-16' : 'w-64'} border-r bg-muted/30 transition-all duration-200`}>
      <div className="p-4 border-b">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Button 
              variant="ghost" 
              size="sm" 
              onClick={onToggleCollapse}
              className="h-8 w-8 p-0"
              title={collapsed ? "Expand sidebar" : "Collapse sidebar"}
            >
              <Menu className="h-4 w-4" />
            </Button>
            {!collapsed && <h1 className="text-lg font-semibold tracking-tight">MCP Manager</h1>}
          </div>
          {!collapsed && (
            <Badge variant="outline" className="text-xs">
              {activeServers}/{totalServers}
            </Badge>
          )}
        </div>
      </div>

      <nav className="p-3 space-y-2">
        <NavButton href="#/dashboard" icon={LayoutDashboard}>
          Dashboard
        </NavButton>
        <NavButton href="#/install" icon={Download}>
          Install
        </NavButton>
        <NavButton href="#/market" icon={ShoppingBag}>
          Marketplace
        <NavButton href="#/external" icon={Globe}>
          External
        </NavButton>
        </NavButton>

        <Separator className="my-3" />

        <ServerGroup
          title={`Local Servers (${localServerItems.length})`}
          servers={localServerItems}
          expanded={localExpanded}
          onToggle={() => setLocalExpanded(!localExpanded)}
          icon={Server}
          type="local"
        />

        <ServerGroup
          title={`Remote Servers (${remoteServerItems.length})`}
          servers={remoteServerItems}
          expanded={remoteExpanded}
          onToggle={() => setRemoteExpanded(!remoteExpanded)}
          icon={Globe}
          type="remote"
        />

        <Separator className="my-3" />

        <NavButton href="#/clients" icon={Users}>
          Clients
        </NavButton>
        <NavButton href="#/logs" icon={FileText}>
          Logs
        </NavButton>
        <NavButton href="#/settings" icon={Settings}>
          Settings
        </NavButton>
      </nav>
    </aside>
  );
}