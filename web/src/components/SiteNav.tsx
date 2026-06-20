"use client";

import Link from "next/link";
import Image from "next/image";
import { useRouter } from "next/navigation";
import { useAuth } from "./AuthProvider";
import { useT } from "@/i18n/I18nProvider";
import LanguageSwitcher from "@/i18n/LanguageSwitcher";

export default function SiteNav() {
  const { user, loading, signOut } = useAuth();
  const router = useRouter();
  const t = useT();

  return (
    <header className="border-b border-gray-200 bg-white">
      <nav className="mx-auto flex max-w-6xl items-center gap-4 px-4 py-3">
        <Link href="/" className="flex items-center gap-2 text-lg font-bold text-gray-900">
          <Image
            src="/logo.svg"
            alt=""
            width={28}
            height={28}
            className="rounded-md"
            priority
          />
          PGNize
        </Link>
        <Link href="/convert" className="text-sm text-gray-600 hover:text-gray-900">
          {t("nav.convert")}
        </Link>
        {user && (
          <Link href="/library" className="text-sm text-gray-600 hover:text-gray-900">
            {t("nav.library")}
          </Link>
        )}

        <div className="ml-auto flex items-center gap-3 text-sm">
          <LanguageSwitcher />
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
                {t("nav.signOut")}
              </button>
            </>
          ) : (
            <>
              <Link href="/login" className="text-gray-600 hover:text-gray-900">
                {t("nav.login")}
              </Link>
              <Link
                href="/register"
                className="rounded bg-blue-600 px-3 py-1 text-white hover:bg-blue-700"
              >
                {t("nav.register")}
              </Link>
            </>
          )}
        </div>
      </nav>
    </header>
  );
}
