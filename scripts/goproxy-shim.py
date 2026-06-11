#!/usr/bin/env python3
"""Local GOPROXY shim: Go fetches modules over plain HTTP from localhost,
this forwards to https://proxy.golang.org through the sandbox egress proxy
using OpenSSL-based TLS (tolerates the proxy CA's DN-encoding quirk that
Go's strict raw-DN chain building rejects).

Dev-sandbox tooling only — never part of the Balaur product or its build.
Usage and the full diagnosis: docs/hyperagent-sandbox.md
    python3 scripts/goproxy-shim.py &                          # modules
    python3 scripts/goproxy-shim.py 8100 https://sum.golang.org &  # sumdb
    export GOPROXY=http://127.0.0.1:8099
    export GOSUMDB="sum.golang.org http://127.0.0.1:8100"

The second instance exists because proxy.golang.org stopped answering
/sumdb/ mirror paths (observed June 2026, 404s), so the Go tool falls
back to sum.golang.org directly and hits the same TLS quirk. Pointing
GOSUMDB's URL field at a shim keeps checksum verification fully ON —
only the transport hop changes, never the trust."""
import sys
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
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8099
    if len(sys.argv) > 2:
        UPSTREAM = sys.argv[2]
    ThreadingHTTPServer(("127.0.0.1", port), Handler).serve_forever()
