import React from "react";
import { Search, Zap } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { type ExternalServerProvider } from "../../api";

export type ProviderSelectionStepProps = {
  providers: ExternalServerProvider[];
  selectedProvider: ExternalServerProvider | null;
  loading: boolean;
  error: string | null;
  onProviderSelect: (providerId: string) => void;
};

export function ProviderSelectionStep({
  providers,
  selectedProvider,
  loading,
  error,
  onProviderSelect,
}: ProviderSelectionStepProps) {
  const [searchQuery, setSearchQuery] = React.useState("");
  const [selectedCategory, setSelectedCategory] = React.useState("all");

  const categories = React.useMemo(() => {
    const cats = new Set<string>();
    providers.forEach((provider) => {
      // Extract category from provider description or use a default categorization
      if (provider.description.toLowerCase().includes("database")) {
        cats.add("database");
      } else if (provider.description.toLowerCase().includes("api") || provider.description.toLowerCase().includes("service")) {
        cats.add("api");
      } else if (provider.description.toLowerCase().includes("file") || provider.description.toLowerCase().includes("storage")) {
        cats.add("storage");
      } else if (provider.description.toLowerCase().includes("ai") || provider.description.toLowerCase().includes("language")) {
        cats.add("ai");
      } else {
        cats.add("other");
      }
    });
    return Array.from(cats).sort();
  }, [providers]);

  const filteredProviders = React.useMemo(() => {
    let filtered = providers;

    // Filter by search query
    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      filtered = filtered.filter(
        (provider) =>
          provider.name.toLowerCase().includes(query) ||
          provider.description.toLowerCase().includes(query)
      );
    }

    // Filter by category
    if (selectedCategory !== "all") {
      filtered = filtered.filter((provider) => {
        const desc = provider.description.toLowerCase();
        switch (selectedCategory) {
          case "database":
            return desc.includes("database") || desc.includes("sql") || desc.includes("postgres") || desc.includes("mysql");
          case "api":
            return desc.includes("api") || desc.includes("service") || desc.includes("http") || desc.includes("rest");
          case "storage":
            return desc.includes("file") || desc.includes("storage") || desc.includes("s3") || desc.includes("blob");
          case "ai":
            return desc.includes("ai") || desc.includes("language") || desc.includes("gpt") || desc.includes("claude");
          default:
            return !desc.includes("database") && !desc.includes("api") && !desc.includes("file") && !desc.includes("ai");
        }
      });
    }

    return filtered;
  }, [providers, searchQuery, selectedCategory]);

  const getProviderCategory = (provider: ExternalServerProvider): string => {
    const desc = provider.description.toLowerCase();
    if (desc.includes("database") || desc.includes("sql") || desc.includes("postgres") || desc.includes("mysql")) return "database";
    if (desc.includes("api") || desc.includes("service") || desc.includes("http") || desc.includes("rest")) return "api";
    if (desc.includes("file") || desc.includes("storage") || desc.includes("s3") || desc.includes("blob")) return "storage";
    if (desc.includes("ai") || desc.includes("language") || desc.includes("gpt") || desc.includes("claude")) return "ai";
    return "other";
  };

  const getCategoryColor = (category: string): string => {
    switch (category) {
      case "database": return "bg-blue-100 text-blue-800";
      case "api": return "bg-green-100 text-green-800";
      case "storage": return "bg-orange-100 text-orange-800";
      case "ai": return "bg-purple-100 text-purple-800";
      default: return "bg-gray-100 text-gray-800";
    }
  };

  if (loading) {
    return (
      <div className="space-y-6">
        <div className="text-center py-12">
          <div className="animate-pulse">
            <div className="h-8 w-48 bg-gray-200 rounded mx-auto mb-4"></div>
            <div className="h-4 w-64 bg-gray-200 rounded mx-auto"></div>
          </div>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <Alert variant="destructive">
        <AlertDescription>{error}</AlertDescription>
      </Alert>
    );
  }

  return (
    <div className="space-y-6">
      <div className="text-center">
        <h2 className="text-2xl font-semibold mb-2">Choose a Provider</h2>
        <p className="text-muted-foreground">
          Select the type of remote MCP server you want to connect to
        </p>
      </div>

      {/* Search and Filter Controls */}
      <div className="space-y-4">
        <div className="space-y-2">
          <Label>Search Providers</Label>
          <div className="relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="Search by name or description..."
              className="pl-9"
            />
          </div>
        </div>

        <div className="space-y-2">
          <Label>Category</Label>
          <div className="flex flex-wrap gap-2">
            <button
              onClick={() => setSelectedCategory("all")}
              className={`px-3 py-1 text-sm rounded-md transition-colors ${
                selectedCategory === "all"
                  ? "bg-blue-100 text-blue-800 border border-blue-300"
                  : "bg-gray-100 text-gray-700 hover:bg-gray-200"
              }`}
            >
              All ({providers.length})
            </button>
            {categories.map((category) => {
              const count = providers.filter((p) => getProviderCategory(p) === category).length;
              return (
                <button
                  key={category}
                  onClick={() => setSelectedCategory(category)}
                  className={`px-3 py-1 text-sm rounded-md transition-colors capitalize ${
                    selectedCategory === category
                      ? "bg-blue-100 text-blue-800 border border-blue-300"
                      : "bg-gray-100 text-gray-700 hover:bg-gray-200"
                  }`}
                >
                  {category} ({count})
                </button>
              );
            })}
          </div>
        </div>
      </div>

      {/* Provider Grid */}
      <ScrollArea className="h-96">
        {filteredProviders.length > 0 ? (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 pr-4">
            {filteredProviders.map((provider) => {
              const isSelected = selectedProvider?.id === provider.id;
              const category = getProviderCategory(provider);
              
              return (
                <Card
                  key={provider.id}
                  className={`cursor-pointer transition-all duration-200 ${
                    isSelected
                      ? "border-blue-500 bg-blue-50 shadow-md"
                      : "hover:border-gray-300 hover:shadow-sm"
                  }`}
                  onClick={() => onProviderSelect(provider.id)}
                >
                  <CardHeader className="pb-3">
                    <div className="flex items-start justify-between">
                      <div className="flex items-center gap-3">
                        {provider.icon ? (
                          <span className="text-2xl">{provider.icon}</span>
                        ) : (
                          <div className="w-8 h-8 bg-gradient-to-br from-blue-500 to-purple-600 rounded-lg flex items-center justify-center">
                            <Zap className="h-4 w-4 text-white" />
                          </div>
                        )}
                        <div>
                          <CardTitle className="text-lg">{provider.name}</CardTitle>
                          <Badge className={`${getCategoryColor(category)} text-xs mt-1`}>
                            {category}
                          </Badge>
                        </div>
                      </div>
                      {isSelected && (
                        <Badge variant="secondary">Selected</Badge>
                      )}
                    </div>
                  </CardHeader>
                  <CardContent>
                    <CardDescription className="text-sm leading-relaxed">
                      {provider.description}
                    </CardDescription>
                    
                    {/* Show required fields preview */}
                    <div className="mt-3 pt-3 border-t border-gray-100">
                      <div className="text-xs text-muted-foreground">
                        <span className="font-medium">Required fields:</span> {
                          provider.configFields
                            .filter(field => field.required)
                            .map(field => field.label)
                            .join(", ") || "None"
                        }
                      </div>
                    </div>
                  </CardContent>
                </Card>
              );
            })}
          </div>
        ) : (
          <div className="text-center py-12">
            <div className="text-muted-foreground">
              <Search className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p className="text-lg font-medium mb-2">No providers found</p>
              <p className="text-sm">
                {searchQuery ? "Try adjusting your search terms" : "Try selecting a different category"}
              </p>
            </div>
          </div>
        )}
      </ScrollArea>

      {/* Selection Summary */}
      {selectedProvider && (
        <Card className="bg-blue-50 border-blue-200">
          <CardContent className="pt-4">
            <div className="flex items-center gap-3">
              {selectedProvider.icon && <span className="text-lg">{selectedProvider.icon}</span>}
              <div>
                <p className="font-medium text-blue-900">
                  {selectedProvider.name} selected
                </p>
                <p className="text-sm text-blue-700">
                  Ready to configure {selectedProvider.configFields.filter(f => f.required).length} required field{selectedProvider.configFields.filter(f => f.required).length !== 1 ? 's' : ''}
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  );
}