import type { Metadata } from "next";
import "./globals.css";
import { AuthProvider } from "@/components/AuthProvider";
import SiteNav from "@/components/SiteNav";

export const metadata: Metadata = {
  title: "pgnize — score sheet to PGN",
  description:
    "Convert photos of handwritten chess score sheets into human-verified PGN.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="h-full antialiased">
      <body className="flex min-h-full flex-col bg-gray-50 text-gray-900">
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
