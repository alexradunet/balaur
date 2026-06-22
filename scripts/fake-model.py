#!/usr/bin/env python3
"""A scriptable fake OpenAI-compatible model server for Balaur harness tests.

Balaur's CLI exists so external harnesses (including LLM agents) can drive
and verify a box. The deterministic commands need no model; `balaur chat`
does. This stub makes chat turns deterministic too: point Balaur at it and
script exactly what the "model" says and which tools it calls.

Usage:
    python3 scripts/fake-model.py script.json [port]    # default port 11435

    # Register as a cloud model (Models page → add an OpenAI-compatible model
    # with base URL http://127.0.0.1:11435/v1 and model ID "fake"), or seed
    # the llm_providers/llm_models/llm_settings rows as ci.yml's harness job does.
    balaur --dir /tmp/box chat "remind me to water the plants tomorrow"

script.json is a list of replies, consumed one per /chat/completions call
(turns make several calls: each tool round, plus the verify repair pass):

    [
      {"tool": "task_add", "args": {"title": "Water the plants", "due": "2027-03-01"}},
      {"text": "I've added watering the plants for March 1."}
    ]

When the script runs out, further calls get an empty reply. Stdlib only;
serves until killed. State is per-process: restart to replay the script.
"""

import json
import sys
import threading
from http.server import BaseHTTPRequestHandler, HTTPServer

REPLIES = []
LOCK = threading.Lock()


def sse(payload):
    return b"data: " + json.dumps(payload).encode() + b"\n\n"


class Handler(BaseHTTPRequestHandler):
    def log_message(self, *args):  # quiet: harness output stays clean
        pass

    def do_POST(self):
        self.rfile.read(int(self.headers.get("Content-Length", 0)))
        if self.path.rstrip("/").endswith("/embeddings"):
            body = json.dumps({"data": []}).encode()
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return

        with LOCK:
            reply = REPLIES.pop(0) if REPLIES else {"text": ""}

        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.end_headers()
        if reply.get("text"):
            self.wfile.write(sse({"choices": [{"delta": {"content": reply["text"]}}]}))
        if reply.get("tool"):
            call = {
                "index": 0,
                "id": "call-fake",
                "function": {
                    "name": reply["tool"],
                    "arguments": json.dumps(reply.get("args", {})),
                },
            }
            self.wfile.write(sse({"choices": [{"delta": {"tool_calls": [call]}}]}))
        self.wfile.write(b"data: [DONE]\n\n")


def main():
    if len(sys.argv) < 2:
        sys.exit(__doc__)
    with open(sys.argv[1]) as f:
        REPLIES.extend(json.load(f))
    port = int(sys.argv[2]) if len(sys.argv) > 2 else 11435
    print(f"fake-model: {len(REPLIES)} scripted replies on http://127.0.0.1:{port}/v1", flush=True)
    HTTPServer(("127.0.0.1", port), Handler).serve_forever()


if __name__ == "__main__":
    main()
