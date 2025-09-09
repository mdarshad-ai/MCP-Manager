import React from "react";
import { ChevronLeft, ChevronRight, X, Loader2, CheckCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Progress } from "@/components/ui/progress";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { 
  getProviderTemplates, 
  createExternalServer, 
  type ExternalServerProvider,
  type ExternalServerConfig 
} from "../api";

import { ProviderSelectionStep } from "./wizard/ProviderSelectionStep";
import { CredentialConfigStep } from "./wizard/CredentialConfigStep";
import { ConfigurationStep, AdvancedConfig } from "./wizard/ConfigurationStep";
import { ConnectionTestStep, TestResult } from "./wizard/ConnectionTestStep";
import { SummaryStep } from "./wizard/SummaryStep";

type WizardStep = {
  id: number;
  title: string;
  description: string;
  component: React.ComponentType<any>;
  isOptional?: boolean;
};

const WIZARD_STEPS: WizardStep[] = [
  {
    id: 0,
    title: "Provider Selection",
    description: "Choose your MCP server type",
    component: ProviderSelectionStep,
  },
  {
    id: 1,
    title: "Credentials",
    description: "Configure authentication",
    component: CredentialConfigStep,
  },
  {
    id: 2,
    title: "Configuration",
    description: "Advanced settings",
    component: ConfigurationStep,
    isOptional: true,
  },
  {
    id: 3,
    title: "Test Connection",
    description: "Verify configuration",
    component: ConnectionTestStep,
    isOptional: true,
  },
  {
    id: 4,
    title: "Review & Confirm",
    description: "Final review",
    component: SummaryStep,
  },
];

const DEFAULT_ADVANCED_CONFIG: AdvancedConfig = {
  timeout: 30000,
  retryAttempts: 3,
  retryDelay: 1000,
  maxConcurrentRequests: 10,
  enableLogging: true,
  logLevel: "info",
  enableCache: true,
  cacheExpiry: 300,
  customHeaders: {},
  environment: "production",
};

export type ProviderSetupWizardProps = {
  open: boolean;
  onClose: () => void;
  onSuccess: (server: ExternalServerConfig) => void;
};

export function ProviderSetupWizard({ open, onClose, onSuccess }: ProviderSetupWizardProps) {
  // State management
  const [currentStep, setCurrentStep] = React.useState(0);
  const [providers, setProviders] = React.useState<ExternalServerProvider[]>([]);
  const [selectedProvider, setSelectedProvider] = React.useState<ExternalServerProvider | null>(null);
  const [serverName, setServerName] = React.useState("");
  const [config, setConfig] = React.useState<Record<string, string>>({});
  const [advancedConfig, setAdvancedConfig] = React.useState<AdvancedConfig>(DEFAULT_ADVANCED_CONFIG);
  const [validationErrors, setValidationErrors] = React.useState<Record<string, string>>({});
  const [testResult, setTestResult] = React.useState<TestResult | null>(null);
  
  // Loading states
  const [loading, setLoading] = React.useState(true);
  const [saving, setSaving] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  // Progress tracking
  const [completedSteps, setCompletedSteps] = React.useState<Set<number>>(new Set());
  const [visitedSteps, setVisitedSteps] = React.useState<Set<number>>(new Set([0]));

  // Load providers on mount
  React.useEffect(() => {
    if (open) {
      loadProviders();
    }
  }, [open]);

  // Reset state when opening/closing
  React.useEffect(() => {
    if (open) {
      resetWizard();
    }
  }, [open]);

  const loadProviders = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await getProviderTemplates();
      setProviders(data);
    } catch (err) {
      setError("Failed to load provider templates");
      console.error("Failed to load providers:", err);
    } finally {
      setLoading(false);
    }
  };

  const resetWizard = () => {
    setCurrentStep(0);
    setSelectedProvider(null);
    setServerName("");
    setConfig({});
    setAdvancedConfig(DEFAULT_ADVANCED_CONFIG);
    setValidationErrors({});
    setTestResult(null);
    setCompletedSteps(new Set());
    setVisitedSteps(new Set([0]));
    setError(null);
    setSaving(false);
  };

  const validateForm = (): boolean => {
    const errors: Record<string, string> = {};

    // Validate server name
    if (!serverName.trim()) {
      errors.serverName = "Server name is required";
    }

    // Validate provider selection
    if (!selectedProvider) {
      errors.provider = "Please select a provider";
      setValidationErrors(errors);
      return false;
    }

    // Validate required config fields
    selectedProvider.configFields.forEach(field => {
      if (field.required && !config[field.key]?.trim()) {
        errors[field.key] = `${field.label} is required`;
      }

      // Validate field-specific rules
      const value = config[field.key];
      if (value && field.validation) {
        if (field.validation.pattern) {
          const regex = new RegExp(field.validation.pattern);
          if (!regex.test(value)) {
            errors[field.key] = `Invalid format for ${field.label}`;
          }
        }
        if (field.validation.minLength && value.length < field.validation.minLength) {
          errors[field.key] = `${field.label} must be at least ${field.validation.minLength} characters`;
        }
        if (field.validation.maxLength && value.length > field.validation.maxLength) {
          errors[field.key] = `${field.label} must not exceed ${field.validation.maxLength} characters`;
        }
      }
    });

    setValidationErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const canProceedToStep = (stepId: number): boolean => {
    switch (stepId) {
      case 0: return true; // Provider selection is always accessible
      case 1: return selectedProvider !== null; // Credentials require provider
      case 2: return selectedProvider !== null; // Configuration requires provider
      case 3: return selectedProvider !== null; // Testing requires provider
      case 4: return selectedProvider !== null && serverName.trim() !== ""; // Summary requires basics
      default: return false;
    }
  };

  const canGoToNextStep = (): boolean => {
    switch (currentStep) {
      case 0: return selectedProvider !== null;
      case 1: {
        if (!selectedProvider) return false;
        const requiredFields = selectedProvider.configFields.filter(f => f.required);
        return requiredFields.every(field => config[field.key]?.trim());
      }
      case 2: return true; // Configuration step is optional
      case 3: return true; // Testing step is optional
      case 4: return serverName.trim() !== "" && selectedProvider !== null;
      default: return false;
    }
  };

  const goToStep = (stepId: number) => {
    if (stepId < 0 || stepId >= WIZARD_STEPS.length) return;
    if (stepId > currentStep && !canProceedToStep(stepId)) return;

    setCurrentStep(stepId);
    setVisitedSteps(prev => new Set([...prev, stepId]));

    // Mark previous steps as completed if moving forward
    if (stepId > currentStep) {
      const newCompleted = new Set(completedSteps);
      for (let i = currentStep; i < stepId; i++) {
        if (canGoToNextStep()) {
          newCompleted.add(i);
        }
      }
      setCompletedSteps(newCompleted);
    }
  };

  const nextStep = () => {
    if (canGoToNextStep() && currentStep < WIZARD_STEPS.length - 1) {
      const newCompleted = new Set(completedSteps);
      newCompleted.add(currentStep);
      setCompletedSteps(newCompleted);
      goToStep(currentStep + 1);
    }
  };

  const prevStep = () => {
    if (currentStep > 0) {
      goToStep(currentStep - 1);
    }
  };

  const handleProviderSelect = (providerId: string) => {
    const provider = providers.find(p => p.id === providerId);
    if (provider) {
      setSelectedProvider(provider);
      setConfig({}); // Reset config when changing provider
      setValidationErrors({});
      setTestResult(null);
      
      // Auto-generate server name if not set
      if (!serverName.trim()) {
        setServerName(`${provider.name} Server`);
      }
    }
  };

  const handleConfigChange = (key: string, value: string) => {
    setConfig(prev => ({
      ...prev,
      [key]: value,
    }));
    
    // Clear validation error for this field
    if (validationErrors[key]) {
      setValidationErrors(prev => {
        const { [key]: _, ...rest } = prev;
        return rest;
      });
    }
  };

  const handleAdvancedConfigChange = (newConfig: Partial<AdvancedConfig>) => {
    setAdvancedConfig(prev => ({ ...prev, ...newConfig }));
  };

  const handleTestComplete = (result: TestResult) => {
    setTestResult(result);
  };

  const createServer = async () => {
    if (!selectedProvider || !validateForm()) return;

    setSaving(true);
    try {
      const serverData = {
        name: serverName.trim(),
        providerId: selectedProvider.id,
        providerName: selectedProvider.name,
        config: {
          ...config,
          // Include advanced config as metadata
          _advanced: JSON.stringify(advancedConfig),
        },
      };

      const result = await createExternalServer(serverData);
      onSuccess(result);
      onClose();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setSaving(false);
    }
  };

  const getProgressPercentage = () => {
    const totalSteps = WIZARD_STEPS.length;
    const completedCount = completedSteps.size;
    const currentProgress = currentStep === totalSteps - 1 ? 1 : 0; // Add current step if it's the last one
    return ((completedCount + currentProgress) / totalSteps) * 100;
  };

  const renderStepComponent = () => {
    const step = WIZARD_STEPS[currentStep];
    const Component = step.component;

    const commonProps = {
      provider: selectedProvider,
      config,
      validationErrors,
      onConfigChange: handleConfigChange,
    };

    switch (currentStep) {
      case 0:
        return (
          <ProviderSelectionStep
            providers={providers}
            selectedProvider={selectedProvider}
            loading={loading}
            error={error}
            onProviderSelect={handleProviderSelect}
          />
        );
      case 1:
        return selectedProvider ? (
          <CredentialConfigStep
            {...commonProps}
            provider={selectedProvider}
          />
        ) : null;
      case 2:
        return selectedProvider ? (
          <ConfigurationStep
            {...commonProps}
            provider={selectedProvider}
            advancedConfig={advancedConfig}
            onAdvancedConfigChange={handleAdvancedConfigChange}
          />
        ) : null;
      case 3:
        return selectedProvider ? (
          <ConnectionTestStep
            provider={selectedProvider}
            config={config}
            advancedConfig={advancedConfig}
            onTestComplete={handleTestComplete}
          />
        ) : null;
      case 4:
        return selectedProvider ? (
          <SummaryStep
            provider={selectedProvider}
            serverName={serverName}
            config={config}
            advancedConfig={advancedConfig}
            testResult={testResult}
            onServerNameChange={setServerName}
            onEditStep={goToStep}
          />
        ) : null;
      default:
        return null;
    }
  };

  return (
    <Dialog open={open} onOpenChange={onClose}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-hidden flex flex-col">
        <DialogHeader className="flex-shrink-0">
          <div className="flex items-center justify-between">
            <DialogTitle>Setup Remote MCP Server</DialogTitle>
            <Button variant="ghost" size="sm" onClick={onClose}>
              <X className="h-4 w-4" />
            </Button>
          </div>
        </DialogHeader>

        {/* Progress Bar */}
        <div className="flex-shrink-0 space-y-2">
          <div className="flex items-center justify-between text-sm">
            <span>Step {currentStep + 1} of {WIZARD_STEPS.length}</span>
            <span>{Math.round(getProgressPercentage())}% Complete</span>
          </div>
          <Progress value={getProgressPercentage()} className="h-2" />
        </div>

        {/* Step Navigation */}
        <div className="flex-shrink-0 flex items-center justify-center gap-2 py-2">
          {WIZARD_STEPS.map((step, index) => (
            <React.Fragment key={step.id}>
              <button
                onClick={() => visitedSteps.has(index) && goToStep(index)}
                disabled={!visitedSteps.has(index)}
                className={`flex items-center gap-2 px-3 py-2 rounded-lg text-sm transition-colors ${
                  index === currentStep
                    ? "bg-blue-100 text-blue-800 border border-blue-300"
                    : completedSteps.has(index)
                    ? "bg-green-100 text-green-800 hover:bg-green-200"
                    : visitedSteps.has(index)
                    ? "bg-gray-100 text-gray-700 hover:bg-gray-200"
                    : "bg-gray-50 text-gray-400 cursor-not-allowed"
                }`}
              >
                {completedSteps.has(index) && <CheckCircle className="h-3 w-3" />}
                <span className="font-medium">{step.title}</span>
                {step.isOptional && <span className="text-xs opacity-75">(Optional)</span>}
              </button>
              {index < WIZARD_STEPS.length - 1 && (
                <ChevronRight className="h-3 w-3 text-gray-400" />
              )}
            </React.Fragment>
          ))}
        </div>

        {/* Step Content */}
        <Card className="flex-1 overflow-hidden">
          <CardContent className="h-full overflow-y-auto p-6">
            {error && (
              <Alert variant="destructive" className="mb-6">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
            
            {renderStepComponent()}
          </CardContent>
        </Card>

        {/* Navigation Buttons */}
        <div className="flex-shrink-0 flex items-center justify-between pt-4 border-t">
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              onClick={prevStep}
              disabled={currentStep === 0}
            >
              <ChevronLeft className="h-4 w-4 mr-2" />
              Back
            </Button>
          </div>

          <div className="flex items-center gap-2">
            {currentStep < WIZARD_STEPS.length - 1 ? (
              <Button
                onClick={nextStep}
                disabled={!canGoToNextStep()}
              >
                Next
                <ChevronRight className="h-4 w-4 ml-2" />
              </Button>
            ) : (
              <Button
                onClick={createServer}
                disabled={saving || !validateForm()}
              >
                {saving ? (
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                ) : (
                  <CheckCircle className="h-4 w-4 mr-2" />
                )}
                Create Server
              </Button>
            )}
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}