"use client";

import Link from "next/link";
import { getJob, type UploadKind } from "@/lib/api-client";
import { useJobPoller } from "@/hooks/useJobPoller";
import Spinner from "@/components/Spinner";
import { useT } from "@/i18n/I18nProvider";

// One recognized item in the account "separate" multi-image flow: polls its job
// and, once done, links to the right review screen (positions go to the editable
// board editor). Used only when a submission produced more than one job — a single
// job auto-redirects from the upload page instead.
export default function UploadJobRow({
  jobId,
  kind,
}: {
  jobId: string;
  kind: UploadKind;
}) {
  const t = useT();
  const poll = useJobPoller(jobId, getJob);

  if (poll.phase === "failed" || poll.phase === "timeout") {
    return (
      <p className="text-sm text-red-600">
        {poll.phase === "timeout"
          ? t("recog.timeout")
          : (poll.error ?? t("recog.failed"))}
      </p>
    );
  }

  if (poll.phase !== "done" || !poll.gameId) {
    return (
      <Spinner
        label={poll.status === "running" ? t("recog.reading") : t("recog.queued")}
      />
    );
  }

  const href =
    kind === "position"
      ? `/scan/review/${poll.gameId}`
      : `/review/${poll.gameId}`;
  return (
    <Link href={href} className="text-sm font-medium text-blue-600 underline">
      {t("upload.reviewLink")}
    </Link>
  );
}
