from __future__ import annotations

import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from client import (
    OPERATIONS_PATH,
    Operation,
    ProductApiError,
    list_operations,
    parse_operation_page,
    read_config,
)
from mock_server import MockServer

API_KEY = "sk_sandbox_example"


class RedirectingServer:
    """A loopback origin that answers every GET with a 302 somewhere else.

    urllib copies the request headers onto the redirected request, so this is
    all it takes to walk a Product API key to another origin.
    """

    def __init__(self, location: str) -> None:
        class Handler(BaseHTTPRequestHandler):
            def do_GET(self) -> None:  # noqa: N802 - the name http.server dispatches on
                self.send_response(302)
                self.send_header("Location", location)
                self.send_header("Content-Length", "0")
                self.end_headers()

            def log_message(self, format: str, *args: object) -> None:
                """Silence the per-request line http.server writes to stderr."""

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


class SmokeTest(unittest.TestCase):
    def setUp(self) -> None:
        self.server = MockServer(api_key=API_KEY)
        self.server.start()
        self.addCleanup(self.server.stop)

    def config(self, **overrides: str):
        environment_variables = {
            "HYPERSCALE_API_KEY": API_KEY,
            "HYPERSCALE_BASE_URL": self.server.base_url,
        }
        environment_variables.update(overrides)
        return read_config(environment_variables)

    def test_sends_the_key_and_the_environment_and_parses_the_operations(self) -> None:
        page = list_operations(self.config())

        self.assertEqual(
            list(page.operations),
            [
                Operation("ops_sandbox_example01", "customer.create", "succeeded"),
                Operation("ops_sandbox_example02", "account.create", "failed"),
            ],
        )
        self.assertTrue(page.has_more)

        request = self.server.received[-1]
        self.assertEqual(request.path, OPERATIONS_PATH)
        self.assertEqual(request.authorization, "Bearer {}".format(API_KEY))
        self.assertEqual(request.environment, "sandbox")
        self.assertEqual(request.accept, "application/json")

    def test_the_live_plane_is_addressed_by_the_header(self) -> None:
        list_operations(self.config(HYPERSCALE_ENVIRONMENT="live"))

        self.assertEqual(self.server.received[-1].environment, "live")

    def test_a_refused_key_surfaces_the_error_code(self) -> None:
        config = self.config(HYPERSCALE_API_KEY="sk_sandbox_wrong")

        with self.assertRaises(ProductApiError) as caught:
            list_operations(config)

        self.assertIn("HTTP 401", str(caught.exception))
        self.assertIn("invalid_credentials", str(caught.exception))

    def test_a_trailing_slash_does_not_double_up_the_path(self) -> None:
        config = self.config(
            HYPERSCALE_BASE_URL="{}///".format(self.server.base_url)
        )

        list_operations(config)

        self.assertEqual(self.server.received[-1].path, OPERATIONS_PATH)


class RedirectTest(unittest.TestCase):
    """The key is half the credential for one origin, so a 3xx never carries
    it to another. README.md and SECURITY.md both say the key goes to the
    configured base URL and nowhere else; this is that sentence, executed.
    """

    def test_a_redirect_does_not_carry_the_key_to_another_origin(self) -> None:
        elsewhere = MockServer(api_key=API_KEY)
        elsewhere.start()
        self.addCleanup(elsewhere.stop)
        redirector = RedirectingServer(elsewhere.base_url + OPERATIONS_PATH)
        redirector.start()
        self.addCleanup(redirector.stop)

        # The destination is live and does log what reaches it, so the empty
        # log below means the request was never made rather than missed.
        list_operations(
            read_config(
                {
                    "HYPERSCALE_API_KEY": API_KEY,
                    "HYPERSCALE_BASE_URL": elsewhere.base_url,
                }
            )
        )
        self.assertEqual(len(elsewhere.received), 1)
        elsewhere.received.clear()

        redirected = read_config(
            {
                "HYPERSCALE_API_KEY": API_KEY,
                "HYPERSCALE_BASE_URL": redirector.base_url,
            }
        )
        with self.assertRaises(ProductApiError) as caught:
            list_operations(redirected)

        self.assertIn("redirected", str(caught.exception))
        self.assertIn("HYPERSCALE_BASE_URL", str(caught.exception))
        self.assertEqual(elsewhere.received, [])


class ConfigTest(unittest.TestCase):
    def test_defaults_to_sandbox_on_the_public_origin(self) -> None:
        config = read_config({"HYPERSCALE_API_KEY": API_KEY})

        self.assertEqual(config.environment, "sandbox")
        self.assertEqual(config.base_url, "https://hyperscale0.ai")

    def test_refuses_a_missing_key_and_an_unknown_plane(self) -> None:
        with self.assertRaises(ProductApiError):
            read_config({})

        with self.assertRaises(ProductApiError):
            read_config(
                {"HYPERSCALE_API_KEY": API_KEY, "HYPERSCALE_ENVIRONMENT": "staging"}
            )


class ParseTest(unittest.TestCase):
    def test_a_response_that_is_not_an_operation_list_is_refused(self) -> None:
        for body in (
            "# Example Product",
            '{"error": "none"}',
            '{"items": [{"name": "customer.create"}]}',
        ):
            with self.assertRaises(ProductApiError):
                parse_operation_page(body)

    def test_an_empty_page_is_still_a_page(self) -> None:
        page = parse_operation_page('{"items": []}')

        self.assertEqual(list(page.operations), [])
        self.assertFalse(page.has_more)


if __name__ == "__main__":
    unittest.main()
