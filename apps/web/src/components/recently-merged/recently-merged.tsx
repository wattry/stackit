"use client";

import { useMemo } from "react";
import { useRepo } from "@/components/providers/repo-provider";
import { groupByTime } from "@/lib/time";
import { CommitItem } from "./commit-items";

export function RecentlyMerged({ compact = false }: { compact?: boolean }) {
  const { recentlyMerged, repo } = useRepo();

  const groups = useMemo(
    () => groupByTime(recentlyMerged ?? []),
    [recentlyMerged]
  );

  if (groups.length === 0) {
    return null;
  }

  return (
    <div
      className={`${compact ? "px-4 pb-2 max-h-36 space-y-1.5" : "px-6 pb-4 max-h-64 space-y-2"} overflow-y-auto`}
    >
      {groups.map((group) => (
        <div key={group.label}>
          <div className={`${compact ? "text-[9px]" : "text-[10px]"} font-medium text-muted-foreground/50 uppercase tracking-wider mb-0.5`}>
            {group.label}
          </div>
          <div>
            {group.items.map((commit) => (
              <CommitItem
                key={commit.sha}
                commit={commit}
                owner={repo?.owner}
                repoName={repo?.repo}
              />
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}
