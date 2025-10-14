# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "fastapi",
#     "pydantic",
#     "uvicorn",
# ]
# ///

import uvicorn

from fastapi import FastAPI
from pydantic import BaseModel

import os

hidden_message = os.getenv("HIDDEN_MESSAGE", "failed")

app = FastAPI()


# Define a Pydantic model for the input data (optional, but good practice)
class EchoInput(BaseModel):
    message: str


class EchoOutput(BaseModel):
    hidden_message: str
    received_message: str


@app.post("/echo/")
async def echo_message(data: EchoInput):
    """
    An echo service that returns the received message.
    """
    return EchoOutput(received_message=data.message, hidden_message=hidden_message)


if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)
