"use client";

import Image from "next/image";
import Link from "next/link";
import Board from "@/components/Board";
import ConvertClient from "./convert/ConvertClient";
import ScanClient from "./scan/ScanClient";
import { useT } from "@/i18n/I18nProvider";

// Real, verified PGN of testdata/brenner_tran.jpg (Brenner, Markus – Tran, Minh
// Cuong, 2026). Shown verbatim as the score-sheet → PGN example output. Truncated
// with an ellipsis line so the card stays compact; the full game is 48 plies.
const EXAMPLE_PGN = `[Event "BL B NF: SF Goeppingen II - SF Denkingen III"]
[Date "2026.03.15"]
[White "Brenner, Markus"]
[Black "Tran, Minh Cuong"]
[Result "1/2-1/2"]

1. e4 c6 2. c4 d5 3. cxd5 cxd5 4. exd5 Nf6 5. Qa4+ Bd7
6. Qb3 Qc7 7. Nc3 Na6 8. d4 Rc8 9. Nf3 e6 10. dxe6 Bxe6
11. Bb5+ Nfd7 12. Qd1 Bb4 13. Bd2 O-O 14. O-O Nf6
…  47. Kg2 Be7 48. Qb7 Bf6 1/2-1/2`;

// FEN of testdata/positions/physical/13.jpg, the board photo used below.
const EXAMPLE_FEN = "r3r1k1/1p1n1pp1/p1p4p/3p2q1/1P1P1Pn1/P1NQP1K1/1B1N2P1/R3R3 w - - 0 1";

function SectionTag({ children }: { children: React.ReactNode }) {
  return (
    <span className="inline-flex w-fit items-center rounded-full bg-blue-50 px-3 py-1 text-xs font-medium text-blue-700">
      {children}
    </span>
  );
}

// Small label that sits above the input/output panels in an example.
function PanelLabel({ children }: { children: React.ReactNode }) {
  return (
    <p className="text-xs font-semibold uppercase tracking-wide text-gray-400">
      {children}
    </p>
  );
}

export default function HomeLanding() {
  const t = useT();

  return (
    <div className="flex flex-col gap-16">
      {/* Hero */}
      <section className="flex flex-col items-center gap-5 pt-6 text-center">
        <SectionTag>{t("landing.hero.eyebrow")}</SectionTag>
        <h1 className="max-w-3xl text-balance text-4xl font-bold tracking-tight text-gray-900 sm:text-5xl">
          {t("landing.hero.title")}
        </h1>
        <p className="max-w-2xl text-pretty text-base text-gray-600 sm:text-lg">
          {t("landing.hero.subtitle")}
        </p>
        <div className="mt-2 flex flex-wrap items-center justify-center gap-3">
          <a
            href="#convert"
            className="rounded-lg bg-blue-600 px-5 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-blue-700"
          >
            {t("landing.hero.ctaConvert")}
          </a>
          <a
            href="#scan"
            className="rounded-lg border border-gray-300 bg-white px-5 py-2.5 text-sm font-semibold text-gray-700 hover:bg-gray-50"
          >
            {t("landing.hero.ctaScan")}
          </a>
          <Link
            href="/new"
            className="rounded-lg border border-indigo-300 bg-indigo-50 px-5 py-2.5 text-sm font-semibold text-indigo-700 hover:bg-indigo-100"
          >
            {t("landing.hero.ctaCoach")}
          </Link>
        </div>
        <p className="text-xs text-gray-400">{t("landing.hero.free")}</p>
      </section>

      {/* Feature 1: score sheet -> PGN */}
      <section className="flex flex-col gap-6">
        <div className="flex flex-col gap-2">
          <SectionTag>{t("landing.f1.tag")}</SectionTag>
          <h2 className="text-2xl font-bold text-gray-900">{t("landing.f1.title")}</h2>
          <p className="max-w-2xl text-sm text-gray-600">{t("landing.f1.body")}</p>
        </div>

        <div className="grid items-start gap-4 md:grid-cols-2">
          {/* Input photo */}
          <figure className="flex flex-col gap-2 rounded-xl border border-gray-200 bg-white p-3 shadow-sm">
            <div className="flex items-center justify-between">
              <PanelLabel>{t("landing.input")}</PanelLabel>
              <span className="rounded-full bg-amber-50 px-2 py-0.5 text-[10px] font-medium text-amber-700">
                {t("landing.exampleBadge")}
              </span>
            </div>
            <Image
              src="/examples/scoresheet.jpg"
              alt={t("landing.f1.inputCaption")}
              width={600}
              height={800}
              className="h-auto w-full rounded-lg border border-gray-100 object-contain"
            />
            <figcaption className="text-xs text-gray-500">
              {t("landing.f1.inputCaption")}
            </figcaption>
          </figure>

          {/* Output PGN */}
          <div className="flex flex-col gap-2 rounded-xl border border-gray-200 bg-white p-3 shadow-sm">
            <PanelLabel>{t("landing.output")}</PanelLabel>
            <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 font-mono text-[13px] leading-relaxed text-gray-100">
              {EXAMPLE_PGN}
            </pre>
            <p className="text-xs text-gray-500">{t("landing.f1.outputCaption")}</p>
          </div>
        </div>

        <a
          href="#convert"
          className="text-sm font-medium text-blue-600 hover:text-blue-700"
        >
          {t("landing.f1.cta")}
        </a>
      </section>

      {/* Feature 2: board photo -> position */}
      <section className="flex flex-col gap-6">
        <div className="flex flex-col gap-2">
          <SectionTag>{t("landing.f2.tag")}</SectionTag>
          <h2 className="text-2xl font-bold text-gray-900">{t("landing.f2.title")}</h2>
          <p className="max-w-2xl text-sm text-gray-600">{t("landing.f2.body")}</p>
        </div>

        <div className="grid items-start gap-4 md:grid-cols-2">
          {/* Input photo */}
          <figure className="flex flex-col gap-2 rounded-xl border border-gray-200 bg-white p-3 shadow-sm">
            <div className="flex items-center justify-between">
              <PanelLabel>{t("landing.input")}</PanelLabel>
              <span className="rounded-full bg-amber-50 px-2 py-0.5 text-[10px] font-medium text-amber-700">
                {t("landing.exampleBadge")}
              </span>
            </div>
            <Image
              src="/examples/board.jpg"
              alt={t("landing.f2.inputCaption")}
              width={768}
              height={768}
              className="h-auto w-full rounded-lg border border-gray-100 object-contain"
            />
            <figcaption className="text-xs text-gray-500">
              {t("landing.f2.inputCaption")}
            </figcaption>
          </figure>

          {/* Output position */}
          <div className="flex flex-col gap-2 rounded-xl border border-gray-200 bg-white p-3 shadow-sm">
            <PanelLabel>{t("landing.output")}</PanelLabel>
            <div className="flex justify-center">
              <Board id="example-board" fen={EXAMPLE_FEN} orientation="white" />
            </div>
            <code className="block overflow-x-auto rounded bg-gray-100 px-2 py-1 font-mono text-xs text-gray-700">
              {EXAMPLE_FEN}
            </code>
            <p className="text-xs text-gray-500">{t("landing.f2.outputCaption")}</p>
          </div>
        </div>

        <p className="text-[11px] text-gray-400">{t("landing.boardCredit")}</p>

        <a
          href="#scan"
          className="text-sm font-medium text-blue-600 hover:text-blue-700"
        >
          {t("landing.f2.cta")}
        </a>
      </section>

      {/* How it works */}
      <section className="flex flex-col gap-6">
        <h2 className="text-2xl font-bold text-gray-900">{t("landing.how.title")}</h2>
        <div className="grid gap-4 sm:grid-cols-3">
          {(["step1", "step2", "step3"] as const).map((step) => (
            <div
              key={step}
              className="flex flex-col gap-2 rounded-xl border border-gray-200 bg-white p-5 shadow-sm"
            >
              <h3 className="text-base font-semibold text-gray-900">
                {t(`landing.how.${step}.title`)}
              </h3>
              <p className="text-sm text-gray-600">{t(`landing.how.${step}.body`)}</p>
            </div>
          ))}
        </div>
      </section>

      {/* Live tool: convert */}
      <section id="convert" className="scroll-mt-20 border-t border-gray-200 pt-12">
        <ConvertClient />
      </section>

      {/* Live tool: scan */}
      <section id="scan" className="scroll-mt-20 border-t border-gray-200 pt-12">
        <ScanClient />
      </section>
    </div>
  );
}
