import React from "react";
import { CATEGORIES, type CatalogItem } from "../data/mcp-catalog";
import { loadCatalog } from "../data/catalog-loader";
import { Search, Download, Globe, Loader2, ExternalLink } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "../components/ui/card";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";
import { ScrollArea } from "../components/ui/scroll-area";
import { useToast } from "../hooks/use-toast";
import { installStart, installLogs, finalizeInstallation, createExternalServer } from "../api";

export function Marketplace() {
  const [query, setQuery] = React.useState("");
  const [category, setCategory] = React.useState<string>("all");
  const [busyId, setBusyId] = React.useState<string | null>(null);
  const [progress, setProgress] = React.useState<Record<string, string>>({});
  const { toast } = useToast();
  const [catalog, setCatalog] = React.useState<CatalogItem[]>([]);

  React.useEffect(() => {
    loadCatalog().then(setCatalog).catch(() => setCatalog([]));
  }, []);

  const items = React.useMemo(() => {
    return catalog.filter((c) => {
      const matchesCategory = category === "all" || c.category === category;
      const q = query.toLowerCase();
      const matchesQuery =
        !q ||
        c.name.toLowerCase().includes(q) ||
        c.slug.toLowerCase().includes(q) ||
        (c.tags || []).some((t) => t.toLowerCase().includes(q)) ||
        c.description.toLowerCase().includes(q);
      return matchesCategory && matchesQuery;
    });
  }, [catalog, query, category]);

  async function oneClickInstall(item: CatalogItem) {
    if (!item.install) return;
    try {
      setBusyId(item.slug);
      setProgress((p) => ({ ...p, [item.slug]: "Starting..." }));

      const res = await installStart({ type: item.install.type, uri: item.install.uri, slug: item.slug });

      let done = false;
      while (!done) {
        const s = await installLogs(res.id);
        setProgress((p) => ({ ...p, [item.slug]: s.logs.slice(-1)[0] || s.message || "" }));
        done = s.done;
        if (!done) await new Promise((r) => setTimeout(r, 1200));
        if (done && s.ok) {
          try { await finalizeInstallation(res.id); } catch {}
        }
      }

      toast({ title: item.name, description: "Installation completed" });
    } catch (err: any) {
      toast({ title: item.name, description: err?.message || "Installation failed", variant: "destructive" });
    } finally {
      setBusyId(null);
    }
  }

  async function oneClickAddRemote(item: CatalogItem) {
    if (!item.remote) return;
    try {
      setBusyId(item.slug);
      const server = await createExternalServer({
        slug: item.slug,
        name: item.name,
        provider: item.remote.provider,
        apiEndpoint: item.remote.apiEndpoint,
        authType: item.remote.authType || "api_key",
        config: {},
        status: { state: "connecting" } as any,
      } as any);
      toast({ title: item.name, description: `Remote MCP added: ${server.provider}` });
    } catch (err: any) {
      toast({ title: item.name, description: err?.message || "Failed to add remote server", variant: "destructive" });
    } finally {
      setBusyId(null);
    }
  }

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">MCP Marketplace</h1>
          <p className="text-muted-foreground">Discover, search, and one‑click install MCP servers.</p>
        </div>
      </div>

      <Card>
        <CardContent className="pt-6">
          <div className="flex gap-3 flex-col md:flex-row">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input placeholder="Search by name, tag, or description..." className="pl-9" value={query} onChange={(e) => setQuery(e.target.value)} />
            </div>
            <Select value={category} onValueChange={(v) => setCategory(v)}>
              <SelectTrigger className="w-full md:w-56">
                <SelectValue placeholder="Category" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Categories</SelectItem>
                {CATEGORIES.map((c) => (
                  <SelectItem key={c} value={c}>{c}</SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      <ScrollArea className="h-[calc(100vh-220px)] pr-1">
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          {items.map((item) => (
            <Card key={item.slug} className="overflow-hidden">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle className="text-base">{item.name}</CardTitle>
                    <CardDescription>{item.description}</CardDescription>
                  </div>
                  <Badge variant="outline">{item.category}</Badge>
                </div>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="flex flex-wrap gap-2">
                  {item.tags?.map((t) => (
                    <Badge key={t} variant="secondary" className="text-xs">{t}</Badge>
                  ))}
                </div>
                <div className="flex gap-2">
                  {item.remote && (
                    <Button variant="secondary" disabled={busyId === item.slug} onClick={() => oneClickAddRemote(item)}>
                      <Globe className="h-4 w-4 mr-2" />Add Remote
                    </Button>
                  )}
                  {item.install && (
                    <Button disabled={busyId === item.slug} onClick={() => oneClickInstall(item)}>
                      {busyId === item.slug ? <Loader2 className="h-4 w-4 mr-2 animate-spin" /> : <Download className="h-4 w-4 mr-2" />}Install
                    </Button>
                  )}
                  {item.repoUrl && (
                    <Button variant="ghost" asChild>
                      <a href={item.repoUrl} target="_blank" rel="noreferrer"><ExternalLink className="h-4 w-4 mr-2" />Repo</a>
                    </Button>
                  )}
                </div>
                {busyId === item.slug && (
                  <div className="text-xs text-muted-foreground flex items-center gap-2"><Loader2 className="h-3 w-3 animate-spin" />{progress[item.slug] || "Working..."}</div>
                )}
                {item.configExample && (
                  <details className="mt-1"><summary className="text-xs text-muted-foreground cursor-pointer">Config example</summary><pre className="text-xs bg-muted p-2 rounded overflow-auto">{item.configExample}</pre></details>
                )}
              </CardContent>
            </Card>
          ))}
        </div>
        {items.length === 0 && (<div className="text-center text-muted-foreground py-12">No results. Try a different search.</div>)}
      </ScrollArea>
    </div>
  );
}
