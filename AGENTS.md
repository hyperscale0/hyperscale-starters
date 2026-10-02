# starters

Three hand-written example apps, one per language, that call a Hyperscale
Product API over plain HTTP. They exist to be read and copied; nothing depends
on them.

This directory is MIT ([`LICENSE`](LICENSE)). Nothing else in the platform tree
is; the units under [`open/`](../open/AGENTS.md) ship Tier 2 source.

- Hand-written only. No generated SDK code is vendored here; the SDK is the
  upgrade path out of a starter, never a dependency of one.
- Zero third-party dependencies in every language.
- Every starter makes the same call and reads the same three environment
  variables. [`CONTRIBUTING.md`](CONTRIBUTING.md) states the four rules a new
  language matches.
- Every test runs the whole path against a mock HTTP server it starts itself,
  with no network and no key.
- Only the public origin and a placeholder registry may appear as hosts. The
  export fails on internal hostnames, the private npm scope and closed-tree
  paths; it never edits the file.

The export replaces the whole working tree of the public GitHub repository, so
a deletion here is a deletion there. Nobody edits that repository, and it
publishes no npm package.
