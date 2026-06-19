"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useAuth } from "./AuthProvider";

export default function SiteNav() {
  const { user, loading, signOut } = useAuth();
  const router = useRouter();

  return (
    <header className="border-b border-gray-200 bg-white">
      <nav className="mx-auto flex max-w-6xl items-center gap-4 px-4 py-3">
        <Link href="/" className="flex items-center gap-2 text-lg font-bold text-gray-900">
          <span aria-hidden>♟</span> pgnize
        </Link>
        <Link href="/convert" className="text-sm text-gray-600 hover:text-gray-900">
          Convert
        </Link>
        {user && (
          <Link href="/library" className="text-sm text-gray-600 hover:text-gray-900">
            Library
          </Link>
        )}

        <div className="ml-auto flex items-center gap-3 text-sm">
          {loading ? null : user ? (
            <>
              <span className="text-gray-500">{user.name}</span>
              <button
                type="button"
                onClick={async () => {
                  await signOut();
                  router.push("/");
                }}
                className="rounded border border-gray-300 px-3 py-1 text-gray-700 hover:bg-gray-100"
              >
                Sign out
              </button>
            </>
          ) : (
            <>
              <Link href="/login" className="text-gray-600 hover:text-gray-900">
                Log in
              </Link>
              <Link
                href="/register"
                className="rounded bg-blue-600 px-3 py-1 text-white hover:bg-blue-700"
              >
                Register
              </Link>
            </>
          )}
        </div>
      </nav>
    </header>
  );
}
