// Pure list transforms backing MultiImagePicker. Extracted so the add/remove
// behavior can be unit-tested without a DOM (the component test runs e2e).

export function addImage(files: File[], file: File): File[] {
  return [...files, file];
}

export function removeImageAt(files: File[], idx: number): File[] {
  return files.filter((_, i) => i !== idx);
}

// How a multi-picture submission is turned into jobs.
//  - "combine":  all pictures → one job → one combined result
//  - "separate": one job per picture → one result each
export type UploadMode = "combine" | "separate";

export interface SubmitResult {
  jobIds: string[];
  rejected: number; // separate-mode pictures whose upload failed (e.g. rate limit)
}

// submitImages turns the picked files into recognition jobs via `submitOne` (the
// endpoint-specific convert/scan call). Combine mode (or a single picture) sends
// everything as one request. Separate mode sends one request per picture and
// tolerates partial failure: it returns the jobs that succeeded plus a count of
// the ones that did not, and only throws when EVERY picture failed (so the caller
// can map the error, e.g. a 429, to a message).
export async function submitImages(
  files: File[],
  mode: UploadMode,
  submitOne: (images: File[]) => Promise<{ jobId: string }>,
): Promise<SubmitResult> {
  if (mode === "combine" || files.length <= 1) {
    const { jobId } = await submitOne(files);
    return { jobIds: [jobId], rejected: 0 };
  }

  const settled = await Promise.allSettled(files.map((f) => submitOne([f])));
  const jobIds = settled
    .filter(
      (s): s is PromiseFulfilledResult<{ jobId: string }> =>
        s.status === "fulfilled",
    )
    .map((s) => s.value.jobId);

  if (jobIds.length === 0) {
    const firstRej = settled.find(
      (s): s is PromiseRejectedResult => s.status === "rejected",
    );
    throw firstRej ? firstRej.reason : new Error("upload failed");
  }
  return { jobIds, rejected: settled.length - jobIds.length };
}
