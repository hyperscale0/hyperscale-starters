# Python

A Python script that calls your Hyperscale Product API over plain HTTP. Two
files, standard library only.

## Install

There is nothing to install. No virtual environment, no `pip`, no
`pyproject.toml`: the starter imports `urllib` and `json`, both of which ship
with Python.

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
python3 -m unittest
```

The suite starts a mock API on `127.0.0.1` and runs the real client against
it, so it needs no key and no network. It asserts the two headers that go out,
the parsed operations that come back, and the message you get when a key is
refused.

## Getting a key

[The starters README](../README.md) says where to mint a Product API key and
where object discovery is documented.
