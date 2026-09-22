# Python

A Python script that calls your Hyperscale Product API over plain HTTP. Two
files, standard library only.

## Install

There is nothing to install. No virtual environment, no `pip`, no
`pyproject.toml`: the starter imports `urllib`, `json`, and `re`, all of which
ship with Python.

```bash
python3 --version   # 3.9 or newer
```

## Run

```bash
cp .env.example .env
# put your key in .env

set -a && . ./.env && set +a
python3 main.py
```

Python does not read `.env` by itself, which is what `set -a` is doing there:
it exports every variable the file assigns. Or skip the file:

```bash
export HYPERSCALE_API_KEY=<your Product API key>
python3 main.py
```

Mock descriptor output looks like this. The current smoke route is retired; see
[the starters README](../README.md).

```
Example Product (sandbox)
2 operations
  GET    /v1/accounts  account_list
  POST   /v1/accounts  account_create
```

## Test

```bash
python3 -m unittest
```

The suite starts a mock API on `127.0.0.1` and runs the real client against
it, so it needs no key and no network. It asserts the two headers that go out,
the parsed descriptor that comes back, and the message you get when a key is
refused.

## The smoke call, and getting a key

The smoke route status and where a Product API key comes from are the same
for all three starters, so they live once in
[the starters README](../README.md).

## Continue with object discovery

Read [Product objects and actions](https://hyperscale0.ai/docs) for runtime
discovery and execution. The shared TypeScript client is `@hyperscale0/sdk`.
Python and Go clients call the same HTTP routes. Discover object kinds and
attached actions before submitting input; retain the returned Build identity,
digest, target and revision. Retry an uncertain mutation with its original
body and idempotency key.
