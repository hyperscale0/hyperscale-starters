# Security

## Reporting a vulnerability

Report privately through GitHub, using this repository's
[private vulnerability reporting form](https://github.com/hyperscale0/hyperscale-starters/security/advisories/new).
That is the only intake. There is no security email address, and nothing
security-sensitive belongs in an issue, a pull request, a discussion, or a
commit message.

Never put a real API key in a report. If you think a key has leaked, rotate it
on the Developers desk first, then tell us.

## What counts

This repository is example code with no dependencies and one read-only call,
so the surface is small and specific:

- A starter that leaks the API key: printing it, writing it somewhere the
  repository tracks, putting it in a URL or a query string, sending it to any
  origin other than the configured base URL.
- A starter that would send a sandbox key to the live plane, or the reverse,
  through the environment header it builds.
- A starter that trusts the response document into something dangerous. The
  descriptor is text from a server the operator chose with
  `HYPERSCALE_BASE_URL`, so a capture that reaches a shell, a file path, or a
  generated file is a bug even though the server is normally ours.
- Anything in the copy-paste path that puts a person at risk: an example
  `.env` that is not ignored, a README step that suggests committing a secret.

Out of scope: the base URL pointing where you told it to point, and anything
that requires already controlling the machine running the starter.

If you have found a vulnerability in the Hyperscale API itself rather than in
this sample code, report it through the form above anyway. We will route it and
tell you where it went.

## Supported versions

Only the current `main` is supported. These are examples you copy, so a fix
lands on `main` and there is nothing to backport into a release you already
pasted.

## Disclosure

We will confirm receipt, tell you what we found, and agree a disclosure date
with you before publishing an advisory. If a fix is not straightforward we will
say so rather than go quiet.
