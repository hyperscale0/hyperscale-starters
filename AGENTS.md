# Lane charter: starters

**Role.** Three hand-written example apps, one per language, that call a
Hyperscale Product API over plain HTTP. They exist to be read and copied, not
imported. Nothing else depends on them.

**License.** MIT, `LICENSE` at the root of this directory, copyright Hyperscale
LLC. That is deliberate and it is narrower than it looks: MIT covers the
starters and the code copied into them, and nothing else in the platform tree
is MIT. The three units under `open/` ship AGPL-3.0-only with a commercial
license from Hyperscale LLC; everything else is proprietary.

**Lane rules.**

- Hand-written only. No generated SDK code is vendored here. The generated
  SDK is the documented upgrade path out of a starter, never a dependency of
  one.
- Zero third-party dependencies, in every language. The standard library and
  its HTTP client, or the change does not land.
- Every starter makes the same one call and reads the same three environment
  variables. `CONTRIBUTING.md` states the four rules a new language matches.
- Every starter's test runs the whole path against a mock HTTP server it
  starts itself. No test reaches the network and no test needs a key.
- The public origin and a placeholder registry are the only hosts that may
  appear in this directory. Internal hostnames, the private npm scope, and
  paths into the closed tree are refused by the export, which is a check and
  not a filter: a hit fails the export rather than editing the file.

**Export.** This directory is exported one-way to a public GitHub repository.
Nobody edits that repository; the export replaces its whole working tree, so a
deletion here becomes a deletion there. It publishes no npm package.
