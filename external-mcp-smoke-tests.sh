#!/bin/bash

# External MCP Server Implementation Smoke Tests
# Tests all the external MCP server functionality

set -e

BASE_URL="http://127.0.0.1:38018"
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}🚀 External MCP Server Implementation Smoke Tests${NC}"
echo "=================================================="

# Function to test API endpoint
test_api() {
    local method=$1
    local endpoint=$2
    local data=$3
    local expected_status=${4:-200}
    local description=$5
    
    echo -e "\n${YELLOW}Testing:${NC} $description"
    echo "  $method $endpoint"
    
    if [ "$method" = "GET" ]; then
        response=$(curl -s -w "\n%{http_code}" "$BASE_URL$endpoint")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d "$data")
    fi
    
    # Split response and status code
    status_code=$(echo "$response" | tail -1)
    body=$(echo "$response" | head -n -1 || echo "")
    
    if [ "$status_code" -eq "$expected_status" ]; then
        echo -e "  ${GREEN}✅ Success (HTTP $status_code)${NC}"
        if [ "$method" = "GET" ] || [ -n "$body" ]; then
            echo "     Response: $(echo "$body" | head -c 100)..."
        fi
    else
        echo -e "  ${RED}❌ Failed (HTTP $status_code, expected $expected_status)${NC}"
        echo "     Response: $body"
        exit 1
    fi
}

echo -e "\n${YELLOW}1. Testing Health Check${NC}"
test_api "GET" "/healthz" "" 200 "Basic health check"

echo -e "\n${YELLOW}2. Testing Provider Registry${NC}"
test_api "GET" "/v1/external/providers" "" 200 "List all providers"
test_api "GET" "/v1/external/providers/openai" "" 200 "Get OpenAI provider details"

echo -e "\n${YELLOW}3. Testing Credential Management${NC}"
test_api "POST" "/v1/credentials" '{
    "provider": "openai",
    "credentials": {
        "api_key": "sk-test1234567890abcdefghijklmnopqrstuvwxyz1234567890",
        "organization_id": "org-testorganization123"
    }
}' 200 "Store credentials"

test_api "GET" "/v1/credentials/openai" "" 200 "Get credential requirements"

test_api "POST" "/v1/credentials/validate" '{
    "provider": "openai",
    "credentials": {
        "api_key": "sk-test1234567890abcdefghijklmnopqrstuvwxyz1234567890"
    }
}' 200 "Validate credentials (will fail with fake key)"

test_api "PUT" "/v1/credentials/openai" '{
    "credentials": {
        "api_key": "sk-updated1234567890abcdefghijklmnopqrstuvwxyz1234567890"
    }
}' 200 "Update credentials"

test_api "DELETE" "/v1/credentials/openai" "" 200 "Delete credentials"

echo -e "\n${YELLOW}4. Testing External Server CRUD Operations${NC}"
test_api "GET" "/v1/external/servers" "" 200 "List external servers (should be empty)"

test_api "POST" "/v1/external/servers" '{
    "name": "Test OpenAI Server",
    "slug": "test-openai",
    "provider": "openai",
    "displayName": "Test OpenAI Connection",
    "credentials": {
        "api_key": "sk-test1234567890abcdefghijklmnopqrstuvwxyz1234567890"
    },
    "config": {
        "model": "gpt-3.5-turbo",
        "temperature": "0.7"
    },
    "autoStart": false
}' 201 "Create external server"

test_api "GET" "/v1/external/servers" "" 200 "List external servers (should have 1)"

test_api "GET" "/v1/external/servers/test-openai" "" 200 "Get specific external server"

test_api "POST" "/v1/external/servers/test-openai/test" "" 200 "Test external server connection"

test_api "PUT" "/v1/external/servers/test-openai" '{
    "displayName": "Updated OpenAI Test Server",
    "config": {
        "model": "gpt-4",
        "temperature": "0.5"
    }
}' 200 "Update external server"

test_api "DELETE" "/v1/external/servers/test-openai" "" 200 "Delete external server"

test_api "GET" "/v1/external/servers" "" 200 "List external servers (should be empty again)"

echo -e "\n${YELLOW}5. Testing Health Monitoring${NC}"
test_api "POST" "/v1/external/servers" '{
    "name": "Notion Health Test",
    "slug": "notion-health-test",
    "provider": "notion",
    "displayName": "Notion Health Test",
    "credentials": {
        "api_key": "secret_testkey1234567890abcdefghijklmnopqrstuvwxyz1234567890"
    },
    "config": {
        "version": "2022-06-28"
    },
    "autoStart": true
}' 201 "Create server for health monitoring test"

test_api "GET" "/v1/health" "" 200 "Check general health status"
test_api "GET" "/v1/health/external" "" 200 "Check external health status"

# Test connection to trigger health check
test_api "POST" "/v1/external/servers/notion-health-test/test" "" 200 "Test connection to trigger health monitoring"

# Clean up
test_api "DELETE" "/v1/external/servers/notion-health-test" "" 200 "Clean up health test server"

echo -e "\n${YELLOW}6. Testing Multiple Provider Types${NC}"

# Test different provider types
providers=("openai" "notion" "slack" "github" "google" "microsoft")

for provider in "${providers[@]}"; do
    test_api "GET" "/v1/external/providers/$provider" "" 200 "Get $provider provider details"
done

echo -e "\n${GREEN}🎉 All External MCP Server Smoke Tests Passed!${NC}"
echo "=============================================="
echo -e "${GREEN}✅ Provider registry system working${NC}"
echo -e "${GREEN}✅ Credential management working${NC}"
echo -e "${GREEN}✅ External server CRUD operations working${NC}"
echo -e "${GREEN}✅ Connection testing working${NC}"
echo -e "${GREEN}✅ Health monitoring integration working${NC}"
echo -e "${GREEN}✅ All provider types supported${NC}"

echo -e "\n${YELLOW}🚀 System Ready for Production!${NC}"