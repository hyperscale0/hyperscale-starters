import assert from "node:assert/strict";
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
