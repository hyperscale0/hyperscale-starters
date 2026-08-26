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

Output looks like this:

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

What the one request proves and where a Product API key comes from are the same
for all three starters, so they live once in
[the starters README](../README.md).

## Upgrade to the generated SDK

This starter builds the request by hand on purpose: it is short enough to read
in one sitting and it works before you have decided anything. Your Product also
has a generated TypeScript SDK, and once you are past the first call it carries
typed inputs and outputs per operation, idempotency keys on the mutations that
require them, retries, pagination, and per-operation examples derived from your
own contract.

The SDK is served from a private npm registry, one per Product, authenticated
with the same key you already have. The Developers desk prints your registry
URL and package name; put them in an `.npmrc` next to a JavaScript project:

```
<scope>:registry=<your registry URL>
//<registry host>/<registry path>/:_authToken=${HYPERSCALE_API_KEY}
```

If you are staying in Python, keep this client and grow it. The moving parts
that matter are the ones the SDK would give you: send an idempotency key on
every mutation, retry only what is safe to retry, and follow the cursor on
every list. Your Product's descriptor names which operations need which, which
is one more reason to make this call first.
