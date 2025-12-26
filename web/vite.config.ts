import tailwindcss from "@tailwindcss/vite";
import vue from "@vitejs/plugin-vue";
import path from "node:path";
import unused from "unplugin-unused/vite";
import router from "unplugin-vue-router/vite";
import { defineConfig } from "vite";
import { VitePWA as pwa } from "vite-plugin-pwa";

export default defineConfig({
  base: "./",
  build: {
    emptyOutDir: true,
    outDir: "dist",
  },
  plugins: [
    router(),
    vue(),
    tailwindcss(),
    pwa({
      includeAssets: ["favicon.ico", "fonts/**/*"],
      manifest: {
        background_color: "#000000",
        description: "Keep your Discord presence online",
        display: "standalone",
        icons: [
          {
            sizes: "192x192",
            src: "android-chrome-192x192.png",
            type: "image/png",
          },
          {
            sizes: "512x512",
            src: "android-chrome-512x512.png",
            type: "image/png",
          },
          {
            purpose: "any maskable",
            sizes: "512x512",
            src: "android-chrome-512x512.png",
            type: "image/png",
          },
        ],
        name: "Discord Stay Online",
        short_name: "Stay Online",
        start_url: "/",
        theme_color: "#5865f2",
      },
      registerType: "autoUpdate",
      workbox: {
        globPatterns: ["**/*.{js,css,html,ico,png,svg,woff2}"],
      },
    }),
    unused({
      ignore: {
        dependencies: ["@tailwindcss/vite", "tailwindcss", "tw-animate-css"],
      },
    }),
  ],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
  server: {
    port: 3000,
    proxy: {
      "/api": {
        changeOrigin: true,
        target: "http://localhost:8080",
      },
      "/ws": {
        target: "ws://localhost:8080",
        ws: true,
      },
    },
  },
});
