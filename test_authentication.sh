#!/bin/bash

echo "Testing Authentication Service Endpoints"
echo "==============================="
echo ""

# Base URL
BASE_URL="http://localhost:8081"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to test endpoint
test_endpoint() {
    local method=$1
    local endpoint=$2
    local data=$3
    local description=$4

    echo -e "${YELLOW}Testing:${NC} $description"
    echo "Endpoint: $method $BASE_URL$endpoint"

    if [ -n "$data" ]; then
        echo "Request Body: $data"
        response=$(curl -s -X $method "$BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d "$data" | python3 -m json.tool 2>/dev/null || echo "Invalid JSON response")
    else
        response=$(curl -s -X $method "$BASE_URL$endpoint" | python3 -m json.tool 2>/dev/null || echo "Invalid JSON response")
    fi

    echo "Response:"
    echo "$response"
    echo "---"
    echo ""
}

# Test health endpoints
echo -e "${GREEN}=== Health Checks ===${NC}"
test_endpoint "GET" "/health" "" "Basic Health Check"
test_endpoint "GET" "/live" "" "Liveness Check"
test_endpoint "GET" "/ready" "" "Readiness Check"
test_endpoint "GET" "/api/v1/authentication/health" "" "Authentication Service Health"

# Test registration
echo -e "${GREEN}=== User Registration ===${NC}"
REGISTER_DATA='{
    "email": "test@example.com",
    "username": "testuser",
    "password": "TestPass123!",
    "first_name": "Test",
    "last_name": "User"
}'
test_endpoint "POST" "/api/v1/authentication/register" "$REGISTER_DATA" "Register New User"

# Test login
echo -e "${GREEN}=== User Login ===${NC}"
LOGIN_DATA='{
    "username": "testuser",
    "password": "TestPass123!"
}'
test_endpoint "POST" "/api/v1/authentication/login" "$LOGIN_DATA" "User Login"

# Test login with wrong password
echo -e "${GREEN}=== Failed Login Test ===${NC}"
WRONG_LOGIN_DATA='{
    "username": "testuser",
    "password": "WrongPassword"
}'
test_endpoint "POST" "/api/v1/authentication/login" "$WRONG_LOGIN_DATA" "Login with Wrong Password"

echo -e "${GREEN}Test completed!${NC}"