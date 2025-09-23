import React, { useState, useEffect, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { credsValidate, credsGetRequirements, credsStore } from "../api";

const Credentials = () => {
  const [provider, setProvider] = useState("notion");
  const [credentials, setCredentials] = useState<Record<string, string>>({});
  const [requirements, setRequirements] = useState<any[]>([]);
  const [validationStatus, setValidationStatus] = useState("");
  const [isSaving, setIsSaving] = useState(false);
  const [saveStatus, setSaveStatus] = useState("");

  const fetchRequirements = useCallback(async () => {
    try {
      const reqs = await credsGetRequirements(provider);
      setRequirements(reqs.credentials);
    } catch (error: any) {
      console.error("Failed to fetch credential requirements:", error);
      setValidationStatus(`Failed to fetch requirements: ${error.message}`);
    }
  }, [provider]);

  useEffect(() => {
    fetchRequirements();
  }, [fetchRequirements]);

  const handleCredentialChange = (key: string, value: string) => {
    setCredentials((prev) => ({ ...prev, [key]: value }));
  };

  const handleValidate = async () => {
    setValidationStatus("Validating...");
    try {
      const result = await credsValidate(provider, credentials);
      if (result.valid) {
        setValidationStatus("Credentials are valid!");
      } else {
        setValidationStatus(result.message || "Invalid credentials");
      }
    } catch (error: any) {
      setValidationStatus(`Validation failed: ${error.message}`);
    }
  };

  const handleSave = async () => {
    setIsSaving(true);
    setSaveStatus("Saving...");
    try {
      await credsStore(provider, credentials);
      setSaveStatus("Credentials saved successfully!");
    } catch (error: any) {
      setSaveStatus(`Failed to save credentials: ${error.message}`);
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <div className="p-6">
      <Card>
        <CardHeader>
          <CardTitle>Configure Credentials</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <Label>Provider</Label>
            <Input value={provider} onChange={(e) => setProvider(e.target.value)} />
          </div>
          {requirements.map((req) => (
            <div key={req.key}>
              <Label htmlFor={req.key}>{req.displayName}</Label>
              <Input
                id={req.key}
                type={req.secret ? "password" : "text"}
                value={credentials[req.key] || ""}
                onChange={(e) => handleCredentialChange(req.key, e.target.value)}
              />
              <p className="text-sm text-muted-foreground mt-1">{req.description}</p>
            </div>
          ))}
          {validationStatus && (
            <Alert variant={validationStatus.includes("valid") ? "default" : "destructive"}>
              <AlertDescription>{validationStatus}</AlertDescription>
            </Alert>
          )}
          <div className="flex space-x-2">
            <Button onClick={handleValidate}>Test Connection</Button>
            <Button onClick={handleSave} disabled={isSaving}>
              {isSaving ? "Saving..." : "Save"}
            </Button>
          </div>
          {saveStatus && <p>{saveStatus}</p>}
        </CardContent>
      </Card>
    </div>
  );
};

export default Credentials;
