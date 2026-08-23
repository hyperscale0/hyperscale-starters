// The whole starter: read three environment variables, make one call, print
// what came back. Run it with `node src/main.ts` (Node 22.18 or newer strips
// the types itself, so there is no build step and no dependency).
import {
  fetchProductDescriptor,
  ProductApiError,
  readConfig,
} from "./client.ts";

/** Enough to see the shape without scrolling; the rest is a count. */
const PREVIEW = 10;

async function main(): Promise<void> {
  const config = readConfig(process.env);
  const product = await fetchProductDescriptor(config);

  console.log(`${product.title} (${config.environment})`);
  console.log(`${product.operations.length} operations`);
  for (const operation of product.operations.slice(0, PREVIEW)) {
    console.log(
      `  ${operation.method.padEnd(6)} ${operation.path}  ${operation.operationId}`,
    );
  }
  const rest = product.operations.length - PREVIEW;
  if (rest > 0) console.log(`  and ${rest} more`);
}

try {
  await main();
} catch (error) {
  if (error instanceof ProductApiError) {
    console.error(error.message);
    process.exit(1);
  }
  throw error;
}
