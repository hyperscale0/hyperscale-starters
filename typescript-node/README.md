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

Output against the test mock looks like this:

```
Operations (sandbox)
  succeeded customer.create  ops_sandbox_example01
  failed    account.create  ops_sandbox_example02
  and more on the next page
```

A new Product prints `none yet` under the heading until you run an action
with the key.

## Test

```bash
npm test
```

The suite starts a mock API on `127.0.0.1` and runs the real client against
it, so it needs no key and no network. It asserts the two headers that go out,
the parsed operations that come back, and the message you get when a key is
refused.

## Getting a key

[The starters README](../README.md) says where to mint a Product API key and
where object discovery is documented.

## Typecheck, if you want to

The starter is written in erasable TypeScript, so `tsconfig.json` is here and
your editor already reads it. Nothing checks the types at runtime, because
stripping is not checking. To check them:

```bash
npm install --save-dev typescript @types/node
npx tsc --noEmit
```

That is the one place a dependency is worth it, and it stays a dev dependency.
