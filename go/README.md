# Go

A Go program that calls your Hyperscale Product API over plain HTTP. Two
files, standard library only, no `go get`.

## Install

There is nothing to install. `go.mod` names no requirements, so the module
graph is empty and `go build` reaches the network never.

```bash
go version   # 1.22 or newer
```

## Run

```bash
cp .env.example .env
# put your key in .env

set -a && . ./.env && set +a
go run .
```

Go does not read `.env` by itself, which is what `set -a` is doing there: it
exports every variable the file assigns. Or skip the file:

```bash
export HYPERSCALE_API_KEY=<your Product API key>
go run .
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
go test ./...
```

The suite starts a mock API on `127.0.0.1` with `httptest` and runs the real
client against it, so it needs no key and no network. It asserts the two
headers that go out, the parsed descriptor that comes back, and the message
you get when a key is refused.

`ReadConfig` takes a lookup function rather than calling `os.Getenv` itself,
which is why no test has to mutate the process environment.

## Build

```bash
go vet ./...
go build -o /dev/null ./...
```

The `-o /dev/null` is deliberate: a bare `go build` in a `main` package writes
an executable named after the directory into your checkout.

## What the smoke call proves

One request, `GET /v1/llms.txt`, carrying `Authorization: Bearer <key>` and
`X-Hyperscale-Environment: sandbox|live`. The response is your Product's own
machine-readable descriptor, and getting it back proves four things:

- The key is real.
- The key belongs to the plane you named. Sandbox keys and live keys are never
  interchangeable, so the environment header is half the credential, not a
  hint.
- Your Product has a build behind it.
- The operations printed are the ones you can call, because the descriptor is
  cut from the same build the API dispatches against.

It is the right first call because it is the only one every Product serves.
Everything else in your surface exists because your Product composed the
capability behind it.

## Get a key

Product API keys are minted on the Developers desk in the Hyperscale portal.
Open your Product there and mint a sandbox key. If you do not have portal
access, ask whoever operates your Hyperscale workspace for it.

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

If you are staying in Go, keep this client and grow it. The moving parts that
matter are the ones the SDK would give you: send an idempotency key on every
mutation, retry only what is safe to retry, and follow the cursor on every
list. Your Product's descriptor names which operations need which, which is
one more reason to make this call first.
