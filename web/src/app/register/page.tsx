"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { ApiError, register } from "@/lib/api-client";
import { useAuth } from "@/components/AuthProvider";
import { useT } from "@/i18n/I18nProvider";

export default function RegisterPage() {
  const router = useRouter();
  const { setUser } = useAuth();
  const t = useT();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    setBusy(true);
    try {
      const { user } = await register({ name, email, password });
      setUser(user);
      router.push("/library");
    } catch (err) {
      setError(
        err instanceof ApiError && err.status === 409
          ? t("register.errConflict")
          : err instanceof Error
            ? err.message
            : t("register.errGeneric"),
      );
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-sm">
      <h1 className="text-2xl font-bold">{t("register.title")}</h1>
      <form onSubmit={submit} className="mt-6 flex flex-col gap-4">
        <label className="flex flex-col gap-1 text-sm">
          {t("common.name")}
          <input
            type="text"
            required
            autoComplete="name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="rounded border border-gray-300 px-2 py-2 focus:border-blue-400 focus:outline-none focus:ring-1 focus:ring-blue-300"
          />
        </label>
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
            minLength={8}
            autoComplete="new-password"
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
          {busy ? t("register.submitting") : t("register.submit")}
        </button>
      </form>
      <p className="mt-4 text-sm text-gray-500">
        {t("register.haveAccount")}{" "}
        <Link href="/login" className="text-blue-600 underline">
          {t("nav.login")}
        </Link>
      </p>
    </div>
  );
}
