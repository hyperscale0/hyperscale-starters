/**
 * The Product API, called with `fetch` and nothing else.
 *
 * `Authorization: Bearer <key>` presents the Product API key and
 * `X-Hyperscale-Environment` picks the sandbox or live plane. Sandbox and live
 * keys are never interchangeable, so the environment header is half the
 * credential.
 *
 * The key goes to HYPERSCALE_BASE_URL and nowhere else: redirects are
 * refused, as in the Go and Python starters.
 */

/** Override with HYPERSCALE_BASE_URL. */
const DEFAULT_BASE_URL = "https://hyperscale0.ai";

/** Deadline for the whole call, so a stalled host fails instead of hanging. */
const TIMEOUT_MS = 30_000;

/** One page of ten, enough to see the shape. */
export const OPERATIONS_PATH = "/v1/operations?limit=10";

export type Environment = "sandbox" | "live";

export interface Config {
  readonly apiKey: string;
  readonly baseUrl: string;
  readonly environment: Environment;
}

/** One recorded execution: what ran and how it ended. */
export interface Operation {
  readonly operationId: string;
  readonly name: string;
  readonly status: string;
}

export interface OperationPage {
  readonly operations: readonly Operation[];
  readonly hasMore: boolean;
}

/** A failure the person running this can fix, printed without a stack. */
export class ProductApiError extends Error {}

export function readConfig(
  environmentVariables: Record<string, string | undefined>,
): Config {
  const apiKey = environmentVariables.HYPERSCALE_API_KEY;
  if (!apiKey) {
    throw new ProductApiError(
      "No API key. Set HYPERSCALE_API_KEY to a Product API key; README.md says where to mint one.",
    );
  }

  const environment = environmentVariables.HYPERSCALE_ENVIRONMENT ?? "sandbox";
  if (environment !== "sandbox" && environment !== "live") {
    throw new ProductApiError(
      `HYPERSCALE_ENVIRONMENT must be sandbox or live, not ${environment}.`,
    );
  }

  // A trailing slash would make every path double up on one.
  const baseUrl = (
    environmentVariables.HYPERSCALE_BASE_URL ?? DEFAULT_BASE_URL
  ).replace(/\/+$/, "");

  return { apiKey, baseUrl, environment };
}

/**
 * The one read every Product key is allowed, whatever the Product was composed
 * from: the capability that serves it is part of every Product. A fresh
 * Product has run nothing yet, so an empty list with HTTP 200 still proves
 * the key.
 */
export async function listOperations(config: Config): Promise<OperationPage> {
  const url = `${config.baseUrl}${OPERATIONS_PATH}`;

  let response: Response;
  let body: string;
  // The body read is inside the try: a stream that dies mid-body owes the
  // reader a message, not a stack.
  try {
    response = await fetch(url, {
      headers: {
        accept: "application/json",
        authorization: `Bearer ${config.apiKey}`,
        "x-hyperscale-environment": config.environment,
      },
      signal: AbortSignal.timeout(TIMEOUT_MS),
      // Refuse a 3xx rather than trust the HTTP client to strip the key on
      // a cross-origin hop. The generated JavaScript SDK does the same.
      redirect: "error",
    });
    body = await response.text();
  } catch (cause) {
    if (isUnexpectedRedirect(cause)) {
      throw new ProductApiError(
        "GET /v1/operations was redirected, so the key was not sent on. " +
          "Set HYPERSCALE_BASE_URL to the origin the API answers on; " +
          "the usual cause is http:// where it serves https://.",
      );
    }
    const reason = cause instanceof Error ? cause.message : String(cause);
    throw new ProductApiError(`Could not reach ${url}: ${reason}`);
  }

  if (!response.ok) {
    const detail = errorEnvelope(body) ?? body.trim();
    throw new ProductApiError(
      `GET /v1/operations failed: HTTP ${response.status}${detail === "" ? "" : ` - ${detail}`}`,
    );
  }
  return parseOperationPage(body);
}

/**
 * Node reports a refused redirect as `TypeError: fetch failed` wrapping
 * `Error: unexpected redirect`. The strings are undici's, so the match is
 * cosmetic: a reworded error falls through to the generic message.
 */
function isUnexpectedRedirect(cause: unknown): boolean {
  return (
    cause instanceof TypeError &&
    cause.message === "fetch failed" &&
    cause.cause instanceof Error &&
    cause.cause.message === "unexpected redirect"
  );
}

/**
 * Refuses anything that is not an operation list, and any item without an id
 * or a name. A starter that printed blank rows would hide the format moving
 * under it.
 */
export function parseOperationPage(body: string): OperationPage {
  let parsed: unknown;
  try {
    parsed = JSON.parse(body);
  } catch {
    parsed = undefined;
  }
  const page = parsed as { items?: unknown; nextCursor?: unknown } | undefined;
  if (typeof page !== "object" || page === null || !Array.isArray(page.items)) {
    throw new ProductApiError(
      "The response is not an operation list. Check HYPERSCALE_BASE_URL points at the API origin.",
    );
  }

  const operations = page.items.map((item: unknown): Operation => {
    const { operationId, name, status } = (item ?? {}) as {
      operationId?: unknown;
      name?: unknown;
      status?: unknown;
    };
    if (
      typeof operationId !== "string" ||
      operationId === "" ||
      typeof name !== "string" ||
      name === ""
    ) {
      throw new ProductApiError(
        "An operation arrived without an id or a name; this starter is out of date.",
      );
    }
    return {
      operationId,
      name,
      status: typeof status === "string" ? status : "",
    };
  });

  return {
    operations,
    hasMore: typeof page.nextCursor === "string" && page.nextCursor !== "",
  };
}

/** The API's error shape: `{"error":{"code","message"},"requestId"}`. */
function errorEnvelope(body: string): string | undefined {
  let parsed: unknown;
  try {
    parsed = JSON.parse(body);
  } catch {
    // A non-JSON body is already the best message available.
    return undefined;
  }
  if (typeof parsed !== "object" || parsed === null) return undefined;
  const error = (parsed as { error?: unknown }).error;
  if (typeof error !== "object" || error === null) return undefined;
  const { code, message } = error as { code?: unknown; message?: unknown };
  if (typeof code !== "string" || typeof message !== "string") return undefined;
  return `${code}: ${message}`;
}
