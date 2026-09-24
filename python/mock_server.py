"""A stand-in for the Product API, listening on 127.0.0.1 so the suite proves
the whole path -- headers out, JSON back, parsed result -- without a key
and without the network.
"""

from __future__ import annotations

import json
import threading
from dataclasses import dataclass
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from typing import List, Optional

#: A two-item page in the shape ``GET /v1/operations`` serves, with a cursor
#: that says another page follows.
SAMPLE_OPERATIONS = json.dumps(
    {
        "items": [
            {
                "operationId": "ops_sandbox_example01",
                "tenantId": "ten_sandbox_example01",
                "name": "customer.create",
                "status": "succeeded",
                "createdAt": "2026-09-25T09:00:00.000Z",
            },
            {
                "operationId": "ops_sandbox_example02",
                "tenantId": "ten_sandbox_example01",
                "name": "account.create",
                "status": "failed",
                "errorCode": "validation_failed",
                "errorMessage": "currency is required",
                "createdAt": "2026-09-25T09:01:00.000Z",
            },
        ],
        "nextCursor": "cursor_example",
        "statusCounts": {"failed": 1, "succeeded": 1},
    }
)


@dataclass(frozen=True)
class ReceivedRequest:
    """What one starter request looked like on the wire."""

    path: str
    authorization: Optional[str]
    environment: Optional[str]
    accept: Optional[str]


class MockServer:
    def __init__(self, api_key: str, document: str = SAMPLE_OPERATIONS) -> None:
        self.received: List[ReceivedRequest] = []
        received = self.received

        class Handler(BaseHTTPRequestHandler):
            def do_GET(self) -> None:  # noqa: N802 - the name http.server dispatches on
                received.append(
                    ReceivedRequest(
                        path=self.path,
                        authorization=self.headers.get("Authorization"),
                        environment=self.headers.get("X-Hyperscale-Environment"),
                        accept=self.headers.get("Accept"),
                    )
                )
                if self.headers.get("Authorization") != "Bearer {}".format(api_key):
                    self._respond(
                        401,
                        "application/json",
                        json.dumps(
                            {
                                "error": {
                                    "code": "invalid_credentials",
                                    "message": "Bearer token is not a valid API key in this environment.",
                                },
                                "requestId": "req_mock",
                            }
                        ),
                    )
                    return
                self._respond(200, "application/json", document)

            def _respond(self, status: int, content_type: str, body: str) -> None:
                encoded = body.encode("utf-8")
                self.send_response(status)
                self.send_header("Content-Type", content_type)
                self.send_header("Content-Length", str(len(encoded)))
                self.end_headers()
                self.wfile.write(encoded)

            def log_message(self, format: str, *args: object) -> None:
                """Silence the per-request line http.server writes to stderr."""

        # Port 0 lets the kernel pick a free port, so two suites never collide.
        self._server = ThreadingHTTPServer(("127.0.0.1", 0), Handler)
        self._thread = threading.Thread(
            target=self._server.serve_forever, daemon=True
        )

    def start(self) -> None:
        self._thread.start()

    def stop(self) -> None:
        self._server.shutdown()
        self._thread.join(timeout=5)
        self._server.server_close()

    @property
    def base_url(self) -> str:
        host, port = self._server.server_address[:2]
        return "http://{}:{}".format(host, port)
