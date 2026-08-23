# Hyperscale starters

Three small apps that call a Hyperscale Product API over plain HTTP. One per
language, around a hundred lines each, and not one third-party dependency
between them.

They answer one question in under a minute: is my key real, and what did my
Product actually build? Every starter makes the same call and prints the
answer, so you can throw away the one you do not need and keep reading code
in the language you write.

These are examples to copy, not a library to depend on. They ship MIT so you
can paste them into your own project; keep the copyright line from `LICENSE`
with whatever you take. The three packages the starters talk about
(`@hyperscale0/udl`, `@hyperscale0/hsx`, `@hyperscale0/provider-adapter`) are
AGPL-3.0-only, not MIT, with a commercial license available from Hyperscale
LLC. These starters depend on none of them.

## The matrix

| Starter                              | Needs               | Run                | Test                         |
| ------------------------------------ | ------------------- | ------------------ | ---------------------------- |
| [typescript-node](./typescript-node) | Node 22.18 or newer | `node src/main.ts` | `node --test test/*.test.ts` |
| [python](./python)                   | Python 3.9 or newer | `python3 main.py`  | `python3 -m unittest`        |
| [go](./go)                           | Go 1.22 or newer    | `go run .`         | `go test ./...`              |

Every starter reads the same three environment variables:

| Variable                 | Default                  | Meaning               |
| ------------------------ | ------------------------ | --------------------- |
| `HYPERSCALE_API_KEY`     | none, required           | Your Product API key. |
| `HYPERSCALE_BASE_URL`    | `https://hyperscale0.ai` | The API origin.       |
| `HYPERSCALE_ENVIRONMENT` | `sandbox`                | `sandbox` or `live`.  |

Each starter carries a `.env.example` naming those three.

## The smoke call

Every starter makes one request:

```
GET {HYPERSCALE_BASE_URL}/v1/llms.txt
Authorization: Bearer {HYPERSCALE_API_KEY}
X-Hyperscale-Environment: {HYPERSCALE_ENVIRONMENT}
```

The response is your Product's own machine-readable descriptor: its title,
every operation it serves with method and path, and the golden paths through
them. The starters parse the title and the operation list and print them.

That one call proves four things at once. The key is real. The key belongs to
the environment you named, because a sandbox key and a live key are never
interchangeable. Your Product has a build behind it. And the operations you
see are the ones you can call, because the descriptor is cut from the same
build the API dispatches against.

It is also the only call worth starting with, because it is the only one every
Product serves. Everything else in your API surface exists because your Product
composed the capability behind it, so no other route is a fair smoke test.

## Get a key

Product API keys are minted on the Developers desk in the Hyperscale portal.
Open your Product there and mint a sandbox key. If you do not have portal
access, ask whoever operates your Hyperscale workspace for it.

Sandbox and live are separate planes with separate keys. Start in sandbox.

## Test without a key

Each starter's test suite runs the whole smoke path against a mock server it
starts on `127.0.0.1`, so `npm test`, `python3 -m unittest`, and `go test`
need neither a key nor a network. That is also the fastest way to read what a
starter does: the test shows the request going out and the parsed result
coming back.

## Add a starter

`CONTRIBUTING.md` has the shape a new language has to match. The short version:
same three environment variables, same one call, no dependencies, and a test
against a local mock server.

## Trademarks

"Hyperscale" is a trademark of Hyperscale LLC. The MIT license covers the code
in this repository and grants no permission to use the name or the marks.
