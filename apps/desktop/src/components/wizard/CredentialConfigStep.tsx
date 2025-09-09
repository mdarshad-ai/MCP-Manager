import React from "react";
import { Eye, EyeOff, AlertCircle, Key, Lock, Globe, Hash } from "lucide-react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { type ExternalServerProvider } from "../../api";

export type CredentialConfigStepProps = {
  provider: ExternalServerProvider;
  config: Record<string, string>;
  validationErrors: Record<string, string>;
  onConfigChange: (key: string, value: string) => void;
};

export function CredentialConfigStep({
  provider,
  config,
  validationErrors,
  onConfigChange,
}: CredentialConfigStepProps) {
  const [showPasswords, setShowPasswords] = React.useState<Record<string, boolean>>({});
  
  const togglePasswordVisibility = (fieldKey: string) => {
    setShowPasswords(prev => ({
      ...prev,
      [fieldKey]: !prev[fieldKey],
    }));
  };

  const getFieldIcon = (fieldType: string, fieldKey: string) => {
    if (fieldType === "password") return <Lock className="h-4 w-4" />;
    if (fieldKey.toLowerCase().includes("key") || fieldKey.toLowerCase().includes("token")) return <Key className="h-4 w-4" />;
    if (fieldType === "url" || fieldKey.toLowerCase().includes("url") || fieldKey.toLowerCase().includes("endpoint")) return <Globe className="h-4 w-4" />;
    if (fieldType === "number" || fieldKey.toLowerCase().includes("port") || fieldKey.toLowerCase().includes("id")) return <Hash className="h-4 w-4" />;
    return null;
  };

  const getFieldDescription = (field: any) => {
    // Generate helpful descriptions based on field type and key
    if (field.type === "password") {
      if (field.key.toLowerCase().includes("token")) {
        return "Enter your API token or access token";
      }
      return "Enter your password (will be stored securely)";
    }
    
    if (field.type === "url") {
      return "Enter the full URL including protocol (https://)";
    }
    
    if (field.key.toLowerCase().includes("username") || field.key.toLowerCase().includes("user")) {
      return "Your account username or email";
    }
    
    if (field.key.toLowerCase().includes("database") || field.key.toLowerCase().includes("db")) {
      return "The database name to connect to";
    }
    
    if (field.key.toLowerCase().includes("port")) {
      return "The port number for the connection";
    }
    
    return field.placeholder || `Configure ${field.label.toLowerCase()}`;
  };

  const requiredFields = provider.configFields.filter(field => field.required);
  const optionalFields = provider.configFields.filter(field => !field.required);
  const hasRequiredFields = requiredFields.length > 0;
  const hasOptionalFields = optionalFields.length > 0;

  const renderField = (field: any) => (
    <div key={field.key} className="space-y-2">
      <Label htmlFor={field.key} className="flex items-center gap-2">
        {getFieldIcon(field.type, field.key)}
        {field.label}
        {field.required && <span className="text-red-500 text-sm">*</span>}
      </Label>
      
      <p className="text-xs text-muted-foreground">
        {getFieldDescription(field)}
      </p>
      
      {field.type === "select" ? (
        <Select
          value={config[field.key] || ""}
          onValueChange={(value) => onConfigChange(field.key, value)}
        >
          <SelectTrigger 
            className={validationErrors[field.key] ? "border-red-500 focus:border-red-500" : ""}
          >
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
        <div className="relative">
          <Input
            id={field.key}
            type={
              field.type === "password" && !showPasswords[field.key] 
                ? "password" 
                : field.type === "number" 
                  ? "number"
                  : "text"
            }
            value={config[field.key] || ""}
            onChange={(e) => onConfigChange(field.key, e.target.value)}
            placeholder={field.placeholder}
            className={`${validationErrors[field.key] ? "border-red-500 focus:border-red-500" : ""} ${
              field.type === "password" ? "pr-10" : ""
            }`}
            autoComplete={
              field.type === "password" ? "new-password" :
              field.key.toLowerCase().includes("username") ? "username" :
              field.key.toLowerCase().includes("email") ? "email" :
              "off"
            }
          />
          {field.type === "password" && (
            <button
              type="button"
              onClick={() => togglePasswordVisibility(field.key)}
              className="absolute right-3 top-1/2 transform -translate-y-1/2 text-gray-400 hover:text-gray-600 transition-colors"
              tabIndex={-1}
            >
              {showPasswords[field.key] ? (
                <EyeOff className="h-4 w-4" />
              ) : (
                <Eye className="h-4 w-4" />
              )}
            </button>
          )}
        </div>
      )}
      
      {validationErrors[field.key] && (
        <div className="flex items-center gap-2 text-xs text-red-600">
          <AlertCircle className="h-3 w-3" />
          {validationErrors[field.key]}
        </div>
      )}

      {field.validation && !validationErrors[field.key] && (
        <div className="text-xs text-muted-foreground space-y-1">
          {field.validation.minLength && (
            <p>• Minimum {field.validation.minLength} characters</p>
          )}
          {field.validation.maxLength && (
            <p>• Maximum {field.validation.maxLength} characters</p>
          )}
          {field.validation.pattern && (
            <p>• Must match the required format</p>
          )}
        </div>
      )}
    </div>
  );

  return (
    <div className="space-y-6">
      <div className="text-center">
        <h2 className="text-2xl font-semibold mb-2">Configure Credentials</h2>
        <p className="text-muted-foreground">
          Set up the authentication details for {provider.name}
        </p>
      </div>

      {/* Security Notice */}
      <Alert>
        <Lock className="h-4 w-4" />
        <AlertDescription>
          Your credentials are stored securely and encrypted. Sensitive information like passwords and API tokens are protected using industry-standard encryption.
        </AlertDescription>
      </Alert>

      {/* Provider Overview */}
      <Card className="bg-blue-50 border-blue-200">
        <CardContent className="pt-4">
          <div className="flex items-center gap-3 mb-3">
            {provider.icon && <span className="text-lg">{provider.icon}</span>}
            <div>
              <p className="font-medium text-blue-900">{provider.name}</p>
              <p className="text-sm text-blue-700">{provider.description}</p>
            </div>
          </div>
          <div className="flex gap-2">
            {hasRequiredFields && (
              <Badge variant="destructive" className="text-xs">
                {requiredFields.length} Required
              </Badge>
            )}
            {hasOptionalFields && (
              <Badge variant="secondary" className="text-xs">
                {optionalFields.length} Optional
              </Badge>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Required Fields */}
      {hasRequiredFields && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg flex items-center gap-2">
              <AlertCircle className="h-5 w-5 text-red-500" />
              Required Configuration
            </CardTitle>
            <CardDescription>
              These fields are required to establish a connection
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {requiredFields.map(renderField)}
          </CardContent>
        </Card>
      )}

      {/* Optional Fields */}
      {hasOptionalFields && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Optional Configuration</CardTitle>
            <CardDescription>
              These fields can be configured to customize your connection
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {optionalFields.map(renderField)}
          </CardContent>
        </Card>
      )}

      {/* Progress Summary */}
      <Card className="bg-gray-50">
        <CardContent className="pt-4">
          <div className="flex justify-between items-center">
            <div>
              <p className="font-medium">Configuration Progress</p>
              <p className="text-sm text-muted-foreground">
                Fill in the required fields to continue
              </p>
            </div>
            <div className="text-right">
              <p className="text-lg font-semibold">
                {requiredFields.filter(field => config[field.key]?.trim()).length}/{requiredFields.length}
              </p>
              <p className="text-xs text-muted-foreground">Required fields</p>
            </div>
          </div>
          
          {/* Progress bar */}
          <div className="mt-3">
            <div className="w-full bg-gray-200 rounded-full h-2">
              <div 
                className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                style={{ 
                  width: `${requiredFields.length > 0 
                    ? (requiredFields.filter(field => config[field.key]?.trim()).length / requiredFields.length) * 100 
                    : 100}%` 
                }}
              />
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Tips */}
      <div className="bg-blue-50 rounded-lg p-4 space-y-2">
        <h4 className="font-medium text-blue-900">Configuration Tips</h4>
        <ul className="text-sm text-blue-700 space-y-1">
          <li>• Double-check your credentials to avoid connection issues</li>
          <li>• API keys and tokens are usually found in your service provider's settings</li>
          <li>• URLs should include the protocol (http:// or https://)</li>
          <li>• You can test the connection on the next step before finalizing</li>
        </ul>
      </div>
    </div>
  );
}