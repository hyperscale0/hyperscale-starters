// A stand-in for the Product API, listening on 127.0.0.1 so the suite proves
// the whole path -- headers out, document back, parsed result -- without a key
// and without the network.
import { createServer, type IncomingMessage, type Server } from "node:http";
import type { AddressInfo } from "node:net";

/** What one starter request looked like on the wire. */
export interface ReceivedRequest {
  readonly url: string;
  readonly authorization: string | undefined;
  readonly environment: string | undefined;
  readonly accept: string | undefined;
}

export interface MockServer {
  readonly baseUrl: string;
  readonly received: readonly ReceivedRequest[];
  stop(): Promise<void>;
}

/**
 * A two-operation descriptor in the exact shape the API serves: an `# ` title,
 * a declared operation count, and one line per operation. The middle dot is
 * part of the format.
 */
export const SAMPLE_DESCRIPTOR = `# Example Product

> Generated for composition cmp_example from its capability closure.

## Integration

- Auth: bearer (secret via HYPERSCALE_API_KEY)
- Error envelope: { error: { code, message }, requestId }

## Operations (2)

- GET /v1/accounts · Account list (account_list; idempotency none)
- POST /v1/accounts · Account create (account_create; idempotency required)

## Golden paths (1)

- Open an account (open_account):
  1. accountCreate (account_create)
`;

export async function startMockServer(options: {
  readonly apiKey: string;
  readonly document?: string;
}): Promise<MockServer> {
  const received: ReceivedRequest[] = [];

  const server = createServer((request, response) => {
    received.push(describe(request));

    if (request.headers.authorization !== `Bearer ${options.apiKey}`) {
      response.writeHead(401, { "content-type": "application/json" });
      response.end(
        JSON.stringify({
          error: {
            code: "invalid_credentials",
            message: "Bearer token is not a valid API key in this environment.",
          },
          requestId: "req_mock",
        }),
      );
      return;
    }

    response.writeHead(200, { "content-type": "text/plain; charset=utf-8" });
    response.end(options.document ?? SAMPLE_DESCRIPTOR);
  });

  // Port 0 lets the kernel pick a free port, so two suites never collide.
  await new Promise<void>((resolve) => {
    server.listen(0, "127.0.0.1", resolve);
  });

  return {
    baseUrl: `http://127.0.0.1:${port(server)}`,
    received,
    stop: () =>
      new Promise<void>((resolve, reject) => {
        server.close((error) => (error ? reject(error) : resolve()));
      }),
  };
}

function describe(request: IncomingMessage): ReceivedRequest {
  return {
    url: request.url ?? "",
    authorization: request.headers.authorization,
    environment: header(request, "x-hyperscale-environment"),
    accept: header(request, "accept"),
  };
}

/** Node lowercases header names but a repeated header arrives as an array. */
function header(request: IncomingMessage, name: string): string | undefined {
  const value = request.headers[name];
  return Array.isArray(value) ? value.join(", ") : value;
}

function port(server: Server): number {
  const address = server.address();
  if (address === null || typeof address === "string") {
    throw new Error("the mock server is not listening on a TCP port");
  }
  return (address as AddressInfo).port;
}
