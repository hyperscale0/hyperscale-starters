"""The whole starter: read three environment variables, make one call, print
what came back. Run it with ``python3 main.py``. Standard library only.
"""

from __future__ import annotations

import os
import sys

from client import ProductApiError, list_operations, read_config


def main() -> int:
    try:
        config = read_config(os.environ)
        page = list_operations(config)
    except ProductApiError as error:
        print(error, file=sys.stderr)
        return 1

    print("Operations ({})".format(config.environment))
    if not page.operations:
        print("  none yet; actions you run with this key show up here")
    for operation in page.operations:
        print(
            "  {:<9} {}  {}".format(
                operation.status, operation.name, operation.operation_id
            )
        )
    if page.has_more:
        print("  and more on the next page")
    return 0


if __name__ == "__main__":
    sys.exit(main())
