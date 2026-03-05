"use client";

import { parsePatchFiles, type FileDiffMetadata } from "@pierre/diffs";
import { FileDiff } from "@pierre/diffs/react";
import { useEffect, useMemo, useState } from "react";
import { fetchBranchDiff, type BranchDiffResponse } from "@/lib/api";

interface BranchDiffProps {
  branchName: string;
  revision: string;
}

const DIFF_OPTIONS = {
  diffStyle: "split",
  lineDiffType: "word",
  hunkSeparators: "metadata",
  overflow: "scroll",
  themeType: "system",
} as const;

export function BranchDiff({ branchName, revision }: BranchDiffProps) {
  const [diff, setDiff] = useState<BranchDiffResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let active = true;
    setLoading(true);
    setError(null);
    setDiff(null);

    fetchBranchDiff(branchName)
      .then((result) => {
        if (!active) return;
        setDiff(result);
      })
      .catch((err) => {
        if (!active) return;
        setError(err instanceof Error ? err.message : "Failed to load branch diff");
      })
      .finally(() => {
        if (!active) return;
        setLoading(false);
      });

    return () => {
      active = false;
    };
  }, [branchName, revision]);

  const parsed = useMemo(() => {
    if (!diff?.patch.trim()) {
      return { files: [] as FileDiffMetadata[], parseError: null as string | null };
    }

    try {
      const files = parsePatchFiles(diff.patch).flatMap((group) => group.files);
      return { files, parseError: null as string | null };
    } catch (err) {
      return {
        files: [] as FileDiffMetadata[],
        parseError: err instanceof Error ? err.message : "Failed to parse patch",
      };
    }
  }, [diff]);

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between gap-2">
        <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
          Diff
        </h4>
        {!loading && !error && parsed.files.length > 0 && (
          <span className="text-xs text-muted-foreground">
            {parsed.files.length} file{parsed.files.length !== 1 && "s"}
          </span>
        )}
      </div>

      {loading && (
        <p className="text-sm text-muted-foreground">Loading diff…</p>
      )}

      {!loading && error && (
        <p className="text-sm text-destructive">{error}</p>
      )}

      {!loading && !error && diff && !diff.patch.trim() && (
        <p className="text-sm text-muted-foreground">
          No changes relative to base commit {diff.baseRevision.slice(0, 7)}.
        </p>
      )}

      {!loading && !error && parsed.parseError && diff?.patch && (
        <div className="space-y-2">
          <p className="text-sm text-destructive">
            Could not parse diff. Showing raw patch.
          </p>
          <pre className="max-h-80 overflow-auto rounded-md border bg-muted/30 p-3 text-xs">
            {diff.patch}
          </pre>
        </div>
      )}

      {!loading && !error && !parsed.parseError && parsed.files.length > 0 && (
        <div className="space-y-3">
          {parsed.files.map((file, i) => (
            <div key={`${file.prevName ?? file.name}:${file.name}:${i}`} className="rounded-md border bg-background">
              <FileDiff fileDiff={file} options={DIFF_OPTIONS} className="block" />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
