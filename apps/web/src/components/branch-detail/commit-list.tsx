"use client";

import type { CommitResponse } from "@/lib/api";

export function CommitList({ commits }: { commits?: CommitResponse[] }) {
  if (!commits || commits.length === 0) return null;

  return (
    <div className="space-y-1">
      <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">
        Commits
      </h4>
      <div className="space-y-0.5">
        {commits.map((commit) => (
          <div key={commit.sha} className="flex items-baseline gap-2 text-sm">
            <code className="text-xs text-muted-foreground font-mono shrink-0">
              {commit.sha}
            </code>
            <span className="truncate">{commit.message}</span>
          </div>
        ))}
      </div>
    </div>
  );
}
