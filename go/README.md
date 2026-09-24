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
go test ./...
```

The suite starts a mock API on `127.0.0.1` with `httptest` and runs the real
client against it, so it needs no key and no network. It asserts the two
headers that go out, the parsed operations that come back, and the message
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

## Getting a key

[The starters README](../README.md) says where to mint a Product API key and
where object discovery is documented.
