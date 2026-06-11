#!/usr/bin/env python3
"""Local GOPROXY shim: Go fetches modules over plain HTTP from localhost,
this forwards to https://proxy.golang.org through the sandbox egress proxy
using OpenSSL-based TLS (tolerates the proxy CA's DN-encoding quirk that
Go's strict raw-DN chain building rejects).

Dev-sandbox tooling only — never part of the Balaur product or its build.
Usage and the full diagnosis: docs/hyperagent-sandbox.md
    python3 scripts/goproxy-shim.py &
    export GOPROXY=http://127.0.0.1:8099"""
import urllib.request
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

UPSTREAM = "https://proxy.golang.org"


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def do_GET(self):
        url = UPSTREAM + self.path
        try:
            req = urllib.request.Request(url, headers={"User-Agent": "goproxy-shim"})
            with urllib.request.urlopen(req, timeout=120) as resp:
                body = resp.read()
                self.send_response(resp.status)
                self.send_header("Content-Length", str(len(body)))
                ct = resp.headers.get("Content-Type")
                if ct:
                    self.send_header("Content-Type", ct)
                self.end_headers()
                self.wfile.write(body)
        except urllib.error.HTTPError as e:
            body = e.read() or b""
            self.send_response(e.code)
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
        except Exception as e:  # network failure: tell Go to try elsewhere
            msg = str(e).encode()
            self.send_response(502)
            self.send_header("Content-Length", str(len(msg)))
            self.end_headers()
            self.wfile.write(msg)

    def log_message(self, *args):
        pass


if __name__ == "__main__":
    ThreadingHTTPServer(("127.0.0.1", 8099), Handler).serve_forever()
