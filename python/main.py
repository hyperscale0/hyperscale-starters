"""The whole starter: read three environment variables, make one call, print
what came back. Run it with ``python3 main.py``. Standard library only.
"""

from __future__ import annotations

import os
import sys

from client import ProductApiError, fetch_product_descriptor, read_config

#: Enough to see the shape without scrolling; the rest is a count.
PREVIEW = 10


def main() -> int:
    try:
        config = read_config(os.environ)
        product = fetch_product_descriptor(config)
    except ProductApiError as error:
        print(error, file=sys.stderr)
        return 1

    print("{} ({})".format(product.title, config.environment))
    print("{} operations".format(len(product.operations)))
    for operation in product.operations[:PREVIEW]:
        print(
            "  {:<6} {}  {}".format(
                operation.method, operation.path, operation.operation_id
            )
        )
    rest = len(product.operations) - PREVIEW
    if rest > 0:
        print("  and {} more".format(rest))
    return 0


if __name__ == "__main__":
    sys.exit(main())
