# /// script
# requires-python = ">=3.10"
# dependencies = [
#     "mcp",
#     "pydantic",
# ]
# ///

"""
A simple MCP echo server that echoes back messages.
"""

from mcp.server.fastmcp import FastMCP
from pydantic import BaseModel
import os

# Initialize the MCP server
mcp = FastMCP("EchoServer", stateless_http=True, json_response=True)
hidden_message = os.getenv("HIDDEN_MESSAGE", "failed")


# Define a Pydantic model for the input data (optional, but good practice)
class EchoInput(BaseModel):
    message: str


class EchoOutput(BaseModel):
    hidden_message: str
    received_message: str


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


if __name__ == "__main__":
    # Start the MCP server
    mcp.run(transport="streamable-http")
