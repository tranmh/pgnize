import { defineConfig } from "vitest/config";
import { fileURLToPath } from "node:url";

// Unit tests for the pure client-side logic (UCI parsing, evaluation
// normalization, move classification, correction ranking). These need no DOM —
// component/hook behavior is covered by the Playwright e2e suite.
export default defineConfig({
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  test: {
    environment: "node",
    include: ["src/**/*.test.ts"],
  },
});
