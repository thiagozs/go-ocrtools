import { defineConfig } from "vite";
import { resolve } from "node:path";
import wails from "@wailsio/runtime/plugins/vite";

// Three pages: settings (index), selection overlay, result.
export default defineConfig({
  server: {
    host: "127.0.0.1",
    port: Number(process.env.WAILS_VITE_PORT) || 9245,
    strictPort: true,
  },
  build: {
    rollupOptions: {
      input: {
        index: resolve(import.meta.dirname, "index.html"),
        overlay: resolve(import.meta.dirname, "overlay.html"),
        result: resolve(import.meta.dirname, "result.html"),
      },
    },
  },
  plugins: [wails("./bindings")],
});
