#!/bin/bash
#
# MCP Gateway Standalone Runner
# Run MCP Gateway without Docker Desktop dependency
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}MCP Gateway - Standalone Mode${NC}"
echo "================================"

# Check if Docker is installed and running
if ! command -v docker &> /dev/null; then
    echo -e "${RED}Error: Docker is not installed${NC}"
    echo "Please install Docker Engine (not Docker Desktop) first"
    exit 1
fi

# Check if Docker daemon is running
if ! docker info &> /dev/null; then
    echo -e "${RED}Error: Docker daemon is not running${NC}"
    echo "Please start Docker daemon first"
    exit 1
fi

# Check if docker-mcp plugin is installed
if ! docker mcp --help &> /dev/null; then
    echo -e "${YELLOW}Warning: docker-mcp plugin not found${NC}"
    echo "Building and installing the plugin..."

    # Build the plugin
    if [ -f "Makefile" ]; then
        make docker-mcp
    else
        echo -e "${RED}Error: Cannot find Makefile${NC}"
        echo "Please run this script from the project root directory"
        exit 1
    fi
fi

# Set environment variable to skip Docker Desktop check
export DOCKER_MCP_SKIP_DESKTOP_CHECK=1

# Check for command argument
if [ $# -eq 0 ]; then
    # Default to gateway run if no arguments
    echo -e "${GREEN}Starting MCP Gateway...${NC}"
    exec docker mcp gateway run
else
    # Pass through all arguments
    exec docker mcp "$@"
fi