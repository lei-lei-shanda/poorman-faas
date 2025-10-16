#!/usr/bin/env bash

MCP_SERVER_PORT=8000
MCP_SERVER_HOST=0.0.0.0
# do health check
curl -X GET "http://${MCP_SERVER_HOST}:${MCP_SERVER_PORT}/health"
echo -e "\nHealth checked"

# for mcp service
# first initialize it
curl -X POST "http://${MCP_SERVER_HOST}:${MCP_SERVER_PORT}/mcp" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json,text/event-stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {
      "protocolVersion": "2025-03-26",
      "capabilities": {
        "tools": {},
        "resources": {},
        "prompts": {}
      },
      "clientInfo": {
        "name": "test-client",
        "version": "1.0.0"
      }
    }
  }'

echo -e "\nInitialized"

curl -X POST "http://${MCP_SERVER_HOST}:${MCP_SERVER_PORT}/mcp" \
    -H "Content-Type: application/json" \
    -H "Accept: application/json,text/event-stream" \
    -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/list",
    "params": {}
  }'

echo -e "\nTools listed"

curl -X POST http://${MCP_SERVER_HOST}:${MCP_SERVER_PORT}/mcp \
  -H "Content-Type: application/json" \
  -H "Accept: application/json,text/event-stream" \
  -d '{
    "jsonrpc": "2.0",
    "id": 3,
    "method": "tools/call",
    "params": {
      "name": "echo",
      "arguments": {
        "message": {"message": "Hello, World!"}
      }
    }
  }'

echo -e "\nTools called"