import React from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import "./index.css";
import { ThemeProvider } from "@/components/theme-provider";
import { Toaster } from "@/components/ui/toaster";
import { ErrorBoundary } from "@/components/ErrorBoundary";

const container = document.getElementById("root")!;
createRoot(container).render(
  <React.StrictMode>
    <ErrorBoundary>
      <ThemeProvider defaultTheme="system" storageKey="mcp-manager-ui-theme">
        <App />
        <Toaster />
      </ThemeProvider>
    </ErrorBoundary>
  </React.StrictMode>,
);
