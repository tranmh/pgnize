import type { MetadataRoute } from "next";

// Served at /manifest.webmanifest. Next automatically injects the
// <link rel="manifest"> tag for this file-based route.
export default function manifest(): MetadataRoute.Manifest {
  return {
    name: "PGNize — score sheet to PGN",
    short_name: "PGNize",
    description:
      "Convert photos of handwritten chess score sheets into human-verified PGN.",
    id: "/",
    start_url: "/",
    scope: "/",
    display: "standalone",
    orientation: "any",
    background_color: "#0b1957",
    theme_color: "#2563eb",
    categories: ["productivity", "utilities", "sports"],
    icons: [
      {
        src: "/icons/icon-192.png",
        sizes: "192x192",
        type: "image/png",
        purpose: "any",
      },
      {
        src: "/icons/icon-512.png",
        sizes: "512x512",
        type: "image/png",
        purpose: "any",
      },
      {
        src: "/icons/maskable-192.png",
        sizes: "192x192",
        type: "image/png",
        purpose: "maskable",
      },
      {
        src: "/icons/maskable-512.png",
        sizes: "512x512",
        type: "image/png",
        purpose: "maskable",
      },
    ],
  };
}
