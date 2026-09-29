#!/usr/bin/env python3
"""Tiny Gemini API reverse proxy for regions blocked by Google.

Forwards /v1beta/* to generativelanguage.googleapis.com from an allowed egress IP.
"""

from __future__ import annotations

import http.client
import http.server
import ssl
import sys
from urllib.parse import urlsplit

UPSTREAM_HOST = "generativelanguage.googleapis.com"
LISTEN_HOST = "127.0.0.1"
LISTEN_PORT = int(sys.argv[1]) if len(sys.argv) > 1 else 8787


class Handler(http.server.BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def do_GET(self):  # noqa: N802
        self._proxy()

    def do_POST(self):  # noqa: N802
        self._proxy()

    def do_OPTIONS(self):  # noqa: N802
        self.send_response(204)
        self.send_header("Access-Control-Allow-Origin", "*")
        self.send_header("Access-Control-Allow-Headers", "*")
        self.send_header("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
        self.end_headers()

    def log_message(self, fmt: str, *args) -> None:
        sys.stderr.write("%s - %s\n" % (self.address_string(), fmt % args))

    def _proxy(self) -> None:
        length = int(self.headers.get("Content-Length", "0") or "0")
        body = self.rfile.read(length) if length > 0 else None

        path = self.path
        if not path.startswith("/"):
            path = "/" + path

        conn = http.client.HTTPSConnection(UPSTREAM_HOST, timeout=90, context=ssl.create_default_context())
        headers = {}
        for key, value in self.headers.items():
            lk = key.lower()
            if lk in {"host", "content-length", "connection", "transfer-encoding"}:
                continue
            headers[key] = value
        headers["Host"] = UPSTREAM_HOST

        try:
            conn.request(self.command, path, body=body, headers=headers)
            upstream = conn.getresponse()
            payload = upstream.read()
        except Exception as exc:  # noqa: BLE001
            msg = ("upstream error: %s" % exc).encode()
            self.send_response(502)
            self.send_header("Content-Type", "text/plain")
            self.send_header("Content-Length", str(len(msg)))
            self.end_headers()
            self.wfile.write(msg)
            return
        finally:
            conn.close()

        self.send_response(upstream.status)
        for key, value in upstream.getheaders():
            lk = key.lower()
            if lk in {"transfer-encoding", "connection", "content-length"}:
                continue
            self.send_header(key, value)
        self.send_header("Content-Length", str(len(payload)))
        self.send_header("Access-Control-Allow-Origin", "*")
        self.end_headers()
        if self.command != "HEAD":
            self.wfile.write(payload)


def main() -> None:
    server = http.server.ThreadingHTTPServer((LISTEN_HOST, LISTEN_PORT), Handler)
    print("gemini-relay listening on http://%s:%d -> https://%s" % (LISTEN_HOST, LISTEN_PORT, UPSTREAM_HOST), flush=True)
    server.serve_forever()


if __name__ == "__main__":
    main()
