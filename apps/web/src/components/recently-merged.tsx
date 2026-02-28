"use client";

import { GitCommitHorizontal } from "lucide-react";
import { useRepo } from "@/components/providers/repo-provider";
import type { TrunkCommitResponse } from "@/lib/api";

function formatTimeAgo(dateStr: string): string {
  const seconds = Math.floor(
    (Date.now() - new Date(dateStr).getTime()) / 1000
  );
  if (seconds < 60) return "just now";
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  return `${days}d ago`;
}

function stripPRSuffix(message: string): string {
  return message.replace(/\s*\(#\d+\)\s*$/, "");
}

function TrunkCommitItem({
  commit,
  owner,
  repoName,
}: {
  commit: TrunkCommitResponse;
  owner?: string;
  repoName?: string;
}) {
  const displayMessage = stripPRSuffix(commit.message);

  return (
    <div className="flex items-start gap-2 py-1.5 text-muted-foreground/70">
      <GitCommitHorizontal className="h-3.5 w-3.5 mt-0.5 shrink-0" />
      <div className="flex-1 min-w-0 flex items-baseline gap-1.5">
        <code className="text-[11px] font-mono shrink-0">{commit.sha}</code>
        <span className="text-xs truncate" title={commit.message}>
          {displayMessage}
        </span>
      </div>
      <div className="flex items-center gap-1.5 shrink-0">
        {commit.prNumber && owner && repoName ? (
          <a
            href={`https://github.com/${owner}/${repoName}/pull/${commit.prNumber}`}
            target="_blank"
            rel="noopener noreferrer"
            className="text-[11px] font-mono text-muted-foreground/50 hover:text-foreground transition-colors"
          >
            #{commit.prNumber}
          </a>
        ) : null}
        <span className="text-[11px] text-muted-foreground/40">
          {formatTimeAgo(commit.date)}
        </span>
      </div>
    </div>
  );
}

export function RecentlyMerged() {
  const { recentlyMerged, repo } = useRepo();

  if (!recentlyMerged || recentlyMerged.length === 0) {
    return null;
  }

  return (
    <div className="px-6 pb-4">
      <div className="space-y-0">
        {recentlyMerged.map((commit) => (
          <TrunkCommitItem
            key={commit.sha}
            commit={commit}
            owner={repo?.owner}
            repoName={repo?.repo}
          />
        ))}
      </div>
    </div>
  );
}
