# MCP Manager Credential System

## Overview

This document describes the secure credential management system implemented for MCP Manager. The system provides secure storage, validation, and management of credentials for external MCP service providers through a robust set of API endpoints and OS-level keychain integration.

## Architecture

### Core Components

1. **Keychain Vault** (`internal/vault/keychain.go`)
   - Cross-platform OS keychain integration
   - AES-GCM encryption for additional security
   - Supports macOS Keychain, Windows Credential Manager, Linux Secret Service
   - Audit logging for credential operations

2. **HTTP API Endpoints** (`internal/httpapi/credentials.go`)
   - RESTful API for credential management
   - Rate limiting for validation attempts
   - Comprehensive error handling and security measures
   - Integration with existing server architecture

3. **Provider Registry Integration** (`internal/providers/registry.go`)
   - Pre-configured templates for popular services
   - Credential validation rules and requirements
   - Authentication type support (API key, OAuth2, Basic)

4. **Health Check Integration** (`internal/health/external.go`)
   - Real-time credential validation against provider APIs
   - Response time monitoring
   - Status tracking and error reporting

## API Endpoints

### POST /v1/credentials
Store credentials securely for a provider.

**Request:**
```json
{
  "provider": "notion",
  "credentials": {
    "api_key": "secret_1234567890123456789012345678901234567890"
  }
}
```

**Response:**
```json
{
  "success": true,
  "message": "Credentials stored successfully"
}
```

### GET /v1/credentials/{provider}
Get credential requirements for a specific provider.

**Response:**
```json
{
  "provider": "notion",
  "credentials": [
    {
      "key": "api_key",
      "displayName": "API Key",
      "required": true,
      "secret": true,
      "validation": "^secret_[a-zA-Z0-9]{40,}$",
      "description": "Notion integration API key from your workspace settings",
      "example": "secret_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
    }
  ],
  "authType": "api_key",
  "description": "Access and manage Notion databases, pages, and blocks through the Notion API"
}
```

### PUT /v1/credentials/{provider}
Update existing credentials for a provider.

**Request:**
```json
{
  "credentials": {
    "api_key": "secret_new1234567890123456789012345678901234567890"
  }
}
```

### DELETE /v1/credentials/{provider}
Remove stored credentials for a provider.

**Response:**
```json
{
  "success": true,
  "message": "Credentials deleted successfully"
}
```

### POST /v1/credentials/validate
Validate credentials against the provider's API.

**Request:**
```json
{
  "provider": "notion",
  "credentials": {
    "api_key": "secret_1234567890123456789012345678901234567890"
  }
}
```

**Response:**
```json
{
  "valid": false,
  "status": "invalid_credentials",
  "message": "Credentials are invalid or have insufficient permissions",
  "healthCheck": {
    "status": "error",
    "statusCode": 401,
    "responseTime": 150,
    "error": "client error: 401",
    "timestamp": "2025-09-09T12:00:00Z"
  }
}
```

## Supported Providers

The system comes with pre-configured templates for popular services:

- **Notion** - Database and page management
- **Slack** - Messaging and collaboration
- **GitHub** - Repository and issue management
- **Google Workspace** - Drive, Sheets, Calendar, Gmail
- **Microsoft 365** - Outlook, OneDrive, Teams, SharePoint
- **OpenAI** - GPT models and AI services

## Security Features

### Encryption
- **AES-GCM** encryption for credential data at rest
- **OS Keychain Integration** for secure key storage
- **Derived encryption keys** from service name and system information

### Access Control
- **Rate limiting** - Maximum 5 validation attempts per minute per provider
- **Audit logging** - All credential operations are logged (without values)
- **Secure error messages** - No credential information leaked in error responses
- **HTTP-only access** - Credentials never exposed in API responses

### Platform Support
- **macOS** - Uses Security.framework and `security` command
- **Windows** - Windows Credential Manager integration
- **Linux** - Secret Service API (partial implementation)

## Usage Examples

### Storing Notion Credentials

```bash
curl -X POST http://localhost:38018/v1/credentials \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "notion",
    "credentials": {
      "api_key": "secret_1234567890123456789012345678901234567890"
    }
  }'
```

### Validating GitHub Credentials

```bash
curl -X POST http://localhost:38018/v1/credentials/validate \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "github",
    "credentials": {
      "personal_access_token": "ghp_1234567890123456789012345678901234567890"
    }
  }'
```

### Getting Provider Requirements

```bash
curl -X GET http://localhost:38018/v1/credentials/openai
```

## Error Handling

### HTTP Status Codes
- **200 OK** - Successful operation
- **400 Bad Request** - Invalid request format or missing parameters
- **401 Unauthorized** - Invalid credentials during validation
- **403 Forbidden** - Missing permissions
- **404 Not Found** - Provider not found
- **429 Too Many Requests** - Rate limit exceeded
- **500 Internal Server Error** - Server error

### Error Response Format
```json
{
  "success": false,
  "message": "Human-readable error description"
}
```

## Testing

The system includes comprehensive tests:

- **Unit tests** for keychain vault operations
- **Integration tests** for HTTP API endpoints
- **Validation tests** for provider configurations
- **Rate limiting tests** for security features

Run tests with:
```bash
go test -v ./internal/vault/
go test -v ./internal/httpapi/ -run TestCredential
go test -v ./internal/providers/
```

## Configuration

### Environment Variables
- **MCP_KEYCHAIN_SERVICE** - Custom keychain service name (default: "mcp-manager")
- **MCP_CREDENTIAL_TIMEOUT** - Validation timeout in seconds (default: 15)

### Keychain Setup
The system automatically creates keychain entries under the service name "mcp-manager". On macOS, you can view stored credentials in Keychain Access.app.

## Best Practices

### For Developers
1. Always use the credential validation endpoint before storing
2. Handle rate limiting gracefully in client applications
3. Never log or expose credential values in application code
4. Use audit logs for security monitoring

### For Users
1. Use dedicated API keys with minimal required permissions
2. Regularly rotate credentials for security
3. Monitor audit logs for unauthorized access attempts
4. Keep provider API keys secure and never share them

## Future Enhancements

### Planned Features
- **Credential rotation** - Automatic refresh for OAuth2 tokens
- **Backup and restore** - Export/import encrypted credential backups
- **Multi-factor authentication** - Additional security layer for sensitive operations
- **Credential sharing** - Team-based credential management
- **Integration webhooks** - Notifications for credential events

### Platform Improvements
- **Complete Linux support** - Full Secret Service API integration
- **Hardware security modules** - HSM integration for enterprise deployments
- **Cloud keychain sync** - Cross-device credential synchronization

## Troubleshooting

### Common Issues

1. **Keychain Access Denied**
   - Solution: Grant permission in system keychain settings
   - macOS: System Preferences → Security & Privacy → Privacy → Keychain Access

2. **Validation Timeouts**
   - Check network connectivity to provider APIs
   - Verify firewall settings allow outbound HTTPS

3. **Rate Limiting**
   - Wait 60 seconds before retrying validation
   - Check for automated scripts causing excessive requests

4. **Credential Format Errors**
   - Use the GET endpoint to check required format
   - Verify credential format against provider documentation

### Debug Mode
Set environment variable `MCP_DEBUG=1` for detailed logging:
```bash
MCP_DEBUG=1 ./manager
```

## Conclusion

The MCP Manager credential system provides a secure, scalable solution for managing external service credentials. It follows security best practices while maintaining ease of use and comprehensive functionality for developers and end users.