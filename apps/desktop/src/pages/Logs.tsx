import { Activity, AlertCircle, Download, Loader2, Pause, Play, ScrollText, Search, Trash2 } from "lucide-react";
import React from "react";
import { createLogStreamEventSource, fetchLogTail, type LogEntry } from "../api";
import { Alert, AlertDescription } from "../components/ui/alert";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Input } from "../components/ui/input";
import { Label } from "../components/ui/label";
import { ScrollArea } from "../components/ui/scroll-area";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "../components/ui/select";

export function Logs({ slug }: { slug?: string }) {
  const [logs, setLogs] = React.useState<LogEntry[]>([]);
  const [staticContent, setStaticContent] = React.useState("");
  const [follow, setFollow] = React.useState(false);
  const [query, setQuery] = React.useState("");
  const [logLevel, setLogLevel] = React.useState<string>("all");
  const [maxLines, setMaxLines] = React.useState(1000);
  const [autoScroll, setAutoScroll] = React.useState(true);
  const [connectionStatus, setConnectionStatus] = React.useState<"disconnected" | "connecting" | "connected" | "error">(
    "disconnected",
  );
  const [error, setError] = React.useState<string | null>(null);
  const logContainerRef = React.useRef<HTMLDivElement>(null);
  const eventSourceRef = React.useRef<EventSource | null>(null);

  // Load initial static logs
  React.useEffect(() => {
    if (!slug) return;

    fetchLogTail(slug, 200)
      .then((content) => {
        setStaticContent(content);
        // Parse static logs into LogEntry format
        const entries: LogEntry[] = content
          .split("\n")
          .filter((line) => line.trim())
          .map((line, index) => ({
            timestamp: new Date().toISOString(),
            level: "info",
            message: line,
            source: slug,
          }));
        setLogs(entries);
      })
      .catch((err) => {
        setError(`Failed to load initial logs: ${err.message}`);
      });
  }, [slug]);

  // Handle real-time streaming
  React.useEffect(() => {
    if (!slug || !follow) {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
        setConnectionStatus("disconnected");
      }
      return;
    }

    setConnectionStatus("connecting");
    setError(null);

    const eventSource = createLogStreamEventSource(slug, logs.length);
    eventSourceRef.current = eventSource;

    eventSource.onopen = () => {
      setConnectionStatus("connected");
    };

    eventSource.onmessage = (event) => {
      try {
        const logEntry: LogEntry = JSON.parse(event.data);
        setLogs((prevLogs) => {
          const newLogs = [...prevLogs, logEntry];
          // Keep only the last maxLines entries
          return newLogs.slice(-maxLines);
        });
      } catch (err) {
        console.error("Failed to parse log entry:", err);
      }
    };

    eventSource.onerror = (err) => {
      console.error("Log stream error:", err);
      setConnectionStatus("error");
      setError("Connection to log stream lost. Trying to reconnect...");
    };

    return () => {
      eventSource.close();
      setConnectionStatus("disconnected");
    };
  }, [slug, follow, logs.length, maxLines]);

  // Auto-scroll to bottom when new logs arrive
  React.useEffect(() => {
    if (autoScroll && logContainerRef.current) {
      logContainerRef.current.scrollTop = logContainerRef.current.scrollHeight;
    }
  }, [logs, autoScroll]);

  const filteredLogs = React.useMemo(() => {
    return logs.filter((log) => {
      const levelMatch = logLevel === "all" || log.level === logLevel;
      const queryMatch =
        !query ||
        log.message.toLowerCase().includes(query.toLowerCase()) ||
        (log.source && log.source.toLowerCase().includes(query.toLowerCase()));
      return levelMatch && queryMatch;
    });
  }, [logs, logLevel, query]);

  const exportLogs = () => {
    const logText = filteredLogs
      .map((log) => `${log.timestamp} [${log.level.toUpperCase()}] ${log.message}`)
      .join("\n");

    const blob = new Blob([logText], { type: "text/plain" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${slug || "logs"}-${new Date().toISOString().split("T")[0]}.log`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  const clearLogs = () => {
    setLogs([]);
    setStaticContent("");
  };

  const getLogLevelColor = (level: string) => {
    switch (level.toLowerCase()) {
      case "error":
        return "text-red-600";
      case "warn":
      case "warning":
        return "text-orange-600";
      case "info":
        return "text-blue-600";
      case "debug":
        return "text-gray-600";
      default:
        return "text-gray-800";
    }
  };

  const formatTimestamp = (timestamp: string) => {
    return new Date(timestamp).toLocaleTimeString();
  };

  const getStatusBadgeVariant = (status: string) => {
    switch (status) {
      case "connected":
        return "default";
      case "connecting":
        return "secondary";
      case "error":
        return "destructive";
      default:
        return "outline";
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case "connected":
        return <Activity className="h-3 w-3" />;
      case "connecting":
        return <Loader2 className="h-3 w-3 animate-spin" />;
      case "error":
        return <AlertCircle className="h-3 w-3" />;
      default:
        return <ScrollText className="h-3 w-3" />;
    }
  };

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Logs</h1>
          {slug && <p className="text-muted-foreground">Server: {slug}</p>}
        </div>
        <div className="flex items-center gap-3">
          <Badge variant={getStatusBadgeVariant(connectionStatus) as any}>
            {getStatusIcon(connectionStatus)}
            <span className="ml-2">
              {connectionStatus === "connected"
                ? "Live"
                : connectionStatus === "connecting"
                  ? "Connecting"
                  : connectionStatus === "error"
                    ? "Disconnected"
                    : "Static"}
            </span>
          </Badge>
          <Badge variant="outline">{filteredLogs.length} lines</Badge>
        </div>
      </div>

      {error && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      <Card>
        <CardHeader className="pb-4">
          <CardTitle className="text-base">Log Controls</CardTitle>
          <CardDescription>Filter and manage log display settings</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex gap-3 items-center flex-wrap">
            <div className="flex gap-2 items-center flex-1 min-w-[300px]">
              <div className="relative flex-1">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  className="pl-9"
                  placeholder="Search logs..."
                  value={query}
                  onChange={(e) => setQuery(e.target.value)}
                />
              </div>
              <Select value={logLevel} onValueChange={setLogLevel}>
                <SelectTrigger className="w-[140px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Levels</SelectItem>
                  <SelectItem value="error">Error</SelectItem>
                  <SelectItem value="warn">Warning</SelectItem>
                  <SelectItem value="info">Info</SelectItem>
                  <SelectItem value="debug">Debug</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="flex gap-2">
              <Button
                variant={follow ? "default" : "outline"}
                size="sm"
                onClick={() => setFollow(!follow)}
                disabled={!slug}
              >
                {follow ? <Pause className="h-4 w-4 mr-2" /> : <Play className="h-4 w-4 mr-2" />}
                {follow ? "Pause" : "Follow"}
              </Button>

              <Button variant={autoScroll ? "default" : "outline"} size="sm" onClick={() => setAutoScroll(!autoScroll)}>
                <ScrollText className="h-4 w-4 mr-2" />
                Auto-scroll
              </Button>

              <Button variant="outline" size="sm" onClick={exportLogs} disabled={filteredLogs.length === 0}>
                <Download className="h-4 w-4 mr-2" />
                Export
              </Button>

              <Button variant="outline" size="sm" onClick={clearLogs}>
                <Trash2 className="h-4 w-4 mr-2" />
                Clear
              </Button>
            </div>

            <div className="flex gap-2 items-center">
              <Label className="text-sm whitespace-nowrap">Max lines:</Label>
              <Select value={maxLines.toString()} onValueChange={(value) => setMaxLines(Number(value))}>
                <SelectTrigger className="w-[100px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="100">100</SelectItem>
                  <SelectItem value="500">500</SelectItem>
                  <SelectItem value="1000">1000</SelectItem>
                  <SelectItem value="5000">5000</SelectItem>
                  <SelectItem value="10000">10000</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-3">
          <CardTitle className="text-base">Log Output</CardTitle>
          <CardDescription>Real-time server logs with syntax highlighting</CardDescription>
        </CardHeader>
        <CardContent className="p-0">
          <ScrollArea className="h-96 w-full border rounded-lg">
            <div ref={logContainerRef} className="p-3 text-xs bg-black text-green-400 font-mono min-h-full">
              {filteredLogs.length > 0 ? (
                filteredLogs.map((log, index) => (
                  <div key={index} className="flex gap-2 hover:bg-gray-800 px-1 -mx-1 rounded mb-1">
                    <span className="text-gray-500 text-xs shrink-0 w-20">{formatTimestamp(log.timestamp)}</span>
                    <span className={`text-xs shrink-0 w-12 uppercase ${getLogLevelColor(log.level)}`}>
                      [{log.level}]
                    </span>
                    <span className="text-gray-300 break-all">{log.message}</span>
                  </div>
                ))
              ) : (
                <div className="text-gray-500 text-center py-8">
                  {slug ? "No logs available" : "Select a server to view logs"}
                </div>
              )}
            </div>
          </ScrollArea>
        </CardContent>
      </Card>

      {/* Fallback to static content if no structured logs */}
      {logs.length === 0 && staticContent && (
        <Card>
          <CardHeader className="pb-3">
            <CardTitle className="text-base">Static Log Content</CardTitle>
            <CardDescription>Historical log data</CardDescription>
          </CardHeader>
          <CardContent className="p-0">
            <ScrollArea className="h-96 w-full border rounded-lg">
              <div className="p-3 text-xs bg-muted font-mono min-h-full whitespace-pre-wrap">
                {query
                  ? staticContent
                      .split("\n")
                      .filter((line) => line.toLowerCase().includes(query.toLowerCase()))
                      .join("\n")
                  : staticContent}
              </div>
            </ScrollArea>
          </CardContent>
        </Card>
      )}
    </div>
  );
}
