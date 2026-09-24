// A stand-in Product API on 127.0.0.1: the suite proves the whole path without
// a key or the network.
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
 * A two-item page in the shape `GET /v1/operations` serves, with a cursor that
 * says another page follows.
 */
export const SAMPLE_OPERATIONS = JSON.stringify({
  items: [
    {
      operationId: "ops_sandbox_example01",
      tenantId: "ten_sandbox_example01",
      name: "customer.create",
      status: "succeeded",
      createdAt: "2026-09-25T09:00:00.000Z",
    },
    {
      operationId: "ops_sandbox_example02",
      tenantId: "ten_sandbox_example01",
      name: "account.create",
      status: "failed",
      errorCode: "validation_failed",
      errorMessage: "currency is required",
      createdAt: "2026-09-25T09:01:00.000Z",
    },
  ],
  nextCursor: "cursor_example",
  statusCounts: { failed: 1, succeeded: 1 },
});

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

    response.writeHead(200, { "content-type": "application/json" });
    response.end(options.document ?? SAMPLE_OPERATIONS);
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
