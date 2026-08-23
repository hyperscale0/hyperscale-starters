"""The Product API, called with urllib and nothing else.

Two headers carry everything. ``Authorization: Bearer <key>`` presents the
Product API key, and ``X-Hyperscale-Environment`` picks which plane of that key
is being addressed. Sandbox keys and live keys are never interchangeable, so
the header is not a hint: it is half the credential. HTTP header names are
case-insensitive; these are the canonical spellings.
"""

from __future__ import annotations

import json
import re
from dataclasses import dataclass
from typing import List, Mapping, Optional, Sequence
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

#: The public origin. Override it with HYPERSCALE_BASE_URL.
DEFAULT_BASE_URL = "https://hyperscale0.ai"

ENVIRONMENTS = ("sandbox", "live")

TIMEOUT_SECONDS = 30


class ProductApiError(Exception):
    """A failure the person running this can fix, printed without a traceback."""


@dataclass(frozen=True)
class Config:
    api_key: str
    base_url: str
    environment: str


@dataclass(frozen=True)
class Operation:
    method: str
    path: str
    operation_id: str


@dataclass(frozen=True)
class ProductDescriptor:
    title: str
    operations: Sequence[Operation]


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


def fetch_product_descriptor(config: Config) -> ProductDescriptor:
    """The one call every Product serves, whatever it was composed from.

    Everything else in the API surface exists because the Product composed the
    capability behind it, so this is the only fair smoke test.
    """
    url = "{}/v1/llms.txt".format(config.base_url)
    request = Request(
        url,
        headers={
            "Accept": "text/plain",
            "Authorization": "Bearer {}".format(config.api_key),
            "X-Hyperscale-Environment": config.environment,
        },
    )

    try:
        with urlopen(request, timeout=TIMEOUT_SECONDS) as response:
            body = response.read().decode("utf-8")
    except HTTPError as error:
        # HTTPError is a response, so the body carries the API's error envelope.
        failed = error.read().decode("utf-8", errors="replace")
        detail = _error_envelope(failed) or failed.strip()
        raise ProductApiError(
            "GET /v1/llms.txt failed: HTTP {}{}".format(
                error.code, "" if not detail else " - {}".format(detail)
            )
        ) from None
    except URLError as error:
        raise ProductApiError(
            "Could not reach {}: {}".format(url, error.reason)
        ) from None

    return parse_product_descriptor(body)


_TITLE = re.compile(r"^# (.+)$", re.MULTILINE)
_DECLARED_COUNT = re.compile(r"^## Operations \((\d+)\)$", re.MULTILINE)
_OPERATION = re.compile(
    r"^- ([A-Z]+) (\S+) · .+ \((\S+); idempotency \S+\)$", re.MULTILINE
)


def parse_product_descriptor(document: str) -> ProductDescriptor:
    """Read the descriptor with three anchored patterns rather than guessing.

    Parsing fewer operations than the document declares means the format moved
    under us, and that has to fail loudly: a starter that silently printed a
    short list would look like a Product missing half its surface.
    """
    title = _TITLE.search(document)
    declared = _DECLARED_COUNT.search(document)
    if title is None or declared is None:
        raise ProductApiError(
            "The response is not a product descriptor. Check "
            "HYPERSCALE_BASE_URL points at the API origin."
        )

    operations: List[Operation] = [
        Operation(method=match.group(1), path=match.group(2), operation_id=match.group(3))
        for match in _OPERATION.finditer(document)
    ]

    expected = int(declared.group(1))
    if len(operations) != expected:
        raise ProductApiError(
            "The descriptor declares {} operations but {} parsed; "
            "this starter is out of date.".format(expected, len(operations))
        )

    return ProductDescriptor(title=title.group(1).strip(), operations=operations)


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
