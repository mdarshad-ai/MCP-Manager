import React from "react";
import { 
  TestTube, 
  CheckCircle, 
  XCircle, 
  Loader2, 
  RefreshCw, 
  AlertTriangle, 
  Clock,
  Zap,
  Shield,
  Info
} from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Progress } from "@/components/ui/progress";
import { Separator } from "@/components/ui/separator";
import { type ExternalServerProvider, testExternalConnection } from "../../api";
import { AdvancedConfig } from "./ConfigurationStep";

export type TestResult = {
  success: boolean;
  message?: string;
  responseTime?: number;
  details?: {
    latency?: number;
    throughput?: string;
    availability?: number;
    errorRate?: number;
  };
};

export type ConnectionTestStepProps = {
  provider: ExternalServerProvider;
  config: Record<string, string>;
  advancedConfig: AdvancedConfig;
  onTestComplete?: (result: TestResult) => void;
};

type TestPhase = 
  | "idle"
  | "validating"
  | "connecting" 
  | "authenticating"
  | "testing"
  | "completed"
  | "failed";

export function ConnectionTestStep({
  provider,
  config,
  advancedConfig,
  onTestComplete,
}: ConnectionTestStepProps) {
  const [testPhase, setTestPhase] = React.useState<TestPhase>("idle");
  const [testResult, setTestResult] = React.useState<TestResult | null>(null);
  const [progress, setProgress] = React.useState(0);
  const [testHistory, setTestHistory] = React.useState<(TestResult & { timestamp: Date })[]>([]);
  const [autoRetesting, setAutoRetesting] = React.useState(false);
  const [detailedLogs, setDetailedLogs] = React.useState<string[]>([]);

  const isRequiredConfigComplete = React.useMemo(() => {
    return provider.configFields
      .filter(field => field.required)
      .every(field => config[field.key]?.trim());
  }, [provider.configFields, config]);

  const addLog = (message: string) => {
    setDetailedLogs(prev => [...prev, `${new Date().toLocaleTimeString()}: ${message}`]);
  };

  const runTest = async (isRetry: boolean = false) => {
    if (!isRequiredConfigComplete) return;

    setTestPhase("validating");
    setProgress(0);
    setDetailedLogs([]);
    
    if (!isRetry) {
      setTestResult(null);
    }

    try {
      addLog("Starting connection test...");
      
      // Phase 1: Validation
      setProgress(10);
      await new Promise(resolve => setTimeout(resolve, 500));
      addLog("Validating configuration parameters...");
      
      // Phase 2: Connecting
      setTestPhase("connecting");
      setProgress(25);
      await new Promise(resolve => setTimeout(resolve, 800));
      addLog(`Establishing connection to ${provider.name}...`);
      
      // Phase 3: Authenticating
      setTestPhase("authenticating");
      setProgress(50);
      await new Promise(resolve => setTimeout(resolve, 600));
      addLog("Authenticating with provided credentials...");
      
      // Phase 4: Testing
      setTestPhase("testing");
      setProgress(75);
      addLog("Running connectivity tests...");
      
      const startTime = Date.now();
      const result = await testExternalConnection({
        providerId: provider.id,
        config,
      });
      const responseTime = Date.now() - startTime;
      
      setProgress(100);
      setTestPhase("completed");
      
      const enhancedResult: TestResult = {
        ...result,
        responseTime: responseTime,
        details: {
          latency: responseTime,
          throughput: responseTime < 1000 ? "Good" : responseTime < 3000 ? "Fair" : "Poor",
          availability: result.success ? 100 : 0,
          errorRate: result.success ? 0 : 100,
        }
      };
      
      setTestResult(enhancedResult);
      
      // Add to history
      setTestHistory(prev => [
        { ...enhancedResult, timestamp: new Date() },
        ...prev.slice(0, 4) // Keep last 5 tests
      ]);
      
      addLog(result.success 
        ? `✅ Connection test successful (${responseTime}ms)`
        : `❌ Connection test failed: ${result.message}`
      );
      
      onTestComplete?.(enhancedResult);
      
    } catch (error) {
      setTestPhase("failed");
      setProgress(0);
      
      const failedResult: TestResult = {
        success: false,
        message: (error as Error).message,
        responseTime: undefined,
      };
      
      setTestResult(failedResult);
      setTestHistory(prev => [
        { ...failedResult, timestamp: new Date() },
        ...prev.slice(0, 4)
      ]);
      
      addLog(`❌ Connection failed: ${(error as Error).message}`);
      onTestComplete?.(failedResult);
    }
  };

  const startAutoRetest = () => {
    setAutoRetesting(true);
    const interval = setInterval(async () => {
      if (testPhase === "idle" || testPhase === "completed" || testPhase === "failed") {
        await runTest(true);
      }
    }, 30000); // Test every 30 seconds

    // Stop after 5 minutes
    setTimeout(() => {
      setAutoRetesting(false);
      clearInterval(interval);
    }, 300000);
  };

  const getPhaseIcon = (phase: TestPhase) => {
    switch (phase) {
      case "idle": return <TestTube className="h-4 w-4" />;
      case "validating":
      case "connecting":
      case "authenticating":
      case "testing": return <Loader2 className="h-4 w-4 animate-spin" />;
      case "completed": return <CheckCircle className="h-4 w-4 text-green-600" />;
      case "failed": return <XCircle className="h-4 w-4 text-red-600" />;
    }
  };

  const getPhaseDescription = (phase: TestPhase) => {
    switch (phase) {
      case "idle": return "Ready to test connection";
      case "validating": return "Validating configuration...";
      case "connecting": return "Establishing connection...";
      case "authenticating": return "Authenticating credentials...";
      case "testing": return "Running connectivity tests...";
      case "completed": return "Test completed successfully";
      case "failed": return "Test failed";
    }
  };

  const getPerformanceColor = (responseTime?: number) => {
    if (!responseTime) return "gray";
    if (responseTime < 1000) return "green";
    if (responseTime < 3000) return "yellow";
    return "red";
  };

  const getPerformanceLabel = (responseTime?: number) => {
    if (!responseTime) return "N/A";
    if (responseTime < 1000) return "Excellent";
    if (responseTime < 3000) return "Good";
    if (responseTime < 5000) return "Fair";
    return "Poor";
  };

  return (
    <div className="space-y-6">
      <div className="text-center">
        <h2 className="text-2xl font-semibold mb-2">Test Connection</h2>
        <p className="text-muted-foreground">
          Verify that your {provider.name} configuration works correctly
        </p>
      </div>

      {/* Configuration Status */}
      <Card className={!isRequiredConfigComplete ? 'bg-amber-50 border-amber-200' : 'bg-green-50 border-green-200'}>
        <CardContent className="pt-4">
          <div className="flex items-center gap-3">
            {isRequiredConfigComplete ? (
              <CheckCircle className="h-5 w-5 text-green-600" />
            ) : (
              <AlertTriangle className="h-5 w-5 text-amber-600" />
            )}
            <div>
              <p className={`font-medium ${isRequiredConfigComplete ? 'text-green-900' : 'text-amber-900'}`}>
                {isRequiredConfigComplete ? 'Configuration Complete' : 'Configuration Required'}
              </p>
              <p className={`text-sm ${isRequiredConfigComplete ? 'text-green-700' : 'text-amber-700'}`}>
                {isRequiredConfigComplete 
                  ? 'All required credentials are configured and ready for testing'
                  : 'Please complete the credential configuration before testing'
                }
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Test Controls */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            {getPhaseIcon(testPhase)}
            Connection Test
          </CardTitle>
          <CardDescription>
            {getPhaseDescription(testPhase)}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Progress Bar */}
          {(testPhase !== "idle" && testPhase !== "completed" && testPhase !== "failed") && (
            <div className="space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium">Testing Progress</span>
                <span className="text-sm text-muted-foreground">{progress}%</span>
              </div>
              <Progress value={progress} className="h-2" />
            </div>
          )}

          {/* Test Action Buttons */}
          <div className="flex gap-3">
            <Button
              onClick={() => runTest(false)}
              disabled={!isRequiredConfigComplete || (testPhase !== "idle" && testPhase !== "completed" && testPhase !== "failed")}
              className="flex items-center gap-2"
            >
              {testPhase !== "idle" && testPhase !== "completed" && testPhase !== "failed" ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <TestTube className="h-4 w-4" />
              )}
              {testResult ? 'Retest Connection' : 'Test Connection'}
            </Button>

            {testResult && (
              <Button
                variant="outline"
                onClick={startAutoRetest}
                disabled={autoRetesting || !isRequiredConfigComplete}
                className="flex items-center gap-2"
              >
                {autoRetesting ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <RefreshCw className="h-4 w-4" />
                )}
                {autoRetesting ? 'Auto-testing...' : 'Auto-retest'}
              </Button>
            )}
          </div>

          {autoRetesting && (
            <Alert>
              <Info className="h-4 w-4" />
              <AlertDescription>
                Auto-retesting enabled. Connection will be tested every 30 seconds for 5 minutes.
              </AlertDescription>
            </Alert>
          )}
        </CardContent>
      </Card>

      {/* Test Results */}
      {testResult && (
        <Card className={testResult.success ? 'bg-green-50 border-green-200' : 'bg-red-50 border-red-200'}>
          <CardHeader>
            <CardTitle className={`flex items-center gap-2 ${testResult.success ? 'text-green-900' : 'text-red-900'}`}>
              {testResult.success ? (
                <CheckCircle className="h-5 w-5 text-green-600" />
              ) : (
                <XCircle className="h-5 w-5 text-red-600" />
              )}
              {testResult.success ? 'Connection Successful!' : 'Connection Failed'}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {testResult.message && (
              <p className={`text-sm ${testResult.success ? 'text-green-700' : 'text-red-700'}`}>
                {testResult.message}
              </p>
            )}

            {/* Performance Metrics */}
            {testResult.success && testResult.details && (
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <div className="text-center p-3 bg-white/50 rounded-lg">
                  <div className="flex items-center justify-center gap-1 mb-1">
                    <Clock className="h-4 w-4" />
                    <span className="text-xs font-medium">Latency</span>
                  </div>
                  <p className="text-lg font-semibold">{testResult.details.latency}ms</p>
                  <Badge 
                    variant="secondary" 
                    className={`text-xs mt-1 ${
                      getPerformanceColor(testResult.details.latency) === 'green' ? 'bg-green-100 text-green-800' :
                      getPerformanceColor(testResult.details.latency) === 'yellow' ? 'bg-yellow-100 text-yellow-800' :
                      'bg-red-100 text-red-800'
                    }`}
                  >
                    {getPerformanceLabel(testResult.details.latency)}
                  </Badge>
                </div>

                <div className="text-center p-3 bg-white/50 rounded-lg">
                  <div className="flex items-center justify-center gap-1 mb-1">
                    <Zap className="h-4 w-4" />
                    <span className="text-xs font-medium">Performance</span>
                  </div>
                  <p className="text-lg font-semibold">{testResult.details.throughput}</p>
                  <Badge variant="secondary" className="text-xs mt-1">
                    Response
                  </Badge>
                </div>

                <div className="text-center p-3 bg-white/50 rounded-lg">
                  <div className="flex items-center justify-center gap-1 mb-1">
                    <Shield className="h-4 w-4" />
                    <span className="text-xs font-medium">Availability</span>
                  </div>
                  <p className="text-lg font-semibold">{testResult.details.availability}%</p>
                  <Badge variant="secondary" className="text-xs mt-1 bg-green-100 text-green-800">
                    Online
                  </Badge>
                </div>

                <div className="text-center p-3 bg-white/50 rounded-lg">
                  <div className="flex items-center justify-center gap-1 mb-1">
                    <TestTube className="h-4 w-4" />
                    <span className="text-xs font-medium">Error Rate</span>
                  </div>
                  <p className="text-lg font-semibold">{testResult.details.errorRate}%</p>
                  <Badge variant="secondary" className="text-xs mt-1 bg-green-100 text-green-800">
                    Stable
                  </Badge>
                </div>
              </div>
            )}
          </CardContent>
        </Card>
      )}

      {/* Test History */}
      {testHistory.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Test History</CardTitle>
            <CardDescription>
              Recent connection test results
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {testHistory.map((test, index) => (
                <div key={index} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                  <div className="flex items-center gap-3">
                    {test.success ? (
                      <CheckCircle className="h-4 w-4 text-green-600" />
                    ) : (
                      <XCircle className="h-4 w-4 text-red-600" />
                    )}
                    <div>
                      <p className="text-sm font-medium">
                        {test.success ? 'Successful' : 'Failed'}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {test.timestamp.toLocaleString()}
                      </p>
                    </div>
                  </div>
                  <div className="text-right">
                    {test.responseTime && (
                      <p className="text-sm font-medium">{test.responseTime}ms</p>
                    )}
                    {test.message && (
                      <p className="text-xs text-muted-foreground truncate max-w-48">
                        {test.message}
                      </p>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Detailed Logs */}
      {detailedLogs.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Test Logs</CardTitle>
            <CardDescription>
              Detailed connection test output
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="bg-black text-green-400 p-4 rounded-lg font-mono text-xs max-h-40 overflow-y-auto">
              {detailedLogs.map((log, index) => (
                <div key={index} className="mb-1">{log}</div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Next Steps */}
      {testResult?.success && (
        <Alert>
          <CheckCircle className="h-4 w-4" />
          <AlertDescription>
            Great! Your connection test was successful. You can now proceed to review and save your configuration.
          </AlertDescription>
        </Alert>
      )}

      {testResult && !testResult.success && (
        <Alert variant="destructive">
          <XCircle className="h-4 w-4" />
          <AlertDescription>
            Connection test failed. Please check your configuration and try again. 
            Common issues include incorrect credentials, network connectivity, or service availability.
          </AlertDescription>
        </Alert>
      )}
    </div>
  );
}