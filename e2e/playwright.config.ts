import { defineConfig, devices } from "@playwright/test";

// Servers are started externally (see e2e/README.md):
//   API_BASE  – Go backend (RECOGNIZER=fake), default http://localhost:8080
//   WEB_BASE  – Next.js frontend,             default http://localhost:3000
const API_BASE = process.env.PGNIZE_API_BASE || "http://localhost:8080";
const WEB_BASE = process.env.PGNIZE_WEB_BASE || "http://localhost:3000";

export default defineConfig({
  testDir: "./tests",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 2 : 4,
  reporter: process.env.CI ? "github" : "list",
  timeout: 60_000,
  projects: [
    {
      // No browser: pure REST against the Go API. This is the default project.
      name: "api",
      testDir: "./tests/api",
      use: { baseURL: API_BASE },
    },
    {
      name: "ui",
      testDir: "./tests/ui",
      use: { ...devices["Desktop Chrome"], baseURL: WEB_BASE },
    },
  ],
});
