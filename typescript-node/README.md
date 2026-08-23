# TypeScript on Node

A Node app that calls your Hyperscale Product API over plain HTTP. Two files,
no dependencies, no build step.

## Install

There is nothing to install. Node 22.18 or newer runs `.ts` files directly by
stripping the types, which is why this starter has no compiler, no bundler,
and no `node_modules`.

```bash
node --version   # 22.18 or newer
```

## Run

```bash
cp .env.example .env
# put your key in .env
npm start
```

Or without the script:

```bash
export HYPERSCALE_API_KEY=<your Product API key>
node src/main.ts
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
npm test
```

The suite starts a mock API on `127.0.0.1` and runs the real client against
it, so it needs no key and no network. It asserts the two headers that go out,
the parsed descriptor that comes back, and the message you get when a key is
refused.

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

## Typecheck, if you want to

The starter is written in erasable TypeScript, so `tsconfig.json` is here and
your editor already reads it. Nothing checks the types at runtime, because
stripping is not checking. To check them:

```bash
npm install --save-dev typescript @types/node
npx tsc --noEmit
```

That is the one place a dependency is worth it, and it stays a dev dependency.

## Upgrade to the generated SDK

This starter builds requests by hand on purpose: it is short enough to read in
one sitting and it works before you have decided anything. Your Product also
has a generated TypeScript SDK, and once you are past the first call, that is
what you want. It carries typed inputs and outputs per operation, idempotency
keys on the mutations that require them, retries, pagination, and per-operation
examples derived from your own contract.

The SDK is served from a private npm registry, one per Product, authenticated
with the same key you already have. The Developers desk prints your registry
URL and package name; put them in an `.npmrc` next to your project:

```
<scope>:registry=<your registry URL>
//<registry host>/<registry path>/:_authToken=${HYPERSCALE_API_KEY}
```

Then install the package the desk names and keep the key in the environment,
exactly as it is here. Nothing about `.env` changes; only the code that makes
the call does.
