/**
 * The Product API, called with `fetch` and nothing else.
 *
 * Two headers carry everything. `Authorization: Bearer <key>` presents the
 * Product API key, and `X-Hyperscale-Environment` picks which plane of that
 * key is being addressed. Sandbox keys and live keys are never
 * interchangeable, so the header is not a hint: it is half the credential.
 * HTTP header names are case-insensitive; these are the canonical spellings.
 *
 * The key goes to HYPERSCALE_BASE_URL and nowhere else: this starter refuses
 * redirects rather than following them, the same as the Go and Python
 * starters. The endpoint is fixed and read-only, so there is no hop worth
 * taking.
 */

/** The public origin. Override it with HYPERSCALE_BASE_URL. */
const DEFAULT_BASE_URL = "https://hyperscale0.ai";

/**
 * The whole deadline for the call, connect through body. A host that sends
 * headers and then stops streaming would otherwise leave the starter waiting
 * with nothing on screen and no error.
 */
const TIMEOUT_MS = 30_000;

export type Environment = "sandbox" | "live";

export interface Config {
  readonly apiKey: string;
  readonly baseUrl: string;
  readonly environment: Environment;
}

export interface Operation {
  readonly method: string;
  readonly path: string;
  readonly operationId: string;
}

export interface ProductDescriptor {
  readonly title: string;
  readonly operations: readonly Operation[];
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
 * The one call every Product serves, whatever it was composed from: the
 * Product's own machine-readable descriptor. Everything else in the API
 * surface exists because the Product composed the capability behind it.
 */
export async function fetchProductDescriptor(
  config: Config,
): Promise<ProductDescriptor> {
  const url = `${config.baseUrl}/v1/llms.txt`;

  let response: Response;
  let body: string;
  // The body read is inside the try because a stream that dies mid-body is
  // the same class of failure as a connection that never opened, and both owe
  // the reader a message rather than a stack.
  try {
    response = await fetch(url, {
      headers: {
        accept: "text/plain",
        authorization: `Bearer ${config.apiKey}`,
        "x-hyperscale-environment": config.environment,
      },
      signal: AbortSignal.timeout(TIMEOUT_MS),
      // The key belongs to one origin, so a 3xx is refused rather than
      // followed. Node happens to strip Authorization on a cross-origin hop
      // on its own, but a starter is copy-paste source: whoever swaps in
      // another HTTP client inherits the header policy, not this one. Same
      // setting the generated JavaScript SDK uses.
      redirect: "error",
    });
    body = await response.text();
  } catch (cause) {
    if (isUnexpectedRedirect(cause)) {
      throw new ProductApiError(
        "GET /v1/llms.txt was redirected, so the key was not sent on. " +
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
      `GET /v1/llms.txt failed: HTTP ${response.status}${detail === "" ? "" : ` - ${detail}`}`,
    );
  }
  return parseProductDescriptor(body);
}

/**
 * Node reports a refused redirect as `TypeError: fetch failed` wrapping
 * `Error: unexpected redirect`.
 *
 * These strings are undici's, not a standard, so this match is COSMETIC and
 * never load-bearing: `redirect: "error"` refuses the hop whatever the
 * runtime calls it, and a reworded error just falls through to the generic
 * "Could not reach" line below. Losing the specific message is the right
 * direction to fail in, so this needs no upkeep.
 */
function isUnexpectedRedirect(cause: unknown): boolean {
  return (
    cause instanceof TypeError &&
    cause.message === "fetch failed" &&
    cause.cause instanceof Error &&
    cause.cause.message === "unexpected redirect"
  );
}

const titleLine = /^# (.+)$/m;
const declaredCount = /^## Operations \((\d+)\)$/m;
const operationLine = /^- ([A-Z]+) (\S+) · .+ \((\S+); idempotency \S+\)$/gm;

/**
 * The descriptor is text, so this reads it with three anchored patterns rather
 * than guessing. Parsing fewer operations than the document declares means the
 * format moved under us, and that has to fail loudly: a starter that silently
 * printed a short list would look like a Product missing half its surface.
 */
export function parseProductDescriptor(document: string): ProductDescriptor {
  const title = titleLine.exec(document)?.[1]?.trim();
  const declared = declaredCount.exec(document)?.[1];
  if (!title || declared === undefined) {
    throw new ProductApiError(
      "The response is not a product descriptor. Check HYPERSCALE_BASE_URL points at the API origin.",
    );
  }

  const operations: Operation[] = [];
  for (const match of document.matchAll(operationLine)) {
    const [, method, path, operationId] = match;
    if (!method || !path || !operationId) continue;
    operations.push({ method, path, operationId });
  }

  const expected = Number(declared);
  if (operations.length !== expected) {
    throw new ProductApiError(
      `The descriptor declares ${expected} operations but ${operations.length} parsed; this starter is out of date.`,
    );
  }

  return { title, operations };
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
