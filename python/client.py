"""The Product API, called with urllib and nothing else.

Two headers carry everything. ``Authorization: Bearer <key>`` presents the
Product API key, and ``X-Hyperscale-Environment`` picks which plane of that key
is being addressed. Sandbox keys and live keys are never interchangeable, so
the header is not a hint: it is half the credential. HTTP header names are
case-insensitive; these are the canonical spellings.

The key goes to HYPERSCALE_BASE_URL and nowhere else: this starter refuses
redirects rather than following them, the same as the Go and TypeScript
starters. The endpoint is fixed and read-only, so there is no hop worth taking.
"""

from __future__ import annotations

import json
from dataclasses import dataclass
from typing import Any, List, Mapping, Optional, Sequence
from urllib.error import HTTPError, URLError
from urllib.request import HTTPRedirectHandler, Request, build_opener

#: The public origin. Override it with HYPERSCALE_BASE_URL.
DEFAULT_BASE_URL = "https://hyperscale0.ai"

ENVIRONMENTS = ("sandbox", "live")

TIMEOUT_SECONDS = 30

#: One page of ten, enough to see the shape.
OPERATIONS_PATH = "/v1/operations?limit=10"


class _RefuseRedirects(HTTPRedirectHandler):
    """Stops urllib from carrying the API key to whatever host a 3xx names.

    ``HTTPRedirectHandler.redirect_request`` copies every request header onto
    the new request except content-length and content-type, so the default
    behavior hands ``Authorization: Bearer <key>`` to any origin the response
    points at. Returning ``None`` declines the redirect, and the 3xx comes back
    as an ``HTTPError`` the caller turns into a message. This endpoint is fixed
    and read-only, so there is no move worth following.
    """

    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None


#: One opener, built once, so every call in this module refuses redirects.
_OPENER = build_opener(_RefuseRedirects())


class ProductApiError(Exception):
    """A failure the person running this can fix, printed without a traceback."""


@dataclass(frozen=True)
class Config:
    api_key: str
    base_url: str
    environment: str


@dataclass(frozen=True)
class Operation:
    """One recorded execution: what ran and how it ended."""

    operation_id: str
    name: str
    status: str


@dataclass(frozen=True)
class OperationPage:
    operations: Sequence[Operation]
    has_more: bool


def read_config(environment_variables: Mapping[str, str]) -> Config:
    api_key = environment_variables.get("HYPERSCALE_API_KEY")
    if not api_key:
        raise ProductApiError(
            "No API key. Set HYPERSCALE_API_KEY to a Product API key; "
            "README.md says where to mint one."
        )

    environment = environment_variables.get("HYPERSCALE_ENVIRONMENT") or "sandbox"
    if environment not in ENVIRONMENTS:
        raise ProductApiError(
            "HYPERSCALE_ENVIRONMENT must be sandbox or live, "
            "not {}.".format(environment)
        )

    # A trailing slash would make every path double up on one.
    base_url = (
        environment_variables.get("HYPERSCALE_BASE_URL") or DEFAULT_BASE_URL
    ).rstrip("/")

    return Config(api_key=api_key, base_url=base_url, environment=environment)


def list_operations(config: Config) -> OperationPage:
    """The one read every Product key is allowed, whatever it was composed from.

    The capability that serves it is part of every Product. A fresh Product has
    run nothing yet, so an empty list with HTTP 200 still proves the key.
    """
    url = "{}{}".format(config.base_url, OPERATIONS_PATH)
    request = Request(
        url,
        headers={
            "Accept": "application/json",
            "Authorization": "Bearer {}".format(config.api_key),
            "X-Hyperscale-Environment": config.environment,
        },
    )

    try:
        with _OPENER.open(request, timeout=TIMEOUT_SECONDS) as response:
            body = response.read().decode("utf-8")
    except HTTPError as error:
        if 300 <= error.code < 400:
            # Nothing reads this body, so the socket has to be closed by hand.
            error.close()
            raise ProductApiError(
                "GET /v1/operations was redirected, so the key was not sent on. "
                "Set HYPERSCALE_BASE_URL to the origin the API answers on; "
                "the usual cause is http:// where it serves https://."
            ) from None
        # HTTPError is a response, so the body carries the API's error envelope.
        failed = error.read().decode("utf-8", errors="replace")
        detail = _error_envelope(failed) or failed.strip()
        raise ProductApiError(
            "GET /v1/operations failed: HTTP {}{}".format(
                error.code, "" if not detail else " - {}".format(detail)
            )
        ) from None
    except URLError as error:
        raise ProductApiError(
            "Could not reach {}: {}".format(url, error.reason)
        ) from None

    return parse_operation_page(body)


def parse_operation_page(body: str) -> OperationPage:
    """Refuse anything that is not an operation list, and any item without an
    id or a name. A starter that printed blank rows would hide the format
    moving under it.
    """
    try:
        parsed: Any = json.loads(body)
    except ValueError:
        parsed = None
    items = parsed.get("items") if isinstance(parsed, dict) else None
    if not isinstance(items, list):
        raise ProductApiError(
            "The response is not an operation list. Check "
            "HYPERSCALE_BASE_URL points at the API origin."
        )

    operations: List[Operation] = []
    for item in items:
        if not isinstance(item, dict):
            item = {}
        operation_id = item.get("operationId")
        name = item.get("name")
        status = item.get("status")
        if not (isinstance(operation_id, str) and operation_id) or not (
            isinstance(name, str) and name
        ):
            raise ProductApiError(
                "An operation arrived without an id or a name; "
                "this starter is out of date."
            )
        operations.append(
            Operation(
                operation_id=operation_id,
                name=name,
                status=status if isinstance(status, str) else "",
            )
        )

    return OperationPage(
        operations=operations, has_more=bool(parsed.get("nextCursor"))
    )


def _error_envelope(body: str) -> Optional[str]:
    """The API's error shape: ``{"error":{"code","message"},"requestId"}``."""
    try:
        parsed = json.loads(body)
    except ValueError:
        # A non-JSON body is already the best message available.
        return None
    if not isinstance(parsed, dict):
        return None
    error = parsed.get("error")
    if not isinstance(error, dict):
        return None
    code = error.get("code")
    message = error.get("message")
    if not isinstance(code, str) or not isinstance(message, str):
        return None
    return "{}: {}".format(code, message)
