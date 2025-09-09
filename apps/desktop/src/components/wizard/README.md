# Provider Setup Wizard

A comprehensive multi-step wizard component for setting up external MCP servers with advanced configuration, testing, and validation capabilities.

## Features

- **Step 1: Provider Selection** - Browse and search through available MCP server providers with category filtering
- **Step 2: Credential Configuration** - Dynamic form generation based on provider requirements with validation
- **Step 3: Advanced Configuration** - Optional performance tuning, retry settings, caching, and custom headers
- **Step 4: Connection Testing** - Real-time connection testing with detailed performance metrics and logs
- **Step 5: Review & Summary** - Final review with configuration scoring and pre-creation checks

## Components

### ProviderSetupWizard (Main Component)

```typescript
import { ProviderSetupWizard } from '@/components/ProviderSetupWizard';

function ExampleUsage() {
  const [wizardOpen, setWizardOpen] = useState(false);

  const handleSuccess = (server: ExternalServerConfig) => {
    console.log('Server created:', server);
    // Refresh server list, show success message, etc.
  };

  return (
    <div>
      <Button onClick={() => setWizardOpen(true)}>
        Add External Server
      </Button>
      
      <ProviderSetupWizard
        open={wizardOpen}
        onClose={() => setWizardOpen(false)}
        onSuccess={handleSuccess}
      />
    </div>
  );
}
```

### Individual Step Components

Each step can also be used individually if needed:

```typescript
import { 
  ProviderSelectionStep,
  CredentialConfigStep,
  ConfigurationStep,
  ConnectionTestStep,
  SummaryStep 
} from '@/components/wizard';
```

## Wizard Flow

1. **Provider Selection**: Users can search and filter through available providers by category (database, API, storage, AI, etc.)
2. **Credential Configuration**: Dynamic form fields based on the selected provider's requirements with field validation
3. **Advanced Configuration**: Optional settings including:
   - Connection timeout and retry behavior
   - Performance presets (High Performance, Reliable, Conservative)
   - Logging and caching settings
   - Custom HTTP headers
4. **Connection Testing**: Real-time testing with:
   - Multi-phase connection validation
   - Performance metrics (latency, throughput, availability)
   - Detailed logs and test history
   - Auto-retry functionality
5. **Review & Summary**: Final review with:
   - Configuration scoring system
   - Editable server name
   - Sensitive data masking/unmasking
   - Pre-creation validation checks

## Key Features

### Form Validation
- Real-time field validation
- Pattern matching for specific field types
- Required field enforcement
- Error recovery and guidance

### User Experience
- Progress tracking and step navigation
- Ability to go back and edit previous steps
- Loading states and error handling
- Responsive design for different screen sizes

### Security
- Secure credential storage
- Sensitive data masking in review
- Input sanitization and validation

### Testing & Monitoring
- Connection health checks
- Performance benchmarking
- Retry mechanisms with exponential backoff
- Detailed error reporting

## Integration with Existing Systems

The wizard integrates seamlessly with the existing MCP Manager API:
- Uses `getProviderTemplates()` to fetch available providers
- Uses `testExternalConnection()` for connection testing  
- Uses `createExternalServer()` to create the final configuration
- Follows existing validation patterns from `ExternalServerForm`

## Customization

The wizard can be customized by:
- Modifying the `WIZARD_STEPS` array to add/remove/reorder steps
- Extending the `AdvancedConfig` type for additional settings
- Adding custom validation rules per provider
- Customizing the UI theme and styling

## Error Handling

The wizard includes comprehensive error handling:
- Network connectivity issues
- Invalid credentials
- Provider-specific errors
- Validation failures with clear user guidance