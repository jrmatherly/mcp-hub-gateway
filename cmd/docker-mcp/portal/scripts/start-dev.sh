#!/bin/bash
# =================================================================
# MCP Portal Development Startup Script
# =================================================================
# Starts both frontend and backend services for development
# Uses the unified .env.local configuration
# =================================================================

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PORTAL_DIR="$( cd "$SCRIPT_DIR/.." && pwd )"
FRONTEND_DIR="$PORTAL_DIR/frontend"

# Check for tmux or screen (for running multiple processes)
if command -v tmux &> /dev/null; then
    MULTIPLEXER="tmux"
elif command -v screen &> /dev/null; then
    MULTIPLEXER="screen"
else
    MULTIPLEXER="none"
fi

# Function to cleanup on exit
cleanup() {
    echo -e "\n${YELLOW}Shutting down services...${NC}"
    if [ -n "$BACKEND_PID" ]; then
        kill $BACKEND_PID 2>/dev/null || true
    fi
    if [ -n "$FRONTEND_PID" ]; then
        kill $FRONTEND_PID 2>/dev/null || true
    fi
    exit 0
}

trap cleanup INT TERM

# Check if .env.local exists
if [ ! -f "$FRONTEND_DIR/.env.local" ]; then
    echo -e "${RED}Error: .env.local not found${NC}"
    echo -e "${YELLOW}Creating from unified example...${NC}"

    if [ -f "$FRONTEND_DIR/.env.local.unified.example" ]; then
        cp "$FRONTEND_DIR/.env.local.unified.example" "$FRONTEND_DIR/.env.local"
        echo -e "${GREEN}Created .env.local from unified example${NC}"
        echo -e "${YELLOW}Please edit $FRONTEND_DIR/.env.local with your configuration${NC}"
        exit 1
    else
        echo -e "${RED}Error: .env.local.unified.example not found${NC}"
        exit 1
    fi
fi

echo -e "${BLUE}=== MCP Portal Development Mode ===${NC}"
echo -e "${GREEN}Using unified configuration from .env.local${NC}"

# Option 1: Use multiplexer if available
if [ "$MULTIPLEXER" = "tmux" ]; then
    echo -e "${GREEN}Starting services in tmux session...${NC}"

    # Kill existing session if it exists
    tmux kill-session -t mcp-portal 2>/dev/null || true

    # Create new session with backend
    tmux new-session -d -s mcp-portal -n backend \
        "cd $PORTAL_DIR && ./scripts/start-with-env.sh; read -p 'Press enter to exit'"

    # Create frontend window
    tmux new-window -t mcp-portal:1 -n frontend \
        "cd $FRONTEND_DIR && npm run dev; read -p 'Press enter to exit'"

    # Create logs window
    tmux new-window -t mcp-portal:2 -n logs \
        "tail -f /tmp/mcp-portal/*.log 2>/dev/null || echo 'Waiting for logs...'; bash"

    echo -e "${GREEN}Services started in tmux session 'mcp-portal'${NC}"
    echo -e "${YELLOW}Commands:${NC}"
    echo "  Attach to session:  tmux attach -t mcp-portal"
    echo "  Switch windows:     Ctrl-B + window number (0-2)"
    echo "  Detach:            Ctrl-B + D"
    echo "  Kill session:      tmux kill-session -t mcp-portal"

    # Attach to session
    tmux attach -t mcp-portal

elif [ "$MULTIPLEXER" = "screen" ]; then
    echo -e "${GREEN}Starting services in screen session...${NC}"

    # Create screen session
    screen -dmS mcp-portal

    # Start backend
    screen -S mcp-portal -X screen -t backend bash -c \
        "cd $PORTAL_DIR && ./scripts/start-with-env.sh; read -p 'Press enter to exit'"

    # Start frontend
    screen -S mcp-portal -X screen -t frontend bash -c \
        "cd $FRONTEND_DIR && npm run dev; read -p 'Press enter to exit'"

    echo -e "${GREEN}Services started in screen session 'mcp-portal'${NC}"
    echo -e "${YELLOW}Commands:${NC}"
    echo "  Attach to session:  screen -r mcp-portal"
    echo "  Switch windows:     Ctrl-A + window number"
    echo "  Detach:            Ctrl-A + D"
    echo "  Kill session:      screen -X -S mcp-portal quit"

    # Attach to session
    screen -r mcp-portal

else
    # Option 2: Run in foreground without multiplexer
    echo -e "${YELLOW}No tmux/screen found. Running services in foreground...${NC}"
    echo -e "${YELLOW}Press Ctrl-C to stop all services${NC}"
    echo ""

    # Start backend in background
    echo -e "${GREEN}Starting backend service...${NC}"
    cd "$PORTAL_DIR"
    ./scripts/start-with-env.sh &
    BACKEND_PID=$!

    # Wait for backend to be ready
    echo -e "${YELLOW}Waiting for backend to be ready...${NC}"
    sleep 3

    # Check if backend is running
    if ! kill -0 $BACKEND_PID 2>/dev/null; then
        echo -e "${RED}Backend failed to start${NC}"
        exit 1
    fi

    # Start frontend in background
    echo -e "${GREEN}Starting frontend service...${NC}"
    cd "$FRONTEND_DIR"
    npm run dev &
    FRONTEND_PID=$!

    # Show status
    echo ""
    echo -e "${GREEN}=== Services Running ===${NC}"
    echo -e "${BLUE}Backend:${NC}  http://localhost:${API_PORT:-8080}"
    echo -e "${BLUE}Frontend:${NC} http://localhost:3000"
    echo -e "${YELLOW}Press Ctrl-C to stop all services${NC}"
    echo ""

    # Wait for processes
    wait $BACKEND_PID $FRONTEND_PID
fi