import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTs from "eslint-config-next/typescript";

const eslintConfig = defineConfig([
  ...nextVitals,
  ...nextTs,
  {
    // Our effects deliberately call setState while synchronizing with external
    // systems (data fetching on mount, job polling, debounced autocomplete,
    // auth session refresh). react-hooks v6 flags these as errors; downgrade to
    // a warning so they don't block CI but stay visible.
    rules: {
      "react-hooks/set-state-in-effect": "warn",
    },
  },
  // Override default ignores of eslint-config-next.
  globalIgnores([
    // Default ignores of eslint-config-next:
    ".next/**",
    "out/**",
    "build/**",
    "next-env.d.ts",
    // Vendored third-party assets (e.g. the Stockfish WASM engine loader).
    "public/**",
  ]),
]);

export default eslintConfig;
