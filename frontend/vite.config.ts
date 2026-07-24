import path from "node:path";

import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: { "@": path.resolve(import.meta.dirname, "src") },
  },
  server: {
    port: 5173,
    // Mirrors production, where Caddy serves the built assets at / and proxies
    // /api/* to Go. Same-origin in both places means no CORS anywhere, and the
    // refresh-token cookie keeps SameSite=Strict. See docs/01-architecture.md §9.
    proxy: {
      "/api": { target: "http://localhost:8080" },
    },
  },
});
