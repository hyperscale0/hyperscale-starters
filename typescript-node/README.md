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
npm test
```

The suite starts a mock API on `127.0.0.1` and runs the real client against
it, so it needs no key and no network. It asserts the two headers that go out,
the parsed descriptor that comes back, and the message you get when a key is
refused.

## The smoke call, and getting a key

The smoke route status and where a Product API key comes from are the same
for all three starters, so they live once in
[the starters README](../README.md).

## Typecheck, if you want to

The starter is written in erasable TypeScript, so `tsconfig.json` is here and
your editor already reads it. Nothing checks the types at runtime, because
stripping is not checking. To check them:

```bash
npm install --save-dev typescript @types/node
npx tsc --noEmit
```

That is the one place a dependency is worth it, and it stays a dev dependency.

## Continue with object discovery

Read [Product objects and actions](https://hyperscale0.ai/docs) for runtime
discovery and execution. The shared TypeScript client is `@hyperscale0/sdk`.
Python and Go clients call the same HTTP routes. Discover object kinds and
attached actions before submitting input; retain the returned Build identity,
digest, target and revision. Retry an uncertain mutation with its original
body and idempotency key.
