import { defineConfig } from "vite";
import { svelte } from "@sveltejs/vite-plugin-svelte";

// `deno task dev` proxies the API to a running `archive serve`
const api = "http://localhost:8080";

export default defineConfig({
  plugins: [svelte()],
  server: { proxy: { "/items": api, "/search": api } },
});
