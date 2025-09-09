import React from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import { 
  Loader2, 
  TestTube, 
  CheckCircle, 
  XCircle, 
  AlertCircle,
  Eye,
  EyeOff,
  Search
} from "lucide-react";
import {
  createExternalServer,
  updateExternalServer,
  testExternalConnection,
  getProviderTemplates,
  type ExternalServerConfig,
  type ExternalServerProvider,
} from "../api";

type ExternalServerFormProps = {
  server?: ExternalServerConfig;
  onSuccess: (server: ExternalServerConfig) => void;
  onCancel: () => void;
};

type FormData = {
  name: string;
  providerId: string;
  config: Record<string, string>;
};

export function ExternalServerForm({ server, onSuccess, onCancel }: ExternalServerFormProps) {
  const [providers, setProviders] = React.useState<ExternalServerProvider[]>([]);
  const [selectedProvider, setSelectedProvider] = React.useState<ExternalServerProvider | null>(null);
  const [formData, setFormData] = React.useState<FormData>({
    name: server?.name || "",
    providerId: server?.provider || "",
    config: server?.config || {},
  });
  const [loading, setLoading] = React.useState(true);
  const [saving, setSaving] = React.useState(false);
  const [testing, setTesting] = React.useState(false);
  const [testResult, setTestResult] = React.useState<{
    success: boolean;
    message?: string;
    responseTime?: number;
  } | null>(null);
  const [error, setError] = React.useState<string | null>(null);
  const [providerSearch, setProviderSearch] = React.useState("");
  const [showPasswords, setShowPasswords] = React.useState<Record<string, boolean>>({});
  const [validationErrors, setValidationErrors] = React.useState<Record<string, string>>({});
  const [autoStart, setAutoStart] = React.useState(server?.autoStart || false);

  // Load provider templates
  React.useEffect(() => {
    const loadProviders = async () => {
      try {
        const data = await getProviderTemplates();
        setProviders(data);
        
        if (server) {
          // Find the provider for editing
          const provider = data.find(p => p.name === server.provider);
          if (provider) {
            setSelectedProvider(provider);
          }
        }
      } catch (err) {
        console.error("Failed to load provider templates:", err);
        setError("Failed to load provider templates");
      } finally {
        setLoading(false);
      }
    };

    loadProviders();
  }, [server]);

  const filteredProviders = React.useMemo(() => {
    if (!providerSearch) return providers;
    const search = providerSearch.toLowerCase();
    return providers.filter(
      provider => 
        provider.displayName.toLowerCase().includes(search) ||
        provider.description.toLowerCase().includes(search)
    );
  }, [providers, providerSearch]);

  const handleProviderSelect = (providerName: string) => {
    const provider = providers.find(p => p.name === providerName);
    if (!provider) return;

    setSelectedProvider(provider);
    setFormData(prev => ({
      ...prev,
      providerId: providerName,
      config: {}, // Reset config when changing provider
    }));
    setTestResult(null);
    setValidationErrors({});
  };

  const handleConfigChange = (key: string, value: string) => {
    setFormData(prev => ({
      ...prev,
      config: {
        ...prev.config,
        [key]: value,
      },
    }));
    
    // Clear validation error for this field
    if (validationErrors[key]) {
      setValidationErrors(prev => {
        const { [key]: _, ...rest } = prev;
        return rest;
      });
    }
  };

  const togglePasswordVisibility = (fieldKey: string) => {
    setShowPasswords(prev => ({
      ...prev,
      [fieldKey]: !prev[fieldKey],
    }));
  };

  const validateForm = (): boolean => {
    const errors: Record<string, string> = {};

    // Validate name
    if (!formData.name.trim()) {
      errors.name = "Name is required";
    }

    // Validate provider selection
    if (!selectedProvider) {
      errors.provider = "Please select a provider";
      setValidationErrors(errors);
      return false;
    }

    // Validate required config fields
    selectedProvider.credentials.forEach(field => {
      if (field.required && !formData.config[field.key]?.trim()) {
        errors[field.key] = `${field.displayName} is required`;
      }

      // Validate field-specific rules
      const value = formData.config[field.key];
      if (value && field.validation) {
        const regex = new RegExp(field.validation);
        if (!regex.test(value)) {
          errors[field.key] = `Invalid format for ${field.displayName}`;
        }
      }
    });

    setValidationErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleTest = async () => {
    if (!selectedProvider || !validateForm()) return;

    setTesting(true);
    setTestResult(null);

    try {
      const result = await testExternalConnection(server?.slug || 'test');
      setTestResult(result);
    } catch (err) {
      console.error("Test connection failed:", err);
      setTestResult({
        success: false,
        message: (err as Error).message,
      });
    } finally {
      setTesting(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    
    if (!validateForm() || !selectedProvider) return;

    setSaving(true);
    setError(null);

    try {
      const serverData = {
        name: formData.name.trim(),
        provider: selectedProvider.name,
        displayName: formData.name.trim(),
        config: formData.config,
        autoStart,
      };

      let result: ExternalServerConfig;
      if (server) {
        result = await updateExternalServer(server.slug, serverData);
      } else {
        result = await createExternalServer(serverData);
      }

      onSuccess(result);
    } catch (err) {
      console.error("Failed to save server:", err);
      setError((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center p-6">
        <div className="flex items-center gap-2">
          <Loader2 className="h-4 w-4 animate-spin" />
          Loading providers...
        </div>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {error && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      )}

      {/* Basic Information */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Basic Information</CardTitle>
          <CardDescription>
            Configure the basic settings for your remote MCP server.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="name">Server Name</Label>
            <Input
              id="name"
              value={formData.name}
              onChange={(e) => setFormData(prev => ({ ...prev, name: e.target.value }))}
              placeholder="Enter a name for this server"
              className={validationErrors.name ? "border-red-500" : ""}
            />
            {validationErrors.name && (
              <p className="text-xs text-red-600">{validationErrors.name}</p>
            )}
          </div>

          {/* Provider Information - Read-only for editing */}
          {server && selectedProvider && (
            <div className="space-y-2">
              <Label>Provider</Label>
              <div className="flex items-center gap-3 p-3 border rounded-lg bg-gray-50">
                {selectedProvider.icon && <span className="text-lg">{selectedProvider.icon}</span>}
                <div>
                  <h4 className="font-medium">{selectedProvider.displayName}</h4>
                  <p className="text-sm text-muted-foreground">{selectedProvider.description}</p>
                </div>
              </div>
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <span>Status:</span>
                {server.status?.state === "active" && (
                  <Badge variant="default" className="bg-green-100 text-green-800 border-green-200">
                    <CheckCircle className="h-3 w-3 mr-1" />
                    Connected
                  </Badge>
                )}
                {server.status?.state === "inactive" && (
                  <Badge variant="outline" className="bg-gray-100 text-gray-800 border-gray-200">
                    <XCircle className="h-3 w-3 mr-1" />
                    Disconnected
                  </Badge>
                )}
                {server.status?.state === "error" && (
                  <Badge variant="destructive" className="bg-red-100 text-red-800 border-red-200">
                    <AlertCircle className="h-3 w-3 mr-1" />
                    Error
                  </Badge>
                )}
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Provider Selection - Only for new servers */}
      {!server && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Provider Selection</CardTitle>
            <CardDescription>
              Choose the type of MCP server you want to connect to.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label>Search Providers</Label>
              <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  value={providerSearch}
                  onChange={(e) => setProviderSearch(e.target.value)}
                  placeholder="Search for provider..."
                  className="pl-9"
                />
              </div>
            </div>

            <div className="grid grid-cols-1 gap-3 max-h-40 overflow-y-auto">
              {filteredProviders.map((provider) => (
                <div
                  key={provider.name}
                  className={`p-3 border rounded-lg cursor-pointer transition-colors ${
                    selectedProvider?.name === provider.name
                      ? "border-blue-500 bg-blue-50"
                      : "border-gray-200 hover:border-gray-300"
                  }`}
                  onClick={() => handleProviderSelect(provider.name)}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3">
                      {provider.icon && <span className="text-lg">{provider.icon}</span>}
                      <div>
                        <h4 className="font-medium">{provider.displayName}</h4>
                        <p className="text-xs text-muted-foreground">{provider.description}</p>
                      </div>
                    </div>
                    {selectedProvider?.name === provider.name && (
                      <Badge variant="secondary">Selected</Badge>
                    )}
                  </div>
                </div>
              ))}
            </div>

            {validationErrors.provider && (
              <p className="text-xs text-red-600">{validationErrors.provider}</p>
            )}
          </CardContent>
        </Card>
      )}

      {/* Configuration Fields */}
      {selectedProvider && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Configuration</CardTitle>
            <CardDescription>
              Configure the connection settings for {selectedProvider.displayName}.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {selectedProvider.credentials?.map((field) => (
              <div key={field.key} className="space-y-2">
                <Label htmlFor={field.key} className="flex items-center gap-2">
                  {field.displayName}
                  {field.required && <span className="text-red-500">*</span>}
                </Label>
                
                <div className="relative">
                  <Input
                    id={field.key}
                    type={
                      field.secret && !showPasswords[field.key] 
                        ? "password" 
                        : "text"
                    }
                    value={formData.config[field.key] || ""}
                    onChange={(e) => handleConfigChange(field.key, e.target.value)}
                    placeholder={field.example}
                    className={validationErrors[field.key] ? "border-red-500" : ""}
                  />
                  {field.secret && (
                    <button
                      type="button"
                      onClick={() => togglePasswordVisibility(field.key)}
                      className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-gray-600"
                    >
                      {showPasswords[field.key] ? (
                        <EyeOff className="h-4 w-4" />
                      ) : (
                        <Eye className="h-4 w-4" />
                      )}
                    </button>
                  )}
                </div>
                
                {field.description && (
                  <p className="text-xs text-muted-foreground">{field.description}</p>
                )}
                
                {validationErrors[field.key] && (
                  <p className="text-xs text-red-600">{validationErrors[field.key]}</p>
                )}
              </div>
            ))}

            {/* Test Connection */}
            <div className="pt-4 border-t space-y-3">
              <div className="flex items-center justify-between">
                <div>
                  <h4 className="font-medium">Test Connection</h4>
                  <p className="text-xs text-muted-foreground">
                    Verify that the configuration works before saving
                  </p>
                </div>
                <Button
                  type="button"
                  variant="outline"
                  onClick={handleTest}
                  disabled={testing || !server}
                >
                  {testing ? (
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  ) : (
                    <TestTube className="h-4 w-4 mr-2" />
                  )}
                  Test
                </Button>
              </div>

              {testResult && (
                <div className={`p-3 rounded-md text-sm ${
                  testResult.success 
                    ? "bg-green-50 text-green-700 border border-green-200" 
                    : "bg-red-50 text-red-700 border border-red-200"
                }`}>
                  <div className="flex items-center gap-2">
                    {testResult.success ? (
                      <CheckCircle className="h-4 w-4" />
                    ) : (
                      <XCircle className="h-4 w-4" />
                    )}
                    <span className="font-medium">
                      {testResult.success ? "Connection successful!" : "Connection failed"}
                    </span>
                    {testResult.responseTime && (
                      <span className="text-xs">({testResult.responseTime}ms)</span>
                    )}
                  </div>
                  {testResult.message && (
                    <p className="mt-1 text-xs">{testResult.message}</p>
                  )}
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Advanced Settings */}
      {selectedProvider && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Advanced Settings</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-center space-x-2">
              <Checkbox 
                id="autoStart" 
                checked={autoStart}
                onCheckedChange={setAutoStart}
              />
              <Label htmlFor="autoStart" className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
                Auto-start with MCP Manager
              </Label>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Form Actions */}
      <div className="flex items-center justify-end gap-3 pt-4 border-t">
        <Button type="button" variant="outline" onClick={onCancel}>
          Cancel
        </Button>
        <Button type="submit" disabled={saving || !selectedProvider}>
          {saving ? (
            <Loader2 className="h-4 w-4 mr-2 animate-spin" />
          ) : null}
          {server ? "Save Changes" : "Add Server"}
        </Button>
      </div>
    </form>
  );
}