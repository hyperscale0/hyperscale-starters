from __future__ import annotations

import unittest

from client import (
    ProductApiError,
    fetch_product_descriptor,
    parse_product_descriptor,
    read_config,
)
from mock_server import SAMPLE_DESCRIPTOR, MockServer

API_KEY = "sk_sandbox_example"


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
