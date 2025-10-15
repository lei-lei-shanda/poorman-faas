# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "mcp>=1.12",
#     "pydantic",
#     "starlette",
#     "uvicorn",
# ]
# ///

"""
A simple MCP echo server that echoes back messages.
"""

from mcp.server.fastmcp import FastMCP
from pydantic import BaseModel
from starlette.applications import Starlette
from starlette.responses import JSONResponse
from starlette.routing import Route, Mount
import uvicorn
import os

hidden_message = os.getenv("HIDDEN_MESSAGE", "failed")


# Define a Pydantic model for the input data (optional, but good practice)
class EchoInput(BaseModel):
    message: str


class EchoOutput(BaseModel):
    hidden_message: str
    received_message: str


# Initialize the MCP server
mcp = FastMCP("EchoServer", stateless_http=True, json_response=True)


@mcp.tool()
def echo(message: EchoInput) -> EchoOutput:
    """
    Echoes back the received message.

    Args:
        message: The message to echo back

    Returns:
        The same message that was received
    """
    return EchoOutput(received_message=message.message, hidden_message=hidden_message)


# Health check endpoint
async def health_check(request):
    """
    Health check endpoint for readiness probe.
    """
    return JSONResponse({"status": "ok"})


# Create Starlette app with health endpoint and MCP server mounted
app = Starlette(
    routes=[
        Route("/health", health_check),
        Mount("/mcp", app=mcp.streamable_http_app()),
    ]
)


if __name__ == "__main__":
    # Start the Starlette server
    uvicorn.run(app, host="0.0.0.0", port=8000)
