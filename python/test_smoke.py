from __future__ import annotations

import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from client import (
    ProductApiError,
    fetch_product_descriptor,
    parse_product_descriptor,
    read_config,
)
from mock_server import SAMPLE_DESCRIPTOR, MockServer

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

    def test_sends_the_key_and_the_environment_and_parses_the_descriptor(self) -> None:
        product = fetch_product_descriptor(self.config())

        self.assertEqual(product.title, "Example Product")
        self.assertEqual(
            [(op.method, op.path, op.operation_id) for op in product.operations],
            [
                ("GET", "/v1/accounts", "account_list"),
                ("POST", "/v1/accounts", "account_create"),
            ],
        )

        request = self.server.received[-1]
        self.assertEqual(request.path, "/v1/llms.txt")
        self.assertEqual(request.authorization, "Bearer {}".format(API_KEY))
        self.assertEqual(request.environment, "sandbox")
        self.assertEqual(request.accept, "text/plain")

    def test_the_live_plane_is_addressed_by_the_header(self) -> None:
        fetch_product_descriptor(self.config(HYPERSCALE_ENVIRONMENT="live"))

        self.assertEqual(self.server.received[-1].environment, "live")

    def test_a_refused_key_surfaces_the_error_code(self) -> None:
        config = self.config(HYPERSCALE_API_KEY="sk_sandbox_wrong")

        with self.assertRaises(ProductApiError) as caught:
            fetch_product_descriptor(config)

        self.assertIn("HTTP 401", str(caught.exception))
        self.assertIn("invalid_credentials", str(caught.exception))

    def test_a_trailing_slash_does_not_double_up_the_path(self) -> None:
        config = self.config(
            HYPERSCALE_BASE_URL="{}///".format(self.server.base_url)
        )

        fetch_product_descriptor(config)

        self.assertEqual(self.server.received[-1].path, "/v1/llms.txt")


class RedirectTest(unittest.TestCase):
    """The key is half the credential for one origin, so a 3xx never carries
    it to another. README.md and SECURITY.md both say the key goes to the
    configured base URL and nowhere else; this is that sentence, executed.
    """

    def test_a_redirect_does_not_carry_the_key_to_another_origin(self) -> None:
        elsewhere = MockServer(api_key=API_KEY)
        elsewhere.start()
        self.addCleanup(elsewhere.stop)
        redirector = RedirectingServer("{}/v1/llms.txt".format(elsewhere.base_url))
        redirector.start()
        self.addCleanup(redirector.stop)

        # The destination is live and does log what reaches it, so the empty
        # log below means the request was never made rather than missed.
        fetch_product_descriptor(
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
            fetch_product_descriptor(redirected)

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
    def test_a_count_that_disagrees_with_the_lines_is_refused(self) -> None:
        short = SAMPLE_DESCRIPTOR.replace("## Operations (2)", "## Operations (3)")

        with self.assertRaises(ProductApiError) as caught:
            parse_product_descriptor(short)

        self.assertIn("declares 3 operations but 2 parsed", str(caught.exception))


if __name__ == "__main__":
    unittest.main()
