# Contributing to the Hyperscale starters

## How changes get made

This repository is issues-only. Hyperscale writes the starters; the public
proposes changes in an issue. That is the whole model, and it is stated up
front so nobody spends a weekend on a branch that was never going to be merged.

A proposal is an issue carrying the use case, in the words of whoever hit it,
and the test the change would add. Hyperscale accepts a pull request rarely,
and only after asking for one. When that happens the maintainer who asked sends the CLA
and the author signs it before the merge: MIT alone does not let Hyperscale LLC relicense the contribution
alongside the rest of the tree, and it says nothing about patents.

## Setup

There is nothing to install for the repository itself. Each starter needs its
own language toolchain and nothing else:

```bash
node --version     # 22.18 or newer
python3 --version  # 3.9 or newer
go version         # 1.22 or newer
```

Run everything the way CI does:

```bash
bun run check      # every starter's build and smoke test
```

Or run one starter on its own:

```bash
node --test typescript-node/test/*.test.ts
python3 -m unittest discover -s python -t python
cd go && go build ./... && go test ./...
```

`bun run check` needs all three toolchains present. If you only have one, run
that starter directly; CI runs each of the three in its own job and none of
them depends on the others.

## The shape a starter has to match

A new language is welcome as a proposal. It is worth writing up when it matches
the four rules the existing three follow, because the point of this repository
is that the same program reads the same way in every language:

1. **The same three environment variables.** `HYPERSCALE_API_KEY`,
   `HYPERSCALE_BASE_URL` defaulting to the public origin, and
   `HYPERSCALE_ENVIRONMENT` defaulting to `sandbox`. A missing key or a
   nonsense environment fails with a sentence a person can act on, not a
   stack trace.
2. **The same one call.** `GET /v1/llms.txt` with the bearer key and the
   environment header, parsed into a title and a list of operations. Assert
   the parsed count against the count the document declares, so a format
   change fails loudly instead of printing a shorter list.
3. **No dependencies.** The standard library, its HTTP client, and nothing
   else. A starter that installs a package stops being readable in one sitting
   and starts being a supply chain. This also means no generated Hyperscale
   SDK code is ever vendored here; the SDK is the upgrade path, documented in
   each README, never a dependency of the example.
4. **A test against a local mock server.** The suite starts an HTTP server on
   `127.0.0.1`, asserts the request headers that went out, and asserts the
   parsed result that came back. No network, no key, no fixtures pulled from
   anywhere.

Add the new directory to the table in `README.md`, to the `check` script in
`package.json`, and to the CI workflow as its own job.

## Style

Match the language, not the other starters. Idiomatic Go looks nothing like
idiomatic Python and it should not. What stays constant is the program: the
same names for the same things, the same order of operations, the same output.

Comments explain constraints the code cannot show. They never narrate the next
line.

## Contributor licence agreement

In the rare case Hyperscale accepts a pull request, the author signs a CLA
first, and it only has to happen once. MIT alone does not let Hyperscale LLC
relicense the contribution alongside the rest of the tree, and it says nothing
about patents, so the patent terms live in the CLA too.

## Reporting

Bugs and proposals go through issues. Security vulnerabilities do not: see
`SECURITY.md`.
