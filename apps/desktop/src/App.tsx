import React from "react";
import { fetchServers } from "./api";
import { Sidebar } from "./components/Sidebar";
import { ErrorBoundary } from "./components/ErrorBoundary";
import { Clients } from "./pages/Clients";
import { Dashboard } from "./pages/Dashboard";
import { Install } from "./pages/Install";
import { Marketplace } from "./pages/Marketplace";
import { Logs } from "./pages/Logs";
import { ServerDetails } from "./pages/ServerDetails";
import { Settings } from "./pages/Settings";
import { ExternalServers } from "./pages/ExternalServers";

export default function App() {
  const [route, setRoute] = React.useState(window.location.hash || "#/dashboard");
  const [servers, setServers] = React.useState<
    Array<{ slug: string; name: string; status: "ready" | "down" | "starting" }>
  >([]);
  const [sidebarCollapsed, setSidebarCollapsed] = React.useState(false);

  React.useEffect(() => {
    const onHash = () => setRoute(window.location.hash || "#/dashboard");
    window.addEventListener("hashchange", onHash);
    return () => window.removeEventListener("hashchange", onHash);
  }, []);

  // Fetch real server data
  React.useEffect(() => {
    const loadServers = async () => {
      try {
        const data = await fetchServers();
        const mappedServers = data.map((s: any) => ({
          slug: s.slug,
          name: s.name,
          status:
            s.state === "running"
              ? ("ready" as const)
              : s.state === "starting"
                ? ("starting" as const)
                : ("down" as const),
        }));
        setServers(mappedServers);
      } catch (error) {
        console.error("Failed to fetch servers:", error);
      }
    };

    loadServers();
    // Refresh every 5 seconds
    const interval = setInterval(loadServers, 5000);
    return () => clearInterval(interval);
  }, []);

  const currentSlug = React.useMemo(() => {
    const m = route.match(/^#\/(server|logs)\/([^?#]+)/);
    return m ? decodeURIComponent(m[2]) : undefined;
  }, [route]);

  return (
    <div className="flex min-h-screen">
      <Sidebar 
        servers={servers} 
        collapsed={sidebarCollapsed} 
        onToggleCollapse={() => setSidebarCollapsed(!sidebarCollapsed)} 
      />
      <main className="flex-1">
        <ErrorBoundary>
          {route.startsWith("#/dashboard") && (
            <Dashboard rows={servers.map((s) => ({ slug: s.slug, name: s.name, status: s.status }))} />
          )}
          {route.startsWith("#/market") && <Marketplace />}
          {route.startsWith("#/install") && <Install />}
          {route.startsWith("#/server/") && <ServerDetails slug={currentSlug!} />}
          {route.startsWith("#/clients") && <Clients />}
          {route.startsWith("#/logs") && <Logs slug={currentSlug} />}
          {route.startsWith("#/external") && <ExternalServers />}
          {route.startsWith("#/settings") && <Settings />}
        </ErrorBoundary>
      </main>
    </div>
  );
}
