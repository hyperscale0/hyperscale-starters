// Read three environment variables, make one call, print the result. Run with
// `node src/main.ts`: Node 22.18+ strips the types, so there is no build step.
import { listOperations, ProductApiError, readConfig } from "./client.ts";

async function main(): Promise<void> {
  const config = readConfig(process.env);
  const page = await listOperations(config);

  console.log(`Operations (${config.environment})`);
  if (page.operations.length === 0) {
    console.log("  none yet; actions you run with this key show up here");
  }
  for (const operation of page.operations) {
    console.log(
      `  ${operation.status.padEnd(9)} ${operation.name}  ${operation.operationId}`,
    );
  }
  if (page.hasMore) console.log("  and more on the next page");
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
