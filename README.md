<p align="left">
<picture>
  <source media="(prefers-color-scheme: dark)" srcset="docs/assets/brand/hyperscale-horizontal-white.svg">
  <source media="(prefers-color-scheme: light)" srcset="docs/assets/brand/hyperscale-horizontal.svg">
  <img src="docs/assets/brand/hyperscale-horizontal.svg" alt="Hyperscale™" width="394">
</picture>
</p>

# Hyperscale starters

Hyperscale™ starters: three small apps that call a Product API over plain HTTP.
One per language, around a hundred lines each, with no third-party dependency.

They answer one question in under a minute: is my key real, and what has my
Product run? Every starter makes the same call and prints the answer, so you
can throw away the one you do not need and keep reading code in the language
you write.

These are examples to copy, not a library to depend on. They ship MIT so you
can paste them into your own project; keep the copyright line from `LICENSE`
with whatever you take. The three packages the starters talk about
(`@hyperscale0/udl`, `@hyperscale0/hsx`, `@hyperscale0/adl`) are
Tier 2 source under the Hyperscale Intellectual Property and Copyright
License, not MIT. These starters depend on none of them.

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
GET {HYPERSCALE_BASE_URL}/v1/operations?limit=10
Authorization: Bearer {HYPERSCALE_API_KEY}
X-Hyperscale-Environment: {HYPERSCALE_ENVIRONMENT}
```

The answer is one page of Operations: each action that ran, its name and how
it ended. Every Product API key may read this list, whatever the Product was
composed from. A new Product has run nothing yet, so an empty list with
HTTP 200 still proves the key. A refused key prints the HTTP status and the
API's error code.

## Continue with object discovery

Read [Product objects and actions](https://hyperscale0.ai/docs) for runtime
discovery and execution. The shared TypeScript client is `@hyperscale0/sdk`;
Python and Go call the same HTTP routes. Discover object kinds and attached
actions before submitting input, and keep the returned Build identity, digest,
target and revision. Retry an uncertain mutation with its original body and
idempotency key.

## Get a key

Product API keys are minted on the Developers desk in Hyperscale Portal.
Open your Product there and mint a sandbox key. If you do not have portal
access, ask whoever operates your Hyperscale Product for it.

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

---

Hyperscale™ is a trademark of Hyperscale LLC. Code licenses do not grant rights to the name or marks.
