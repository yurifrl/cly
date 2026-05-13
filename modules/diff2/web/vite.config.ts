import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// Vite config for the cly diff2 frontend.
// `vite dev` proxies /api/* to the Go server on :54771 so the React app
// runs against a live backend without rebuilding.
// `vite build` outputs a single-page app to dist/ which Go embeds.
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      "/api": "http://127.0.0.1:54771",
    },
  },
  build: {
    outDir: "dist",
    emptyOutDir: true,
  },
});
