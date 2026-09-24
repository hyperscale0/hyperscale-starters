import assert from "node:assert/strict";
import { createServer } from "node:http";
import type { AddressInfo } from "node:net";
import { after, test } from "node:test";

import {
  listOperations,
  OPERATIONS_PATH,
  parseOperationPage,
  ProductApiError,
  readConfig,
} from "../src/client.ts";
import { startMockServer } from "./mock-server.ts";

const API_KEY = "sk_sandbox_example";

const server = await startMockServer({ apiKey: API_KEY });
after(() => server.stop());

test("the smoke call sends the key and the environment, and parses the operations", async () => {
  const config = readConfig({
    HYPERSCALE_API_KEY: API_KEY,
    HYPERSCALE_BASE_URL: server.baseUrl,
    HYPERSCALE_ENVIRONMENT: "sandbox",
  });

  const page = await listOperations(config);

  assert.deepEqual(page, {
    operations: [
      {
        operationId: "ops_sandbox_example01",
        name: "customer.create",
        status: "succeeded",
      },
      {
        operationId: "ops_sandbox_example02",
        name: "account.create",
        status: "failed",
      },
    ],
    hasMore: true,
  });

  const request = server.received.at(-1);
  assert.equal(request?.url, OPERATIONS_PATH);
  assert.equal(request?.authorization, `Bearer ${API_KEY}`);
  assert.equal(request?.environment, "sandbox");
  assert.equal(request?.accept, "application/json");
});

test("the live plane is addressed by the header, not by a second host", async () => {
  const config = readConfig({
    HYPERSCALE_API_KEY: API_KEY,
    HYPERSCALE_BASE_URL: server.baseUrl,
    HYPERSCALE_ENVIRONMENT: "live",
  });

  await listOperations(config);

  assert.equal(server.received.at(-1)?.environment, "live");
});

test("a refused key surfaces the error code, not a stack", async () => {
  const config = readConfig({
    HYPERSCALE_API_KEY: "sk_sandbox_wrong",
    HYPERSCALE_BASE_URL: server.baseUrl,
  });

  await assert.rejects(
    () => listOperations(config),
    (error: unknown) =>
      error instanceof ProductApiError &&
      error.message.includes("HTTP 401") &&
      error.message.includes("invalid_credentials"),
  );
});

test("a trailing slash on the base URL does not double up the path", async () => {
  const config = readConfig({
    HYPERSCALE_API_KEY: API_KEY,
    HYPERSCALE_BASE_URL: `${server.baseUrl}///`,
  });

  await listOperations(config);

  assert.equal(server.received.at(-1)?.url, OPERATIONS_PATH);
});

test("config defaults to sandbox and refuses anything but the two planes", () => {
  assert.equal(
    readConfig({ HYPERSCALE_API_KEY: API_KEY }).environment,
    "sandbox",
  );
  assert.equal(
    readConfig({ HYPERSCALE_API_KEY: API_KEY }).baseUrl,
    "https://hyperscale0.ai",
  );

  assert.throws(() => readConfig({}), ProductApiError);
  assert.throws(
    () =>
      readConfig({
        HYPERSCALE_API_KEY: API_KEY,
        HYPERSCALE_ENVIRONMENT: "staging",
      }),
    ProductApiError,
  );
});

test("a response that is not an operation list is refused", () => {
  for (const body of [
    "# Example Product",
    '{"error":"none"}',
    '{"items":[{"name":"customer.create"}]}',
  ]) {
    assert.throws(() => parseOperationPage(body), ProductApiError);
  }

  assert.deepEqual(parseOperationPage('{"items":[]}'), {
    operations: [],
    hasMore: false,
  });
});

test("a redirect does not carry the key to another origin", async () => {
  const elsewhere = await startMockServer({ apiKey: API_KEY });
  const redirector = createServer((request, response) => {
    response.writeHead(302, {
      location: `${elsewhere.baseUrl}${request.url ?? ""}`,
    });
    response.end();
  });
  await new Promise<void>((resolve) => {
    redirector.listen(0, "127.0.0.1", resolve);
  });
  const { port } = redirector.address() as AddressInfo;

  try {
    // The destination records what reaches it, so one entry means no hop.
    await listOperations(
      readConfig({
        HYPERSCALE_API_KEY: API_KEY,
        HYPERSCALE_BASE_URL: elsewhere.baseUrl,
      }),
    );
    assert.equal(elsewhere.received.length, 1);

    await assert.rejects(
      listOperations(
        readConfig({
          HYPERSCALE_API_KEY: API_KEY,
          HYPERSCALE_BASE_URL: `http://127.0.0.1:${port}`,
        }),
      ),
      (error: unknown) =>
        error instanceof ProductApiError &&
        error.message.includes("redirected") &&
        error.message.includes("HYPERSCALE_BASE_URL"),
    );
    assert.equal(
      elsewhere.received.length,
      1,
      "the key was carried to another origin",
    );
  } finally {
    await new Promise<void>((resolve) => {
      redirector.close(() => resolve());
    });
    await elsewhere.stop();
  }
});
