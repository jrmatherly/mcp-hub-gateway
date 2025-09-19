#!/bin/bash

# Test script to verify dynamic MCP tools are available

echo "Starting MCP Gateway and testing for dynamic tools..."

# Start the gateway in background
export DOCKER_MCP_SKIP_DESKTOP_CHECK=1
docker mcp gateway run &> gateway.log &
GATEWAY_PID=$!

# Wait for gateway to start
sleep 5

# Look for our dynamic tools in the log
echo "Checking for dynamic MCP management tools:"
echo "========================================="

# Check each tool
tools=("mcp-find" "mcp-add" "mcp-remove" "mcp-official-registry-import" "mcp-config-set")

for tool in "${tools[@]}"; do
    if grep -q "$tool" gateway.log; then
        echo "✅ $tool - FOUND"
    else
        echo "❌ $tool - NOT FOUND"
    fi
done

# Kill the gateway
kill $GATEWAY_PID 2>/dev/null

# Show tool count
echo ""
echo "Total tools listed:"
grep "tools listed in" gateway.log || echo "Could not determine tool count"

# Cleanup
rm -f gateway.log