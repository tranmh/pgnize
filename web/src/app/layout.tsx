import type { Metadata, Viewport } from "next";
import "./globals.css";
import { AuthProvider } from "@/components/AuthProvider";
import SiteNav from "@/components/SiteNav";
import ServiceWorkerRegister from "@/components/ServiceWorkerRegister";

export const metadata: Metadata = {
  applicationName: "PGNize",
  title: "PGNize — score sheet to PGN",
  description:
    "Convert photos of handwritten chess score sheets into human-verified PGN.",
  appleWebApp: {
    capable: true,
    statusBarStyle: "default",
    title: "PGNize",
  },
};

export const viewport: Viewport = {
  themeColor: "#2563eb",
  width: "device-width",
  initialScale: 1,
  viewportFit: "cover",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="h-full antialiased">
      <body className="flex min-h-full flex-col bg-gray-50 text-gray-900">
        <ServiceWorkerRegister />
        <AuthProvider>
          <SiteNav />
          <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-6">
            {children}
          </main>
        </AuthProvider>
      </body>
    </html>
  );
}
