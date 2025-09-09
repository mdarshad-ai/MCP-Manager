import React from "react";
import { 
  CheckCircle, 
  AlertTriangle, 
  Edit2, 
  Eye, 
  EyeOff, 
  Clock, 
  Shield, 
  Zap,
  Settings,
  Key,
  Globe,
  Database
} from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Separator } from "@/components/ui/separator";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { type ExternalServerProvider } from "../../api";
import { AdvancedConfig } from "./ConfigurationStep";
import { TestResult } from "./ConnectionTestStep";

export type SummaryStepProps = {
  provider: ExternalServerProvider;
  serverName: string;
  config: Record<string, string>;
  advancedConfig: AdvancedConfig;
  testResult: TestResult | null;
  onServerNameChange: (name: string) => void;
  onEditStep: (step: number) => void;
};

export function SummaryStep({
  provider,
  serverName,
  config,
  advancedConfig,
  testResult,
  onServerNameChange,
  onEditStep,
}: SummaryStepProps) {
  const [showSensitiveData, setShowSensitiveData] = React.useState(false);

  const requiredFields = provider.configFields.filter(field => field.required);
  const optionalFields = provider.configFields.filter(field => !field.required);
  const configuredOptionalFields = optionalFields.filter(field => config[field.key]?.trim());

  const isSensitiveField = (field: any) => {
    return field.type === "password" || 
           field.key.toLowerCase().includes("token") ||
           field.key.toLowerCase().includes("key") ||
           field.key.toLowerCase().includes("secret");
  };

  const maskValue = (value: string, field: any) => {
    if (!isSensitiveField(field) || showSensitiveData) return value;
    return "•".repeat(Math.min(value.length, 12));
  };

  const getFieldIcon = (field: any) => {
    if (field.type === "password" || field.key.toLowerCase().includes("password")) {
      return <Shield className="h-4 w-4 text-red-500" />;
    }
    if (field.key.toLowerCase().includes("key") || field.key.toLowerCase().includes("token")) {
      return <Key className="h-4 w-4 text-blue-500" />;
    }
    if (field.type === "url" || field.key.toLowerCase().includes("url")) {
      return <Globe className="h-4 w-4 text-green-500" />;
    }
    if (field.key.toLowerCase().includes("database") || field.key.toLowerCase().includes("db")) {
      return <Database className="h-4 w-4 text-purple-500" />;
    }
    return <Settings className="h-4 w-4 text-gray-500" />;
  };

  const getConfigurationScore = () => {
    const requiredScore = (requiredFields.filter(f => config[f.key]?.trim()).length / requiredFields.length) * 70;
    const optionalScore = optionalFields.length > 0 
      ? (configuredOptionalFields.length / optionalFields.length) * 20 
      : 20;
    const testScore = testResult?.success ? 10 : 0;
    return Math.round(requiredScore + optionalScore + testScore);
  };

  const getScoreColor = (score: number) => {
    if (score >= 90) return "text-green-600";
    if (score >= 70) return "text-yellow-600";
    return "text-red-600";
  };

  const getScoreLabel = (score: number) => {
    if (score >= 90) return "Excellent";
    if (score >= 70) return "Good";
    if (score >= 50) return "Fair";
    return "Needs Attention";
  };

  const configScore = getConfigurationScore();

  return (
    <div className="space-y-6">
      <div className="text-center">
        <h2 className="text-2xl font-semibold mb-2">Review & Confirm</h2>
        <p className="text-muted-foreground">
          Review your configuration before creating the remote MCP server
        </p>
      </div>

      {/* Configuration Score */}
      <Card className={`${configScore >= 70 ? 'bg-green-50 border-green-200' : 'bg-amber-50 border-amber-200'}`}>
        <CardContent className="pt-6">
          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium text-lg">Configuration Score</p>
              <p className="text-sm text-muted-foreground">
                Based on completeness and testing
              </p>
            </div>
            <div className="text-right">
              <div className={`text-3xl font-bold ${getScoreColor(configScore)}`}>
                {configScore}%
              </div>
              <Badge 
                variant="secondary" 
                className={`${
                  configScore >= 90 ? 'bg-green-100 text-green-800' :
                  configScore >= 70 ? 'bg-yellow-100 text-yellow-800' :
                  'bg-red-100 text-red-800'
                }`}
              >
                {getScoreLabel(configScore)}
              </Badge>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Server Details */}
      <Card>
        <CardHeader className="pb-4">
          <div className="flex items-center justify-between">
            <CardTitle className="text-lg">Server Details</CardTitle>
            <Button variant="ghost" size="sm" onClick={() => onEditStep(0)}>
              <Edit2 className="h-4 w-4 mr-2" />
              Edit
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex items-center gap-4">
            {provider.icon && <span className="text-2xl">{provider.icon}</span>}
            <div className="flex-1">
              <h3 className="font-semibold text-lg">{provider.name}</h3>
              <p className="text-sm text-muted-foreground">{provider.description}</p>
            </div>
          </div>

          <div className="space-y-2">
            <Label htmlFor="final-server-name">Server Name</Label>
            <Input
              id="final-server-name"
              value={serverName}
              onChange={(e) => onServerNameChange(e.target.value)}
              placeholder="Enter a name for this server"
            />
            <p className="text-xs text-muted-foreground">
              This name will be used to identify the server in your MCP Manager
            </p>
          </div>
        </CardContent>
      </Card>

      {/* Configuration Summary */}
      <Card>
        <CardHeader className="pb-4">
          <div className="flex items-center justify-between">
            <CardTitle className="text-lg">Configuration</CardTitle>
            <div className="flex gap-2">
              <Button 
                variant="ghost" 
                size="sm" 
                onClick={() => setShowSensitiveData(!showSensitiveData)}
              >
                {showSensitiveData ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </Button>
              <Button variant="ghost" size="sm" onClick={() => onEditStep(1)}>
                <Edit2 className="h-4 w-4 mr-2" />
                Edit
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Required Configuration */}
          <div>
            <h4 className="font-medium mb-3 flex items-center gap-2">
              <Shield className="h-4 w-4 text-red-500" />
              Required Fields ({requiredFields.filter(f => config[f.key]?.trim()).length}/{requiredFields.length})
            </h4>
            <div className="grid gap-3">
              {requiredFields.map((field) => {
                const value = config[field.key];
                const isConfigured = Boolean(value?.trim());
                
                return (
                  <div key={field.key} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                    <div className="flex items-center gap-3">
                      {getFieldIcon(field)}
                      <div>
                        <p className="font-medium text-sm">{field.label}</p>
                        {isConfigured && (
                          <p className="text-xs text-muted-foreground font-mono">
                            {maskValue(value, field)}
                          </p>
                        )}
                      </div>
                    </div>
                    <Badge 
                      variant={isConfigured ? "default" : "destructive"}
                      className="text-xs"
                    >
                      {isConfigured ? "Configured" : "Missing"}
                    </Badge>
                  </div>
                );
              })}
            </div>
          </div>

          {/* Optional Configuration */}
          {configuredOptionalFields.length > 0 && (
            <>
              <Separator />
              <div>
                <h4 className="font-medium mb-3 flex items-center gap-2">
                  <Settings className="h-4 w-4 text-blue-500" />
                  Optional Fields ({configuredOptionalFields.length}/{optionalFields.length})
                </h4>
                <div className="grid gap-3">
                  {configuredOptionalFields.map((field) => (
                    <div key={field.key} className="flex items-center justify-between p-3 bg-gray-50 rounded-lg">
                      <div className="flex items-center gap-3">
                        {getFieldIcon(field)}
                        <div>
                          <p className="font-medium text-sm">{field.label}</p>
                          <p className="text-xs text-muted-foreground font-mono">
                            {maskValue(config[field.key], field)}
                          </p>
                        </div>
                      </div>
                      <Badge variant="secondary" className="text-xs">
                        Configured
                      </Badge>
                    </div>
                  ))}
                </div>
              </div>
            </>
          )}
        </CardContent>
      </Card>

      {/* Advanced Settings Summary */}
      <Card>
        <CardHeader className="pb-4">
          <div className="flex items-center justify-between">
            <CardTitle className="text-lg">Advanced Settings</CardTitle>
            <Button variant="ghost" size="sm" onClick={() => onEditStep(2)}>
              <Edit2 className="h-4 w-4 mr-2" />
              Edit
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div className="text-center p-3 bg-gray-50 rounded-lg">
              <Clock className="h-4 w-4 mx-auto mb-2 text-blue-500" />
              <p className="text-sm font-medium">Timeout</p>
              <p className="text-xs text-muted-foreground">{advancedConfig.timeout / 1000}s</p>
            </div>
            <div className="text-center p-3 bg-gray-50 rounded-lg">
              <Shield className="h-4 w-4 mx-auto mb-2 text-green-500" />
              <p className="text-sm font-medium">Retries</p>
              <p className="text-xs text-muted-foreground">{advancedConfig.retryAttempts}</p>
            </div>
            <div className="text-center p-3 bg-gray-50 rounded-lg">
              <Zap className="h-4 w-4 mx-auto mb-2 text-yellow-500" />
              <p className="text-sm font-medium">Max Requests</p>
              <p className="text-xs text-muted-foreground">{advancedConfig.maxConcurrentRequests}</p>
            </div>
            <div className="text-center p-3 bg-gray-50 rounded-lg">
              <Settings className="h-4 w-4 mx-auto mb-2 text-purple-500" />
              <p className="text-sm font-medium">Cache</p>
              <p className="text-xs text-muted-foreground">
                {advancedConfig.enableCache ? 'Enabled' : 'Disabled'}
              </p>
            </div>
          </div>

          {Object.keys(advancedConfig.customHeaders).length > 0 && (
            <div className="mt-4 pt-4 border-t">
              <h4 className="font-medium mb-2 text-sm">Custom Headers</h4>
              <div className="space-y-2">
                {Object.entries(advancedConfig.customHeaders).map(([key, value]) => (
                  <div key={key} className="flex justify-between text-xs font-mono p-2 bg-gray-50 rounded">
                    <span className="font-medium">{key}:</span>
                    <span className="text-muted-foreground">{value}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Connection Test Results */}
      {testResult && (
        <Card className={testResult.success ? 'bg-green-50 border-green-200' : 'bg-red-50 border-red-200'}>
          <CardHeader className="pb-4">
            <div className="flex items-center justify-between">
              <CardTitle className={`text-lg flex items-center gap-2 ${
                testResult.success ? 'text-green-900' : 'text-red-900'
              }`}>
                {testResult.success ? (
                  <CheckCircle className="h-5 w-5 text-green-600" />
                ) : (
                  <AlertTriangle className="h-5 w-5 text-red-600" />
                )}
                Connection Test Results
              </CardTitle>
              <Button variant="ghost" size="sm" onClick={() => onEditStep(3)}>
                <Edit2 className="h-4 w-4 mr-2" />
                Retest
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <div className="flex items-center justify-between">
              <div>
                <p className={`font-medium ${testResult.success ? 'text-green-700' : 'text-red-700'}`}>
                  {testResult.success ? 'Connection Successful' : 'Connection Failed'}
                </p>
                {testResult.message && (
                  <p className={`text-sm ${testResult.success ? 'text-green-600' : 'text-red-600'}`}>
                    {testResult.message}
                  </p>
                )}
              </div>
              <div className="text-right">
                {testResult.responseTime && (
                  <div>
                    <p className="font-semibold">{testResult.responseTime}ms</p>
                    <p className="text-xs text-muted-foreground">Response time</p>
                  </div>
                )}
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Pre-Creation Checks */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Pre-Creation Checks</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            <div className="flex items-center gap-3">
              {requiredFields.every(f => config[f.key]?.trim()) ? (
                <CheckCircle className="h-4 w-4 text-green-600" />
              ) : (
                <AlertTriangle className="h-4 w-4 text-red-600" />
              )}
              <div>
                <p className="font-medium text-sm">Required Configuration</p>
                <p className="text-xs text-muted-foreground">
                  {requiredFields.filter(f => config[f.key]?.trim()).length}/{requiredFields.length} required fields configured
                </p>
              </div>
            </div>

            <div className="flex items-center gap-3">
              {serverName.trim() ? (
                <CheckCircle className="h-4 w-4 text-green-600" />
              ) : (
                <AlertTriangle className="h-4 w-4 text-red-600" />
              )}
              <div>
                <p className="font-medium text-sm">Server Name</p>
                <p className="text-xs text-muted-foreground">
                  {serverName.trim() ? 'Server name provided' : 'Please provide a server name'}
                </p>
              </div>
            </div>

            <div className="flex items-center gap-3">
              {testResult?.success ? (
                <CheckCircle className="h-4 w-4 text-green-600" />
              ) : testResult ? (
                <AlertTriangle className="h-4 w-4 text-yellow-600" />
              ) : (
                <AlertTriangle className="h-4 w-4 text-gray-400" />
              )}
              <div>
                <p className="font-medium text-sm">Connection Test</p>
                <p className="text-xs text-muted-foreground">
                  {testResult?.success 
                    ? 'Connection test passed' 
                    : testResult 
                      ? 'Connection test failed (you can still proceed)'
                      : 'No connection test performed'
                  }
                </p>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Final Warnings */}
      {(!testResult || !testResult.success) && (
        <Alert variant="destructive">
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>
            {!testResult 
              ? 'Connection has not been tested. Consider testing before creating the server.'
              : 'Connection test failed. The server may not work correctly until the configuration is fixed.'
            }
          </AlertDescription>
        </Alert>
      )}

      {configScore < 70 && (
        <Alert>
          <AlertTriangle className="h-4 w-4" />
          <AlertDescription>
            Configuration score is below recommended levels. Consider reviewing and completing more fields for better reliability.
          </AlertDescription>
        </Alert>
      )}
    </div>
  );
}