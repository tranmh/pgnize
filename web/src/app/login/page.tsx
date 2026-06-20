"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { ApiError, login } from "@/lib/api-client";
import { useAuth } from "@/components/AuthProvider";
import { useT } from "@/i18n/I18nProvider";

export default function LoginPage() {
  const router = useRouter();
  const { setUser } = useAuth();
  const t = useT();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setBusy(true);
    try {
      const { user } = await login({ email, password });
      setUser(user);
      router.push("/library");
    } catch (err) {
      setError(
        err instanceof ApiError && err.status === 401
          ? t("login.errInvalid")
          : err instanceof Error
            ? err.message
            : t("login.errGeneric"),
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-sm">
      <h1 className="text-2xl font-bold">{t("login.title")}</h1>
      <form onSubmit={submit} className="mt-6 flex flex-col gap-4">
        <label className="flex flex-col gap-1 text-sm">
          {t("common.email")}
          <input
            type="email"
            required
            autoComplete="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="rounded border border-gray-300 px-2 py-2 focus:border-blue-400 focus:outline-none focus:ring-1 focus:ring-blue-300"
          />
        </label>
        <label className="flex flex-col gap-1 text-sm">
          {t("common.password")}
          <input
            type="password"
            required
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="rounded border border-gray-300 px-2 py-2 focus:border-blue-400 focus:outline-none focus:ring-1 focus:ring-blue-300"
          />
        </label>
        {error && <p className="text-sm text-red-600">{error}</p>}
        <button
          type="submit"
          disabled={busy}
          className="rounded bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:bg-gray-300"
        >
          {busy ? t("login.submitting") : t("login.submit")}
        </button>
      </form>
      <p className="mt-4 text-sm text-gray-500">
        {t("login.noAccount")}{" "}
        <Link href="/register" className="text-blue-600 underline">
          {t("nav.register")}
        </Link>
      </p>
    </div>
  );
}
