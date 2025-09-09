import React from "react";
import { Settings, Clock, Zap, Shield, AlertTriangle, Info, CheckCircle2 } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Separator } from "@/components/ui/separator";
import { type ExternalServerProvider } from "../../api";

export type AdvancedConfig = {
  timeout: number;
  retryAttempts: number;
  retryDelay: number;
  maxConcurrentRequests: number;
  enableLogging: boolean;
  logLevel: string;
  enableCache: boolean;
  cacheExpiry: number;
  customHeaders: Record<string, string>;
  environment: string;
};

export type ConfigurationStepProps = {
  provider: ExternalServerProvider;
  config: Record<string, string>;
  advancedConfig: AdvancedConfig;
  onConfigChange: (key: string, value: string) => void;
  onAdvancedConfigChange: (config: Partial<AdvancedConfig>) => void;
};

const DEFAULT_ADVANCED_CONFIG: AdvancedConfig = {
  timeout: 30000,
  retryAttempts: 3,
  retryDelay: 1000,
  maxConcurrentRequests: 10,
  enableLogging: true,
  logLevel: "info",
  enableCache: true,
  cacheExpiry: 300, // 5 minutes
  customHeaders: {},
  environment: "production",
};

export function ConfigurationStep({
  provider,
  config,
  advancedConfig = DEFAULT_ADVANCED_CONFIG,
  onConfigChange,
  onAdvancedConfigChange,
}: ConfigurationStepProps) {
  const [showAdvanced, setShowAdvanced] = React.useState(false);
  const [customHeaderKey, setCustomHeaderKey] = React.useState("");
  const [customHeaderValue, setCustomHeaderValue] = React.useState("");

  const optionalFields = provider.configFields.filter(field => !field.required);
  const hasOptionalFields = optionalFields.length > 0;

  const addCustomHeader = () => {
    if (customHeaderKey.trim() && customHeaderValue.trim()) {
      onAdvancedConfigChange({
        customHeaders: {
          ...advancedConfig.customHeaders,
          [customHeaderKey.trim()]: customHeaderValue.trim(),
        },
      });
      setCustomHeaderKey("");
      setCustomHeaderValue("");
    }
  };

  const removeCustomHeader = (key: string) => {
    const { [key]: _, ...rest } = advancedConfig.customHeaders;
    onAdvancedConfigChange({
      customHeaders: rest,
    });
  };

  const presetConfigurations = [
    {
      name: "High Performance",
      description: "Optimized for speed with lower timeout and higher concurrency",
      icon: <Zap className="h-4 w-4" />,
      config: {
        timeout: 10000,
        retryAttempts: 2,
        retryDelay: 500,
        maxConcurrentRequests: 20,
        enableCache: true,
        cacheExpiry: 600,
      },
    },
    {
      name: "Reliable",
      description: "Balanced settings for stable connections",
      icon: <Shield className="h-4 w-4" />,
      config: {
        timeout: 30000,
        retryAttempts: 3,
        retryDelay: 1000,
        maxConcurrentRequests: 10,
        enableCache: true,
        cacheExpiry: 300,
      },
    },
    {
      name: "Conservative",
      description: "Safe settings for unstable connections",
      icon: <Clock className="h-4 w-4" />,
      config: {
        timeout: 60000,
        retryAttempts: 5,
        retryDelay: 2000,
        maxConcurrentRequests: 5,
        enableCache: false,
        cacheExpiry: 0,
      },
    },
  ];

  const applyPreset = (presetConfig: any) => {
    onAdvancedConfigChange(presetConfig);
  };

  const getConfigurationSummary = () => {
    const requiredConfigured = provider.configFields
      .filter(field => field.required)
      .every(field => config[field.key]?.trim());
    
    const optionalConfigured = optionalFields
      .filter(field => config[field.key]?.trim()).length;

    return {
      requiredConfigured,
      optionalConfigured,
      totalOptional: optionalFields.length,
    };
  };

  const summary = getConfigurationSummary();

  return (
    <div className="space-y-6">
      <div className="text-center">
        <h2 className="text-2xl font-semibold mb-2">Advanced Configuration</h2>
        <p className="text-muted-foreground">
          Fine-tune your connection settings and configure optional parameters
        </p>
      </div>

      {/* Configuration Status */}
      <Card className={`${summary.requiredConfigured ? 'bg-green-50 border-green-200' : 'bg-amber-50 border-amber-200'}`}>
        <CardContent className="pt-4">
          <div className="flex items-center gap-3">
            {summary.requiredConfigured ? (
              <CheckCircle2 className="h-5 w-5 text-green-600" />
            ) : (
              <AlertTriangle className="h-5 w-5 text-amber-600" />
            )}
            <div>
              <p className={`font-medium ${summary.requiredConfigured ? 'text-green-900' : 'text-amber-900'}`}>
                {summary.requiredConfigured ? 'Ready to Connect' : 'Configuration Incomplete'}
              </p>
              <p className={`text-sm ${summary.requiredConfigured ? 'text-green-700' : 'text-amber-700'}`}>
                {summary.requiredConfigured 
                  ? `All required fields configured. ${summary.optionalConfigured}/${summary.totalOptional} optional fields set.`
                  : 'Please complete the credential configuration before proceeding.'
                }
              </p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Optional Provider Fields */}
      {hasOptionalFields && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              <Settings className="h-5 w-5" />
              Optional Provider Settings
            </CardTitle>
            <CardDescription>
              Configure additional settings specific to {provider.name}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {optionalFields.map((field) => (
              <div key={field.key} className="space-y-2">
                <Label htmlFor={field.key}>{field.label}</Label>
                <p className="text-xs text-muted-foreground">
                  {field.placeholder || `Optional: Configure ${field.label.toLowerCase()}`}
                </p>
                
                {field.type === "select" ? (
                  <Select
                    value={config[field.key] || ""}
                    onValueChange={(value) => onConfigChange(field.key, value)}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder={field.placeholder || `Select ${field.label}`} />
                    </SelectTrigger>
                    <SelectContent>
                      {field.options?.map((option) => (
                        <SelectItem key={option.value} value={option.value}>
                          {option.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                ) : (
                  <Input
                    id={field.key}
                    type={field.type === "number" ? "number" : "text"}
                    value={config[field.key] || ""}
                    onChange={(e) => onConfigChange(field.key, e.target.value)}
                    placeholder={field.placeholder}
                  />
                )}
              </div>
            ))}
          </CardContent>
        </Card>
      )}

      {/* Performance Presets */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Performance Presets</CardTitle>
          <CardDescription>
            Choose a preset configuration optimized for your use case
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            {presetConfigurations.map((preset) => (
              <Card
                key={preset.name}
                className="cursor-pointer hover:bg-gray-50 transition-colors"
                onClick={() => applyPreset(preset.config)}
              >
                <CardContent className="pt-4">
                  <div className="flex items-center gap-2 mb-2">
                    {preset.icon}
                    <h4 className="font-medium">{preset.name}</h4>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    {preset.description}
                  </p>
                </CardContent>
              </Card>
            ))}
          </div>
        </CardContent>
      </Card>

      {/* Advanced Settings Toggle */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-lg">Advanced Settings</CardTitle>
              <CardDescription>
                Configure timeout, retry behavior, and other advanced options
              </CardDescription>
            </div>
            <Switch
              checked={showAdvanced}
              onCheckedChange={setShowAdvanced}
            />
          </div>
        </CardHeader>

        {showAdvanced && (
          <CardContent className="space-y-6">
            {/* Connection Settings */}
            <div className="space-y-4">
              <h4 className="font-medium flex items-center gap-2">
                <Clock className="h-4 w-4" />
                Connection Settings
              </h4>
              
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="timeout">Connection Timeout (ms)</Label>
                  <Input
                    id="timeout"
                    type="number"
                    value={advancedConfig.timeout}
                    onChange={(e) => onAdvancedConfigChange({ timeout: parseInt(e.target.value) || 30000 })}
                    min="1000"
                    max="300000"
                  />
                  <p className="text-xs text-muted-foreground">
                    Maximum time to wait for a response
                  </p>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="maxRequests">Max Concurrent Requests</Label>
                  <Input
                    id="maxRequests"
                    type="number"
                    value={advancedConfig.maxConcurrentRequests}
                    onChange={(e) => onAdvancedConfigChange({ maxConcurrentRequests: parseInt(e.target.value) || 10 })}
                    min="1"
                    max="100"
                  />
                  <p className="text-xs text-muted-foreground">
                    Maximum number of simultaneous requests
                  </p>
                </div>
              </div>
            </div>

            <Separator />

            {/* Retry Settings */}
            <div className="space-y-4">
              <h4 className="font-medium flex items-center gap-2">
                <Shield className="h-4 w-4" />
                Retry Settings
              </h4>
              
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="retryAttempts">Retry Attempts</Label>
                  <Input
                    id="retryAttempts"
                    type="number"
                    value={advancedConfig.retryAttempts}
                    onChange={(e) => onAdvancedConfigChange({ retryAttempts: parseInt(e.target.value) || 3 })}
                    min="0"
                    max="10"
                  />
                  <p className="text-xs text-muted-foreground">
                    Number of retry attempts on failure
                  </p>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="retryDelay">Retry Delay (ms)</Label>
                  <Input
                    id="retryDelay"
                    type="number"
                    value={advancedConfig.retryDelay}
                    onChange={(e) => onAdvancedConfigChange({ retryDelay: parseInt(e.target.value) || 1000 })}
                    min="100"
                    max="10000"
                  />
                  <p className="text-xs text-muted-foreground">
                    Delay between retry attempts
                  </p>
                </div>
              </div>
            </div>

            <Separator />

            {/* Logging Settings */}
            <div className="space-y-4">
              <h4 className="font-medium">Logging & Debugging</h4>
              
              <div className="flex items-center justify-between">
                <div>
                  <Label htmlFor="enableLogging">Enable Logging</Label>
                  <p className="text-xs text-muted-foreground">
                    Log connection attempts and responses
                  </p>
                </div>
                <Switch
                  id="enableLogging"
                  checked={advancedConfig.enableLogging}
                  onCheckedChange={(checked) => onAdvancedConfigChange({ enableLogging: checked })}
                />
              </div>

              {advancedConfig.enableLogging && (
                <div className="space-y-2">
                  <Label htmlFor="logLevel">Log Level</Label>
                  <Select
                    value={advancedConfig.logLevel}
                    onValueChange={(value) => onAdvancedConfigChange({ logLevel: value })}
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="debug">Debug (Verbose)</SelectItem>
                      <SelectItem value="info">Info (Standard)</SelectItem>
                      <SelectItem value="warn">Warning (Important)</SelectItem>
                      <SelectItem value="error">Error (Critical Only)</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              )}
            </div>

            <Separator />

            {/* Caching Settings */}
            <div className="space-y-4">
              <h4 className="font-medium">Caching</h4>
              
              <div className="flex items-center justify-between">
                <div>
                  <Label htmlFor="enableCache">Enable Response Caching</Label>
                  <p className="text-xs text-muted-foreground">
                    Cache responses to improve performance
                  </p>
                </div>
                <Switch
                  id="enableCache"
                  checked={advancedConfig.enableCache}
                  onCheckedChange={(checked) => onAdvancedConfigChange({ enableCache: checked })}
                />
              </div>

              {advancedConfig.enableCache && (
                <div className="space-y-2">
                  <Label htmlFor="cacheExpiry">Cache Expiry (seconds)</Label>
                  <Input
                    id="cacheExpiry"
                    type="number"
                    value={advancedConfig.cacheExpiry}
                    onChange={(e) => onAdvancedConfigChange({ cacheExpiry: parseInt(e.target.value) || 300 })}
                    min="0"
                    max="3600"
                  />
                  <p className="text-xs text-muted-foreground">
                    How long to cache responses (0 = no expiry)
                  </p>
                </div>
              )}
            </div>

            <Separator />

            {/* Custom Headers */}
            <div className="space-y-4">
              <h4 className="font-medium">Custom HTTP Headers</h4>
              <p className="text-sm text-muted-foreground">
                Add custom headers to all requests
              </p>
              
              <div className="space-y-3">
                <div className="grid grid-cols-2 gap-2">
                  <Input
                    placeholder="Header name"
                    value={customHeaderKey}
                    onChange={(e) => setCustomHeaderKey(e.target.value)}
                  />
                  <div className="flex gap-2">
                    <Input
                      placeholder="Header value"
                      value={customHeaderValue}
                      onChange={(e) => setCustomHeaderValue(e.target.value)}
                    />
                    <button
                      type="button"
                      onClick={addCustomHeader}
                      disabled={!customHeaderKey.trim() || !customHeaderValue.trim()}
                      className="px-3 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed text-sm"
                    >
                      Add
                    </button>
                  </div>
                </div>

                {Object.entries(advancedConfig.customHeaders).length > 0 && (
                  <div className="space-y-2">
                    {Object.entries(advancedConfig.customHeaders).map(([key, value]) => (
                      <div key={key} className="flex items-center justify-between p-2 bg-gray-50 rounded">
                        <div className="font-mono text-sm">
                          <span className="font-medium">{key}:</span> {value}
                        </div>
                        <button
                          type="button"
                          onClick={() => removeCustomHeader(key)}
                          className="text-red-600 hover:text-red-800 text-sm"
                        >
                          Remove
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </CardContent>
        )}
      </Card>

      {/* Configuration Warning */}
      <Alert>
        <Info className="h-4 w-4" />
        <AlertDescription>
          Advanced settings can significantly impact performance and reliability. 
          Use the preset configurations unless you have specific requirements.
        </AlertDescription>
      </Alert>
    </div>
  );
}