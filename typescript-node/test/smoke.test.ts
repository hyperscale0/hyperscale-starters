import assert from "node:assert/strict";
import { createServer } from "node:http";
import type { AddressInfo } from "node:net";
import { after, test } from "node:test";

import {
  fetchProductDescriptor,
  parseProductDescriptor,
  ProductApiError,
  readConfig,
} from "../src/client.ts";
import { SAMPLE_DESCRIPTOR, startMockServer } from "./mock-server.ts";

const API_KEY = "sk_sandbox_example";

const server = await startMockServer({ apiKey: API_KEY });
after(() => server.stop());

test("the smoke call sends the key and the environment, and parses the descriptor", async () => {
  const config = readConfig({
    HYPERSCALE_API_KEY: API_KEY,
    HYPERSCALE_BASE_URL: server.baseUrl,
    HYPERSCALE_ENVIRONMENT: "sandbox",
  });

  const product = await fetchProductDescriptor(config);

  assert.equal(product.title, "Example Product");
  assert.deepEqual(product.operations, [
    { method: "GET", path: "/v1/accounts", operationId: "account_list" },
    { method: "POST", path: "/v1/accounts", operationId: "account_create" },
  ]);

  const request = server.received.at(-1);
  assert.equal(request?.url, "/v1/llms.txt");
  assert.equal(request?.authorization, `Bearer ${API_KEY}`);
  assert.equal(request?.environment, "sandbox");
  assert.equal(request?.accept, "text/plain");
});

test("the live plane is addressed by the header, not by a second host", async () => {
  const config = readConfig({
    HYPERSCALE_API_KEY: API_KEY,
    HYPERSCALE_BASE_URL: server.baseUrl,
    HYPERSCALE_ENVIRONMENT: "live",
  });

  await fetchProductDescriptor(config);

  assert.equal(server.received.at(-1)?.environment, "live");
});

test("a refused key surfaces the error code, not a stack", async () => {
  const config = readConfig({
    HYPERSCALE_API_KEY: "sk_sandbox_wrong",
    HYPERSCALE_BASE_URL: server.baseUrl,
  });

  await assert.rejects(
    () => fetchProductDescriptor(config),
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

  await fetchProductDescriptor(config);

  assert.equal(server.received.at(-1)?.url, "/v1/llms.txt");
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

test("a descriptor whose count disagrees with its lines is refused", () => {
  const short = SAMPLE_DESCRIPTOR.replace(
    "## Operations (2)",
    "## Operations (3)",
  );

  assert.throws(() => parseProductDescriptor(short), {
    message: /declares 3 operations but 2 parsed/,
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
    // The destination is live and does record what reaches it, so a log that
    // stays at one below means the request was never made, not that the test
    // looked in the wrong place.
    await fetchProductDescriptor(
      readConfig({
        HYPERSCALE_API_KEY: API_KEY,
        HYPERSCALE_BASE_URL: elsewhere.baseUrl,
      }),
    );
    assert.equal(elsewhere.received.length, 1);

    await assert.rejects(
      fetchProductDescriptor(
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
