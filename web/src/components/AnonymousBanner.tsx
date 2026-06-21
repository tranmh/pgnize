"use client";

import Link from "next/link";
import { useT } from "@/i18n/I18nProvider";

// Notice shown above the anonymous convert/scan flows: results are not saved to
// a library unless the user has an account.
export default function AnonymousBanner() {
  const t = useT();
  return (
    <div className="rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-800">
      {t("anon.prefix")} <strong>{t("anon.notSaved")}</strong>{" "}
      {t("anon.middle")}{" "}
      <Link href="/register" className="font-medium underline">
        {t("anon.createAccount")}
      </Link>{" "}
      {t("anon.or")}{" "}
      <Link href="/login" className="font-medium underline">
        {t("anon.login")}
      </Link>
      {t("anon.suffix")}
    </div>
  );
}
